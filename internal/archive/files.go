package archive

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Publish links a verified same-filesystem partial into place without replacement.
// A hard link is atomic and fails if another process created the final path.
func Publish(partial, final string) error {
	if err := os.Link(partial, final); err != nil {
		return err
	}
	return os.Remove(partial)
}

// Snapshot makes a private import source so user edits cannot change queued input.
// The caller owns cleanup; a full copy and optional normalization need extra space.
func Snapshot(ctx context.Context, source string) (string, func() error, error) {
	input, err := os.Open(source)
	if err != nil {
		return "", nil, err
	}
	output, err := os.CreateTemp("", "imageport-import-*.tar")
	if err != nil {
		return "", nil, errors.Join(err, input.Close())
	}
	_, copyErr := io.Copy(output, contextReader{ctx, input})
	closeErr := errors.Join(input.Close(), output.Close())
	cleanup := func() error { return os.Remove(output.Name()) }
	if err := errors.Join(copyErr, closeErr); err != nil {
		return "", nil, errors.Join(err, cleanup())
	}
	return output.Name(), cleanup, nil
}

// Destination canonicalizes the output directory for process-local target locking.
func Destination(filename string) (string, error) {
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	directory, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, filepath.Base(absolute)), nil
}
