package docker

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/containerd/errdefs"
	"github.com/distribution/reference"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/moby/moby/client"
)

// daemonError preserves cancellation and hides untrusted SDK error bodies.
func daemonError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return errors.New("Cannot access Docker Engine; start Docker Desktop or your Docker provider and refresh local images")
}

// Images lists actual images from the active Docker context without requiring a CLI.
func (adapter *Client) Images(ctx context.Context) (domain.LocalImages, error) {
	return adapter.ImagesIn(ctx, domain.Engine{})
}

// ImagesIn lists a captured or named engine without changing Docker's active context.
func (adapter *Client) ImagesIn(ctx context.Context, expected domain.Engine) (domain.LocalImages, error) {
	connection, engine, err := adapter.open(expected)
	if err != nil {
		return domain.LocalImages{}, err
	}
	defer closeClient(connection)
	response, err := connection.ImageList(ctx, client.ImageListOptions{All: true})
	if err != nil {
		return domain.LocalImages{Engine: engine}, daemonError(ctx, err)
	}
	result := domain.LocalImages{Engine: engine}
	for _, item := range response.Items {
		result.Images = append(result.Images, domain.LocalImage{ID: item.ID, Tags: item.RepoTags, Size: item.Size, Created: time.Unix(item.Created, 0).UTC()})
	}
	return result, nil
}

// Inspect resolves a selected ID or tag and returns the daemon's actual image metadata.
func (adapter *Client) Inspect(ctx context.Context, engine domain.Engine, input string) (domain.LocalImage, error) {
	if _, err := reference.ParseAnyReference(input); err != nil {
		return domain.LocalImage{}, errors.New("Select a valid local image ID or tag")
	}
	connection, _, err := adapter.open(engine)
	if err != nil {
		return domain.LocalImage{}, err
	}
	defer closeClient(connection)
	item, err := connection.ImageInspect(ctx, input)
	if err != nil {
		return domain.LocalImage{}, daemonError(ctx, err)
	}
	platform := item.Os + "/" + item.Architecture
	if item.Variant != "" {
		platform += "/" + item.Variant
	}
	return domain.LocalImage{ID: item.ID, Tags: item.RepoTags, Size: item.Size, Platform: platform}, nil
}

// TagID returns empty only for an absent local tag, never for an engine failure.
func (adapter *Client) TagID(ctx context.Context, engine domain.Engine, target string) (string, error) {
	if _, err := localTag(target); err != nil {
		return "", err
	}
	connection, _, err := adapter.open(engine)
	if err != nil {
		return "", err
	}
	defer closeClient(connection)
	image, err := connection.ImageInspect(ctx, target)
	if errdefs.IsNotFound(err) {
		return "", nil
	}
	return image.ID, daemonError(ctx, err)
}

// localTag requires an explicit mutable tag and rejects digest destinations.
func localTag(input string) (reference.Named, error) {
	parsed, err := reference.ParseNormalizedNamed(input)
	if err != nil {
		return nil, errors.New("Enter a valid image reference with an explicit tag")
	}
	if _, valid := parsed.(reference.Tagged); !valid {
		return nil, errors.New("Enter a valid image reference with an explicit tag")
	}
	if _, digested := parsed.(reference.Digested); digested {
		return nil, errors.New("Enter a valid image reference with an explicit tag")
	}
	return parsed, nil
}

// Tag adds a local alias after rechecking its observed target; original tags remain.
func (adapter *Client) Tag(ctx context.Context, source domain.LocalSource, target, previous string) error {
	if err := validateSource(source); err != nil {
		return err
	}
	if _, err := localTag(target); err != nil {
		return err
	}
	actual, err := adapter.TagID(ctx, source.Engine, target)
	if err != nil {
		return err
	}
	if actual != previous {
		return errors.New("The local target tag changed after confirmation; check it again")
	}
	connection, _, err := adapter.open(source.Engine)
	if err != nil {
		return err
	}
	defer closeClient(connection)
	if _, err := connection.ImageTag(ctx, client.ImageTagOptions{Source: source.ImageID, Target: target}); err != nil {
		return daemonError(ctx, err)
	}
	actual, err = adapter.TagID(ctx, source.Engine, target)
	if err != nil {
		return err
	}
	if actual != source.ImageID {
		return errors.New("The local tag could not be verified; refresh images before trying again")
	}
	return nil
}

// validateSource requires an immutable content ID and a captured engine identity.
func validateSource(source domain.LocalSource) error {
	return domain.ValidateLocalSource(source)
}

// Export streams exactly the pinned image ID; no mutable tag is used for content.
func (adapter *Client) Export(ctx context.Context, source domain.LocalSource, output io.Writer) error {
	if err := validateSource(source); err != nil {
		return err
	}
	connection, _, err := adapter.open(source.Engine)
	if err != nil {
		return err
	}
	defer closeClient(connection)
	stream, err := connection.ImageSave(ctx, []string{source.ImageID})
	if err != nil {
		return daemonError(ctx, err)
	}
	_, writeErr := io.Copy(output, stream)
	return errors.Join(writeErr, daemonError(ctx, stream.Close()))
}
