package transfer

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/local"
	"github.com/imneov/ImagePort/internal/registry"
	"github.com/imneov/ImagePort/internal/task"
)

// runLocal dispatches engine/archive operations which do not need a registry account.
func (runner *Runner) runLocal(ctx context.Context, spec domain.Spec, checkpoint task.Checkpoint) (string, error) {
	if err := checkpoint("preflight", "", ""); err != nil {
		return "", err
	}
	switch spec.Operation {
	case "export-local":
		return runner.exportLocal(ctx, spec, checkpoint)
	case "load":
		return runner.loadArchive(ctx, spec, checkpoint)
	case "copy-local":
		return runner.copyLocal(ctx, spec, checkpoint)
	case "delete-local":
		port, valid := runner.Local.(local.Remover)
		if !valid {
			return "", errors.New("Local image removal is unavailable")
		}
		err := port.RemoveTag(ctx, spec.Local.Engine, spec.Target, spec.Local.ImageID)
		var unknown *local.UnknownResult
		if errors.As(err, &unknown) {
			return "", &task.Unknown{Reason: unknown.Error()}
		}
		return spec.Target, err
	case "delete-file":
		return runner.deleteFile(ctx, spec)
	default:
		return "", errors.New("Unsupported image management operation")
	}
}

// copyLocal stages a pinned daemon image for a different tag or engine without removing its source.
func (runner *Runner) copyLocal(ctx context.Context, spec domain.Spec, checkpoint task.Checkpoint) (string, error) {
	file, err := os.CreateTemp("", "imageport-engine-copy-*.tar")
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
	spec.Entry = inspection.Entries[0]
	return runner.loadSnapshot(ctx, spec, file.Name(), checkpoint)
}

// exportLocal verifies an immutable daemon export before publishing without overwriting.
func (runner *Runner) exportLocal(ctx context.Context, spec domain.Spec, checkpoint task.Checkpoint) (string, error) {
	file, err := os.CreateTemp(filepath.Dir(spec.Output), ".imageport-*.partial")
	if err != nil {
		return "", err
	}
	defer runner.cleanup(func() error { return os.Remove(file.Name()) })
	if err := checkpoint("export", "", file.Name()); err != nil {
		return "", errors.Join(err, file.Close())
	}
	exportErr := runner.Local.Export(ctx, spec.Local, file)
	if err := errors.Join(exportErr, file.Sync(), file.Close()); err != nil {
		return "", err
	}
	inspection, err := archive.Inspect(ctx, file.Name(), runner.Limits)
	if err != nil {
		return "", err
	}
	if err := checkpoint("commit", inspection.SHA256, spec.Output); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := archive.Publish(file.Name(), spec.Output); err != nil {
		return "", &task.Unknown{Reason: "Archive publication could not be verified; check the destination before retrying"}
	}
	return spec.Output + "\nSHA256: " + inspection.SHA256, nil
}

// loadArchive snapshots checked input and selects exactly one entry before loading it.
func (runner *Runner) loadArchive(ctx context.Context, spec domain.Spec, checkpoint task.Checkpoint) (string, error) {
	snapshot, cleanup, err := archive.Snapshot(ctx, spec.Archive.Path)
	if err != nil {
		return "", err
	}
	defer runner.cleanup(cleanup)
	inspection, err := archive.Inspect(ctx, snapshot, runner.Limits)
	if err != nil {
		return "", err
	}
	if inspection.SHA256 != spec.Archive.SHA256 {
		return "", errors.New("The archive changed after inspection; select and inspect it again rather than retrying the old task")
	}
	return runner.loadSnapshot(ctx, spec, snapshot, checkpoint)
}

// loadSnapshot normalizes a single chosen entry with one explicit target alias.
func (runner *Runner) loadSnapshot(ctx context.Context, spec domain.Spec, snapshot string, checkpoint task.Checkpoint) (string, error) {
	loader, valid := runner.Local.(local.Loader)
	exporter, exports := runner.Registry.(registry.ArchiveExporter)
	if !valid || !exports {
		return "", errors.New("Local image import is unavailable")
	}
	file, err := os.CreateTemp("", "imageport-load-*.tar")
	if err != nil {
		return "", err
	}
	defer runner.cleanup(func() error { return os.Remove(file.Name()) })
	expected, exportErr := exporter.ExportEntry(ctx, snapshot, spec.Entry, spec.Source.Platform, spec.Target, file)
	if err := errors.Join(exportErr, file.Close()); err != nil {
		return "", err
	}
	if err := checkpoint("transfer", expected, ""); err != nil {
		return "", err
	}
	input, err := os.Open(file.Name())
	if err != nil {
		return "", err
	}
	result, loadErr := loader.Load(ctx, spec.DestinationEngine, input, spec.Target, spec.PreviousDigest, expected)
	if err := errors.Join(loadErr, input.Close()); err != nil {
		return "", &task.Unknown{Reason: "The local load did not complete with a verified result; inspect the destination before retrying" + "; " + err.Error()}
	}
	return result, nil
}

// pull exports the selected remote manifest into a private archive, then loads its chosen platform.
func (runner *Runner) pull(ctx context.Context, profile domain.Profile, spec domain.Spec, checkpoint task.Checkpoint) (string, error) {
	file, err := os.CreateTemp("", "imageport-pull-*.tar")
	if err != nil {
		return "", err
	}
	defer runner.cleanup(func() error { return os.Remove(file.Name()) })
	exportErr := runner.Registry.Export(ctx, profile, spec.Source, file)
	if err := errors.Join(exportErr, file.Close()); err != nil {
		return "", err
	}
	inspection, err := archive.Inspect(ctx, file.Name(), runner.Limits)
	if err != nil {
		return "", err
	}
	if len(inspection.Entries) != 1 || inspection.Entries[0].Digest != spec.Source.Digest {
		return "", errors.New("The downloaded archive root differs from the selected manifest digest")
	}
	spec.Entry = inspection.Entries[0]
	return runner.loadSnapshot(ctx, spec, file.Name(), checkpoint)
}

// deleteFile revalidates the complete selected archive before removing that file only.
func (runner *Runner) deleteFile(ctx context.Context, spec domain.Spec) (string, error) {
	before, err := os.Lstat(spec.Archive.Path)
	if err != nil {
		return "", err
	}
	inspection, err := archive.Inspect(ctx, spec.Archive.Path, runner.Limits)
	if err != nil {
		return "", err
	}
	if inspection.SHA256 != spec.Archive.SHA256 {
		return "", errors.New("The archive changed after inspection; select and inspect it again rather than retrying the old task")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	after, err := os.Lstat(spec.Archive.Path)
	if err != nil {
		return "", err
	}
	if !os.SameFile(before, after) || !after.Mode().IsRegular() || after.Size() != inspection.Size || !after.ModTime().Equal(before.ModTime()) {
		return "", errors.New("The archive changed after inspection; select and inspect it again rather than retrying the old task")
	}
	return spec.Archive.Path, os.Remove(spec.Archive.Path)
}
