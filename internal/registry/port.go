// Package registry defines the application-owned registry boundary.
package registry

import (
	"context"
	"io"

	"github.com/imneov/ImagePort/internal/domain"
)

// Port isolates SDK operations; every network call obeys the supplied context.
type Port interface {
	Check(context.Context, domain.Profile) error
	Repositories(context.Context, domain.Profile, string, int) (domain.Page, error)
	Tags(context.Context, domain.Profile, string, string, int) (domain.Page, error)
	Resolve(context.Context, domain.Profile, string) ([]domain.Source, error)
	Digest(context.Context, domain.Profile, string) (string, error)
	Export(context.Context, domain.Profile, domain.Source, io.Writer) error
	Copy(context.Context, domain.Profile, domain.Source, string) error
	Import(context.Context, domain.Profile, string, domain.ArchiveEntry, string) error
	Normalize(context.Context, string, domain.ArchiveEntry) (string, domain.Source, func() error, error)
}

// ProjectBrowser exposes a registry product's project-scoped browsing API.
type ProjectBrowser interface {
	Projects(context.Context, domain.Profile, string, int) (domain.Page, error)
	ProjectRepositories(context.Context, domain.Profile, string, string, int) (domain.Page, error)
}

// CrossRegistryCopier transfers a pinned image using independent saved connections.
type CrossRegistryCopier interface {
	CopyBetween(context.Context, domain.Profile, domain.Profile, domain.Source, string) error
}

// ArchiveExporter prepares only the chosen entry/platform with an explicit local tag.
type ArchiveExporter interface {
	ExportEntry(context.Context, string, domain.ArchiveEntry, string, string, io.Writer) (string, error)
}

// TagRemover removes exactly one tag after checking its previously observed digest.
type TagRemover interface {
	DeleteTag(context.Context, domain.Profile, string, string) error
}
