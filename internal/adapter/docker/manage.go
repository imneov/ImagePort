package docker

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/imneov/ImagePort/internal/local"
	"io"
	"time"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/moby/moby/client"
)

// Load checks target consent, consumes daemon errors, and reads back the exact content ID.
func (adapter *Client) Load(ctx context.Context, engine domain.Engine, input io.Reader, target, previous, expected string) (string, error) {
	actual, err := adapter.TagID(ctx, engine, target)
	if err != nil {
		return "", err
	}
	if actual != previous {
		return "", errors.New("The local target tag changed after confirmation; check it again")
	}
	connection, _, err := adapter.open(engine)
	if err != nil {
		return "", err
	}
	defer closeClient(connection)
	// The SDK transfers request-body ownership when a Reader also implements Closer.
	// Hide Close so the caller retains ownership of its archive descriptor.
	stream, err := connection.ImageLoad(ctx, struct{ io.Reader }{input}, client.ImageLoadWithQuiet(true))
	if err != nil {
		return "", daemonError(ctx, err)
	}
	loadErr := errors.Join(readLoadResult(stream), stream.Close())
	actual, verifyErr := adapter.TagID(ctx, engine, target)
	if verifyErr == nil && actual == expected {
		return target + "\n" + actual, nil
	}
	if loadErr != nil {
		return "", loadErr
	}
	return "", errors.New("The loaded image could not be verified; check the local image before retrying")
}

// readLoadResult consumes the bounded status stream without exposing server text or ignoring errors.
func readLoadResult(input io.Reader) error {
	decoder := json.NewDecoder(io.LimitReader(input, 16<<20))
	for {
		var message struct {
			Error string `json:"error"`
		}
		err := decoder.Decode(&message)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil || message.Error != "" {
			return errors.New("Docker could not load the selected image archive")
		}
	}
}

// RemoveTag removes only a pinned alias, never forcing container removal or pruning parents.
func (adapter *Client) RemoveTag(ctx context.Context, engine domain.Engine, target, expected string) error {
	actual, err := adapter.TagID(ctx, engine, target)
	if err != nil {
		return err
	}
	if actual != expected {
		return errors.New("The local image changed after confirmation; refresh and select it again")
	}
	connection, _, err := adapter.open(engine)
	if err != nil {
		return err
	}
	defer closeClient(connection)
	_, writeErr := connection.ImageRemove(ctx, target, client.ImageRemoveOptions{Force: false, PruneChildren: false})
	verification, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	actual, verifyErr := adapter.TagID(verification, engine, target)
	if verifyErr == nil && actual == "" {
		return nil
	}
	if verifyErr == nil && actual == expected && writeErr != nil {
		return daemonError(ctx, writeErr)
	}
	return &local.UnknownResult{Message: "The local tag removal could not be verified; refresh local images"}
}
