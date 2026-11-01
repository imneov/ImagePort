package presenter

import (
	"context"

	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
)

// TaskProfile gives anonymous operations a durable account ID without storing a secret.
func (presenter *Presenter) TaskProfile(target domain.Target, id string) (domain.Profile, error) {
	profile, err := presenter.AccessProfile(target, id)
	if err != nil || profile.ID != "" {
		return profile, err
	}
	accounts, err := presenter.TargetAccounts(target.ID)
	if err != nil {
		return profile, err
	}
	for _, account := range accounts {
		if account.Username == "" {
			return account, nil
		}
	}
	profile.Name = "Anonymous"
	return presenter.SaveProfile(profile, "")
}

// InspectArchive validates an image archive off the UI thread without extraction.
func (presenter *Presenter) InspectArchive(ctx context.Context, path string) (domain.Archive, error) {
	return archive.Inspect(ctx, path, presenter.Limits)
}
