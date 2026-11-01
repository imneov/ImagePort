package regclient

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
	sdk "github.com/regclient/regclient"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
)

// Resolve returns available platforms without choosing the local CPU implicitly.
func (client *Client) Resolve(ctx context.Context, profile domain.Profile, input string) ([]domain.Source, error) {
	connection, err := client.sdk(profile)
	if err != nil {
		return nil, err
	}
	reference, err := Reference(profile, input)
	if err != nil {
		return nil, err
	}
	image, err := connection.ManifestGet(ctx, reference)
	if err != nil {
		return nil, registry.Safe(err)
	}
	sources, err := resolveManifest(ctx, connection, reference, image, "", 0)
	for index := range sources {
		sources[index].Reference = reference.CommonName()
	}
	return sources, err
}

// resolveManifest walks bounded index trees and rejects non-runnable descriptors.
func resolveManifest(ctx context.Context, connection *sdk.RegClient, reference ref.Ref, image manifest.Manifest, root string, depth int) ([]domain.Source, error) {
	if depth > 8 {
		return nil, errors.New("The image index exceeds the nesting limit")
	}
	if !image.IsList() {
		return resolveSingle(ctx, connection, reference, image, root)
	}
	if root == "" {
		root = image.GetDescriptor().Digest.String()
	}
	index, ok := image.(manifest.Indexer)
	if !ok {
		return nil, errors.New("Cannot read the image index")
	}
	descriptors, err := index.GetManifestList()
	if err != nil {
		return nil, registry.Safe(err)
	}
	if len(descriptors) > 256 {
		return nil, errors.New("The image index contains too many entries")
	}
	var sources []domain.Source
	for _, descriptor := range descriptors {
		if descriptor.Platform != nil && (descriptor.Platform.OS == "unknown" || descriptor.Platform.Architecture == "unknown") {
			continue
		}
		childRef := reference.SetDigest(descriptor.Digest.String())
		child, err := connection.ManifestGet(ctx, childRef)
		if err != nil {
			return nil, registry.Safe(err)
		}
		children, err := resolveManifest(ctx, connection, childRef, child, root, depth+1)
		if err != nil {
			return nil, err
		}
		sources = append(sources, children...)
	}
	if len(sources) == 0 {
		return nil, errors.New("No selectable platform images were found")
	}
	return sources, nil
}

// resolveSingle reads the image config to expose a verified platform and size.
func resolveSingle(ctx context.Context, connection *sdk.RegClient, reference ref.Ref, image manifest.Manifest, root string) ([]domain.Source, error) {
	content, ok := image.(manifest.Imager)
	if !ok {
		return nil, errors.New("The selected object is not a downloadable container image")
	}
	configuration, err := content.GetConfig()
	if err != nil {
		return nil, registry.Safe(err)
	}
	configBlob, err := connection.BlobGetOCIConfig(ctx, reference, configuration)
	if err != nil {
		return nil, registry.Safe(err)
	}
	platform := configBlob.GetConfig().Platform
	if platform.OS == "" || platform.Architecture == "" {
		return nil, errors.New("The image configuration has no platform information")
	}
	label := platform.OS + "/" + platform.Architecture
	if platform.Variant != "" {
		label += "/" + platform.Variant
	}
	if platform.OSVersion != "" {
		label += ",osver=" + platform.OSVersion
	}
	size, err := content.GetSize()
	if err != nil {
		return nil, registry.Safe(err)
	}
	return []domain.Source{{Reference: reference.CommonName(), IndexDigest: root,
		Digest: image.GetDescriptor().Digest.String(), Platform: label, Size: size}}, nil
}

// pinned binds a validated immutable digest to the source repository.
func pinned(profile domain.Profile, source domain.Source) (ref.Ref, error) {
	reference, err := Reference(profile, source.Reference)
	if err != nil {
		return ref.Ref{}, err
	}
	if !strings.Contains(source.Digest, ":") {
		return ref.Ref{}, fmt.Errorf("Invalid source digest")
	}
	return reference.SetDigest(source.Digest), nil
}
