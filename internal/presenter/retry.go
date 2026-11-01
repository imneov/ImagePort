package presenter

import (
	"context"
	"errors"
	"strings"

	"github.com/imneov/ImagePort/internal/domain"
)

// RecheckTaskTarget observes the original destination; deletion never follows replacement content.
func (presenter *Presenter) RecheckTaskTarget(ctx context.Context, item domain.Task) (string, error) {
	if item.State != domain.Failed && item.State != domain.Cancelled {
		return "", errors.New("Only failed or cancelled tasks can be retried; verify unknown outcomes first")
	}
	spec := item.Spec
	if spec.Output != "" || spec.Operation == "delete-file" {
		return spec.PreviousDigest, nil
	}
	if spec.DestinationEngine.Identity != "" {
		return presenter.Local.TagID(ctx, spec.DestinationEngine, spec.Target)
	}
	if spec.Operation == "delete-local" {
		actual, err := presenter.Local.TagID(ctx, spec.Local.Engine, spec.Target)
		if err == nil && actual != spec.Local.ImageID {
			err = errors.New("The local image changed after confirmation; refresh and select it again")
		}
		return spec.PreviousDigest, err
	}
	profile, err := presenter.Store.Profile(spec.ProfileID)
	if err != nil {
		return "", err
	}
	if profile.Host != spec.RegistryHost {
		return "", errors.New("The connection address changed; check the target again")
	}
	_, previous, err := presenter.Target(ctx, profile, spec.Target)
	if err == nil && strings.HasPrefix(spec.Operation, "delete-") && previous != spec.PreviousDigest {
		err = errors.New("The remote version changed after confirmation; refresh and select it again")
	}
	return previous, err
}
