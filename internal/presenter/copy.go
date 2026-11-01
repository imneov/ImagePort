package presenter

import (
	"errors"

	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/opencontainers/go-digest"
)

// validateCopy binds immutable content to the captured source connection before queueing.
func (presenter *Presenter) validateCopy(spec domain.Spec, target domain.Profile) error {
	source := target
	if spec.SourceProfileID != "" {
		var err error
		source, err = presenter.Store.Profile(spec.SourceProfileID)
		if err != nil {
			return err
		}
		if source.Host != spec.SourceRegistryHost {
			return errors.New("The source registry address changed; select the source again")
		}
	}
	if _, err := adapter.Canonical(source, spec.Source.Reference, false); err != nil {
		return err
	}
	if err := digest.Digest(spec.Source.Digest).Validate(); err != nil {
		return errors.New("Select a valid immutable source manifest")
	}
	return nil
}
