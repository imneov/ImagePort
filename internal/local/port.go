// Package local defines access to images in an optional Docker Engine.
package local

import (
	"context"
	"github.com/imneov/ImagePort/internal/domain"
	"io"
)

// Port separates daemon metadata, tagging and immutable export from the desktop.
type Port interface {
	Images(context.Context) (domain.LocalImages, error)
	Inspect(context.Context, domain.Engine, string) (domain.LocalImage, error)
	Tag(context.Context, domain.LocalSource, string, string) error
	TagID(context.Context, domain.Engine, string) (string, error)
	Export(context.Context, domain.LocalSource, io.Writer) error
}

// Loader imports a prepared single-image archive and verifies its expected content ID.
type Loader interface {
	Load(context.Context, domain.Engine, io.Reader, string, string, string) (string, error)
}

// Remover removes one local tag without force or pruning related content.
type Remover interface {
	RemoveTag(context.Context, domain.Engine, string, string) error
}
