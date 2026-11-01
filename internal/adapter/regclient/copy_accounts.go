package regclient

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/imneov/ImagePort/internal/domain"
)

// copyAccounts stages immutable content to keep two accounts on one host isolated.
func (client *Client) copyAccounts(ctx context.Context, sourceProfile, targetProfile domain.Profile, source domain.Source, target string) error {
	file, err := os.CreateTemp("", "imageport-registry-copy-*.tar")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil {
			slog.Warn("registry copy temporary archive cleanup failed")
		}
	}()
	exportErr := client.Export(ctx, sourceProfile, source, file)
	if err := errors.Join(exportErr, file.Close()); err != nil {
		return err
	}
	return client.Import(ctx, targetProfile, file.Name(), domain.ArchiveEntry{Digest: source.Digest}, target)
}
