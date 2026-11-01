package transfer

import (
	"context"
	"errors"
	"os"

	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/task"
)

// push snapshots the exact daemon ID and uses the verified registry upload pipeline.
func (runner *Runner) push(ctx context.Context, profile domain.Profile, spec domain.Spec, checkpoint task.Checkpoint) (string, error) {
	if runner.Local == nil {
		return "", errors.New("Local Docker image support is unavailable")
	}
	if err := runner.targetUnchanged(ctx, profile, spec); err != nil {
		return "", err
	}
	if err := checkpoint("export", "", ""); err != nil {
		return "", err
	}
	file, err := os.CreateTemp("", "imageport-docker-*.tar")
	if err != nil {
		return "", err
	}
	defer runner.cleanup(func() error { return os.Remove(file.Name()) })
	exportErr := runner.Local.Export(ctx, spec.Local, file)
	if err := errors.Join(exportErr, file.Close()); err != nil {
		return "", err
	}
	inspection, err := archive.Inspect(ctx, file.Name(), runner.Limits)
	if err != nil {
		return "", err
	}
	if len(inspection.Entries) != 1 {
		return "", errors.New("Docker export did not contain exactly one image root; export and inspect the archive explicitly")
	}
	spec.Archive, spec.Entry = inspection, inspection.Entries[0]
	return runner.uploadSnapshot(ctx, profile, spec, file.Name(), inspection, checkpoint)
}
