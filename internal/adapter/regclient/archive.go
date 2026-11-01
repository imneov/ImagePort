package regclient

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
	sdk "github.com/regclient/regclient"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
)

// ExportEntry isolates one chosen image in a private OCI layout, returning its config ID.
func (client *Client) ExportEntry(ctx context.Context, path string, entry domain.ArchiveEntry, platform, target string, output io.Writer) (string, error) {
	_, source, cleanup, err := client.Normalize(ctx, path, entry)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := cleanup(); err != nil {
			slog.Warn("temporary archive cleanup failed")
		}
	}()
	connection := sdk.New(sdk.WithSlog(slogDiscard()))
	reference, err := ref.New(source.Reference)
	if err != nil {
		return "", err
	}
	image, err := connection.ManifestGet(ctx, reference)
	if err != nil {
		return "", registry.Safe(err)
	}
	if image.IsList() {
		reference, image, err = localPlatform(ctx, connection, reference, image, platform)
		if err != nil {
			return "", err
		}
	}
	imager, valid := image.(manifest.Imager)
	if !valid {
		return "", errors.New("Choose one platform to load into the local image engine")
	}
	configuration, err := imager.GetConfig()
	if err != nil {
		return "", registry.Safe(err)
	}
	alias, err := ref.New(target)
	if err != nil {
		return "", registry.Safe(err)
	}
	return configuration.Digest.String(), registry.Safe(connection.ImageExport(ctx, reference, output, sdk.ImageWithExportRef(alias)))
}

// localPlatform resolves an explicit platform within a normalized archive's root index.
func localPlatform(ctx context.Context, connection *sdk.RegClient, reference ref.Ref, image manifest.Manifest, platform string) (ref.Ref, manifest.Manifest, error) {
	sources, err := resolveManifest(ctx, connection, reference, image, "", 0)
	if err != nil {
		return reference, image, err
	}
	for _, source := range sources {
		if source.Platform == platform {
			reference = reference.SetDigest(source.Digest)
			image, err = connection.ManifestGet(ctx, reference)
			return reference, image, registry.Safe(err)
		}
	}
	return reference, image, errors.New("Choose one platform to load into the local image engine")
}
