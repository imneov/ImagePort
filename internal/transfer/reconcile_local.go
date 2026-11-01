package transfer

import (
	"context"
	"errors"
	"os"

	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
)

// reconcileLocal checks observed local/file/deletion results without repeating a write.
func (runner *Runner) reconcileLocal(ctx context.Context, item domain.Task) (bool, error) {
	spec := item.Spec
	switch spec.Operation {
	case "load", "pull", "copy-local":
		actual, err := runner.Local.TagID(ctx, spec.DestinationEngine, spec.Target)
		return actual != "" && actual == item.Current().ExpectedDigest, err
	case "delete-local":
		actual, err := runner.Local.TagID(ctx, spec.Local.Engine, spec.Target)
		return actual == "", err
	case "delete-remote":
		profile, err := runner.Profiles.Profile(spec.ProfileID)
		if err != nil {
			return false, err
		}
		if profile.Host != spec.RegistryHost {
			return false, errors.New("The registry address changed; restore the original connection before verification")
		}
		_, err = runner.Registry.Digest(ctx, profile, spec.Target)
		if registry.IsCode(err, "NOT_FOUND") {
			return true, nil
		}
		return false, err
	case "delete-file":
		_, err := os.Lstat(spec.Archive.Path)
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	case "export-local":
		if item.Current().ExpectedDigest == "" {
			return false, nil
		}
		inspection, err := archive.Inspect(ctx, spec.Output, runner.Limits)
		return inspection.SHA256 == item.Current().ExpectedDigest, err
	}
	return false, nil
}
