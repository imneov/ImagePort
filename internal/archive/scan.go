// Package archive inspects image archives without extracting untrusted paths.
package archive

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	_ "crypto/sha512" // Enable SHA-384/512 OCI descriptors.
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/opencontainers/go-digest"
)

// Limits bounds archive resource use; callers may choose stricter deployment limits.
type Limits struct {
	Entries                               int
	EntryBytes, TotalBytes, MetadataBytes int64
}

// DefaultLimits returns documented safety limits, not a maximum supported image size.
func DefaultLimits() Limits { return Limits{200000, 1 << 40, 4 << 40, 32 << 20} }

type fileInfo struct {
	size   int64
	digest string
	json   []byte
}
type scan struct {
	ctx        context.Context
	graphNodes int
	resolved   map[string][]string
	files      map[string]fileInfo
	metadata   int64
	limits     Limits
}

// contextReader aborts file work promptly when the task context is cancelled.
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

// Read returns the underlying bytes unless cancellation was requested.
func (reader contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}

// SafePath rejects paths whose interpretation differs across supported platforms.
func SafePath(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") {
		return "", errors.New("The archive contains an absolute or cross-platform unsafe path")
	}
	for _, component := range strings.Split(name, "/") {
		if component == ".." {
			return "", errors.New("The archive contains path traversal")
		}
	}
	cleaned := path.Clean(name)
	if cleaned == "." {
		return "", nil
	}
	return cleaned, nil
}

// Inspect validates the complete regular tar, blob hashes, and image entry graphs.
func Inspect(ctx context.Context, filename string, limits Limits) (domain.Archive, error) {
	identity, err := os.Lstat(filename)
	if err != nil {
		return domain.Archive{}, err
	}
	if !identity.Mode().IsRegular() {
		return domain.Archive{}, errors.New("Choose a regular image tar file; links and directories are not supported")
	}
	file, err := os.Open(filename)
	if err != nil {
		return domain.Archive{}, err
	}
	result, inspectErr := inspect(ctx, file, limits)
	closeErr := file.Close()
	if err := errors.Join(inspectErr, closeErr); err != nil {
		return domain.Archive{}, err
	}
	result.Path = filename
	return result, nil
}

// inspect reads through EOF so the file checksum includes trailing tar padding.
func inspect(ctx context.Context, input io.Reader, limits Limits) (domain.Archive, error) {
	if limits.Entries <= 0 || limits.MetadataBytes <= 0 {
		return domain.Archive{}, errors.New("Invalid archive resource limit configuration")
	}
	checksum := sha256.New()
	reader := io.TeeReader(contextReader{ctx, input}, checksum)
	counted := &countReader{Reader: reader}
	scanner := &scan{ctx: ctx, resolved: make(map[string][]string), files: make(map[string]fileInfo), limits: limits}
	if err := scanner.read(tar.NewReader(counted)); err != nil {
		return domain.Archive{}, err
	}
	if _, err := io.Copy(io.Discard, counted); err != nil {
		return domain.Archive{}, err
	}
	result, err := scanner.entries()
	if err != nil {
		return domain.Archive{}, err
	}
	result.Size, result.SHA256 = counted.total, hex.EncodeToString(checksum.Sum(nil))
	return result, nil
}

type countReader struct {
	io.Reader
	total int64
}

// Read records actual file bytes, not network or upload progress.
func (reader *countReader) Read(buffer []byte) (int, error) {
	count, err := reader.Reader.Read(buffer)
	reader.total += int64(count)
	return count, err
}

// read bounds entries and expanded byte sizes before reading each tar member.
func (scanner *scan) read(reader *tar.Reader) error {
	var total int64
	for index := 0; ; index++ {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("The archive is corrupt or truncated: %w", err)
		}
		if index >= scanner.limits.Entries {
			return errors.New("The archive exceeds the entry count safety limit")
		}
		name, err := SafePath(header.Name)
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return errors.New("Image archives must not contain links, devices, or special members")
		}
		if _, exists := scanner.files[name]; exists || name == "" {
			return errors.New("The archive contains duplicate or invalid file paths")
		}
		if header.Size < 0 || header.Size > scanner.limits.EntryBytes || header.Size > scanner.limits.TotalBytes-total {
			return errors.New("The archive exceeds the content size safety limit")
		}
		total += header.Size
		if err := scanner.member(reader, name, header.Size); err != nil {
			return err
		}
	}
}

// member hashes content incrementally; only bounded JSON metadata stays in memory.
func (scanner *scan) member(reader io.Reader, name string, size int64) error {
	algorithm := digest.SHA256
	parts := strings.Split(name, "/")
	if len(parts) == 3 && parts[0] == "blobs" {
		algorithm = digest.Algorithm(parts[1])
	}
	if !algorithm.Available() {
		return errors.New("The archive uses an unsupported digest algorithm")
	}
	hasher := algorithm.Digester()
	stream := io.TeeReader(reader, hasher.Hash())
	var metadata []byte
	if size <= scanner.limits.MetadataBytes {
		prefix := make([]byte, min(size, 512))
		if _, err := io.ReadFull(stream, prefix); err != nil {
			return err
		}
		trimmed := bytes.TrimSpace(prefix)
		if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
			if size > scanner.limits.MetadataBytes-scanner.metadata {
				return errors.New("The archive exceeds the JSON metadata safety limit")
			}
			remainder, err := io.ReadAll(stream)
			if err != nil {
				return err
			}
			metadata = append(prefix, remainder...)
			scanner.metadata += size
		}
	}
	if _, err := io.Copy(io.Discard, stream); err != nil {
		return err
	}
	actual := hasher.Digest().String()
	if len(parts) == 3 && parts[0] == "blobs" && actual != parts[1]+":"+parts[2] {
		return errors.New("An archive blob does not match its content digest")
	}
	scanner.files[name] = fileInfo{size, actual, metadata}
	return nil
}
