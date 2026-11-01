package transfer

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/task"
)

// upload freezes local input, preserving the selected entry's complete index.
func (runner *Runner) upload(ctx context.Context, profile domain.Profile, spec domain.Spec, checkpoint task.Checkpoint) (string, error) {
	snapshot, cleanup, err := archive.Snapshot(ctx, spec.Archive.Path)
	if err != nil {
		return "", err
	}
	defer runner.cleanup(cleanup)
	inspection, err := archive.Inspect(ctx, snapshot, runner.Limits)
	if err != nil {
		return "", err
	}
	if inspection.SHA256 != spec.Archive.SHA256 || inspection.Size != spec.Archive.Size {
		return "", errors.New("The archive changed after inspection; select and inspect it again rather than retrying the old task")
	}
	return runner.uploadSnapshot(ctx, profile, spec, snapshot, inspection, checkpoint)
}

// uploadSnapshot publishes one verified private archive using the shared conflict/readback path.
func (runner *Runner) uploadSnapshot(ctx context.Context, profile domain.Profile, spec domain.Spec, snapshot string, inspection domain.Archive, checkpoint task.Checkpoint) (string, error) {
	if err := runner.targetUnchanged(ctx, profile, spec); err != nil {
		return "", err
	}
	if inspection.Format == "docker save tar" {
		return runner.uploadDocker(ctx, profile, spec, snapshot, checkpoint)
	}
	if spec.Entry.Digest == "" {
		return "", errors.New("The selected OCI entry has no digest")
	}
	if err := checkpoint("transfer", spec.Entry.Digest, ""); err != nil {
		return "", err
	}
	writeErr := runner.Registry.Import(ctx, profile, snapshot, spec.Entry, spec.Target)
	if err := checkpoint("verify", "", ""); err != nil {
		return "", &task.Unknown{Reason: err.Error()}
	}
	return runner.remoteResult(profile, spec.Target, spec.Entry.Digest, writeErr)
}

// uploadDocker asks the SDK to normalize first, obtaining the exact target digest.
func (runner *Runner) uploadDocker(ctx context.Context, profile domain.Profile, spec domain.Spec, snapshot string, checkpoint task.Checkpoint) (string, error) {
	if err := checkpoint("resolve", "", ""); err != nil {
		return "", err
	}
	expected, source, cleanup, err := runner.Registry.Normalize(ctx, snapshot, spec.Entry)
	if err != nil {
		return "", err
	}
	defer runner.cleanup(cleanup)
	if err := runner.targetUnchanged(ctx, profile, spec); err != nil {
		return "", err
	}
	if err := checkpoint("transfer", expected, ""); err != nil {
		return "", err
	}
	writeErr := runner.Registry.Copy(ctx, profile, source, spec.Target)
	if err := checkpoint("verify", "", ""); err != nil {
		return "", &task.Unknown{Reason: err.Error()}
	}
	return runner.remoteResult(profile, spec.Target, expected, writeErr)
}

// cleanup reports temporary disk cleanup failure without exposing source filenames.
func (runner *Runner) cleanup(remove func() error) {
	if err := remove(); err != nil && !errors.Is(err, os.ErrNotExist) {
		logger := runner.Logger
		if logger == nil {
			logger = slog.Default()
		}
		logger.Warn("import temporary files require cleanup")
	}
}
