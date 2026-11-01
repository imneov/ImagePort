package regclient

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
	sdk "github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
)

// slogDiscard prevents SDK diagnostics from leaking server-controlled auth data.
func slogDiscard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// Export writes an OCI-layout tar from a previously selected immutable manifest.
func (client *Client) Export(ctx context.Context, profile domain.Profile, source domain.Source, writer io.Writer) error {
	connection, err := client.sdk(profile)
	if err != nil {
		return err
	}
	reference, err := pinned(profile, source)
	if err != nil {
		return err
	}
	return registry.Safe(connection.ImageExport(ctx, reference, writer))
}

// Copy preserves the complete pinned root; no implicit platform filtering occurs.
func (client *Client) Copy(ctx context.Context, profile domain.Profile, source domain.Source, target string) error {
	connection, err := client.sdk(profile)
	if err != nil {
		return err
	}
	var sourceRef ref.Ref
	if len(source.Reference) > 9 && source.Reference[:9] == "ocidir://" {
		sourceRef, err = ref.New(source.Reference)
	} else {
		sourceRef, err = pinned(profile, source)
	}
	if err != nil {
		return err
	}
	targetRef, err := Reference(profile, target)
	if err != nil {
		return err
	}
	return registry.Safe(connection.ImageCopy(ctx, sourceRef, targetRef))
}

// CopyBetween uses source credentials only at the source host and target credentials at its host.
func (client *Client) CopyBetween(ctx context.Context, sourceProfile, targetProfile domain.Profile, source domain.Source, target string) error {
	if sourceProfile.Host == targetProfile.Host {
		if sourceProfile.ID != targetProfile.ID {
			return client.copyAccounts(ctx, sourceProfile, targetProfile, source, target)
		}
		return client.Copy(ctx, targetProfile, source, target)
	}
	connection, err := client.sdkProfiles(sourceProfile, targetProfile)
	if err != nil {
		return err
	}
	sourceRef, err := pinned(sourceProfile, source)
	if err != nil {
		return err
	}
	targetRef, err := Reference(targetProfile, target)
	if err != nil {
		return err
	}
	return registry.Safe(connection.ImageCopy(ctx, sourceRef, targetRef))
}

// Import pushes one checked OCI root directly from a seekable tar to the target.
func (client *Client) Import(ctx context.Context, profile domain.Profile, path string, entry domain.ArchiveEntry, target string) error {
	connection, err := client.sdk(profile)
	if err != nil {
		return err
	}
	targetRef, err := Reference(profile, target)
	if err != nil {
		return err
	}
	archiveFile, err := os.Open(path)
	if err != nil {
		return err
	}
	err = connection.ImageImport(ctx, targetRef, archiveFile, sdk.ImageWithImportName(entry.Name))
	return errors.Join(registry.Safe(err), archiveFile.Close())
}

// Normalize uses the SDK's Docker conversion in a private temporary OCI layout.
// The returned cleanup must be called; source is pinned to the normalized digest.
func (client *Client) Normalize(ctx context.Context, path string, entry domain.ArchiveEntry) (string, domain.Source, func() error, error) {
	directory, err := os.MkdirTemp("", "imageport-normalize-")
	if err != nil {
		return "", domain.Source{}, nil, err
	}
	cleanup := func() error { return os.RemoveAll(directory) }
	connection := sdk.New(sdk.WithSlog(slogDiscard()))
	reference, err := ref.New("ocidir://" + filepath.ToSlash(directory) + ":import")
	if err != nil {
		return "", domain.Source{}, nil, errors.Join(err, cleanup())
	}
	file, err := os.Open(path)
	if err != nil {
		return "", domain.Source{}, nil, errors.Join(err, cleanup())
	}
	err = connection.ImageImport(ctx, reference, file, sdk.ImageWithImportName(entry.Name))
	err = errors.Join(err, file.Close())
	if err == nil {
		err = connection.Close(ctx, reference)
	}
	if err != nil {
		return "", domain.Source{}, nil, errors.Join(registry.Safe(err), cleanup())
	}
	image, err := connection.ManifestGet(ctx, reference)
	if err != nil {
		return "", domain.Source{}, nil, errors.Join(registry.Safe(err), cleanup())
	}
	digest := image.GetDescriptor().Digest.String()
	return digest, domain.Source{Reference: reference.SetDigest(digest).CommonName(), Digest: digest}, cleanup, nil
}
