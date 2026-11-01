package transfer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/task"
)

// download verifies a same-directory partial before publishing without overwriting.
func (runner *Runner) download(ctx context.Context, profile domain.Profile, spec domain.Spec, checkpoint task.Checkpoint) (result string, resultErr error) {
	if spec.Source.Digest == "" || spec.Source.Platform == "" {
		return "", errors.New("Select a platform explicitly before continuing")
	}
	if _, err := os.Lstat(spec.Output); err == nil {
		return "", errors.New("The destination file already exists; choose another filename")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	file, err := os.CreateTemp(filepath.Dir(spec.Output), ".imageport-*.partial")
	if err != nil {
		return "", err
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
			runner.Logger.Warn("partial cleanup failed")
		}
	}()
	if err := checkpoint("transfer", spec.Source.Digest, file.Name()); err != nil {
		return "", errors.Join(err, file.Close())
	}
	exportErr := runner.Registry.Export(ctx, profile, spec.Source, file)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(exportErr, syncErr, closeErr); err != nil {
		return "", err
	}
	if err := checkpoint("verify", "", ""); err != nil {
		return "", err
	}
	inspection, err := archive.Inspect(ctx, file.Name(), runner.Limits)
	if err != nil {
		return "", err
	}
	if !hasDigest(inspection, spec.Source.Digest) {
		return "", errors.New("The downloaded archive root differs from the selected manifest digest")
	}
	if err := checkpoint("commit", "", spec.Output+"\nSHA256: "+inspection.SHA256); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := archive.Publish(file.Name(), spec.Output); err != nil {
		return "", &task.Unknown{Reason: fmt.Sprintf("File publication could not be confirmed; verify the destination before retrying: %v", err)}
	}
	return spec.Output + "\n" + spec.Source.Platform + "\n" + spec.Source.Digest + "\nFile SHA256: " + inspection.SHA256, nil
}

// hasDigest compares root manifest digests, not the tar file checksum.
func hasDigest(inspection domain.Archive, expected string) bool {
	for _, entry := range inspection.Entries {
		if entry.Digest == expected {
			return true
		}
	}
	return false
}

// checkDownload identifies a previously committed file without replacing it.
func (runner *Runner) checkDownload(ctx context.Context, spec domain.Spec) (bool, error) {
	if _, err := os.Lstat(spec.Output); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	inspection, err := archive.Inspect(ctx, spec.Output, runner.Limits)
	if err != nil {
		return false, err
	}
	if !hasDigest(inspection, spec.Source.Digest) {
		return false, errors.New("The destination does not match the expected image; resolve the filename conflict first")
	}
	return true, nil
}
