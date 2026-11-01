package presenter

import (
	"context"
	"strings"

	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/domain"
)

// browseReference validates direct access without requiring permission to enumerate repositories.
func (presenter *Presenter) browseReference(ctx context.Context, profile domain.Profile, input string) (domain.LocationPage, error) {
	canonical, err := adapter.Canonical(profile, input, false)
	if err != nil {
		return domain.LocationPage{}, err
	}
	if _, err := presenter.Registry.Resolve(ctx, profile, canonical); err != nil {
		return domain.LocationPage{}, err
	}
	repository, version, tagged := strings.Cut(canonical, ":")
	// A hostname can include a port; only a separator in the final path component is a tag.
	last := strings.LastIndex(canonical, "/")
	if separator := strings.LastIndex(canonical, ":"); separator > last && !strings.Contains(canonical, "@") {
		repository, version, tagged = canonical[:separator], canonical[separator+1:], true
	}
	if !tagged || strings.Contains(canonical, "@") {
		repository = ""
		version = canonical
	}
	return domain.LocationPage{Entries: []domain.LocationEntry{{ID: "version:" + canonical, Name: version, Path: canonical, Repository: repository, Kind: "version"}}}, nil
}
