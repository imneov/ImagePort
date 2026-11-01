package presenter

import (
	"github.com/imneov/ImagePort/internal/domain"
	"os"
)

// EnsureLocalTargets initializes OS-derived local locations once per data directory, without probing Docker.
func (presenter *Presenter) EnsureLocalTargets() (bool, error) {
	presenter.targetsMu.Lock()
	defer presenter.targetsMu.Unlock()
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	engineID, err := domain.NewID()
	if err != nil {
		return false, err
	}
	folderID, err := domain.NewID()
	if err != nil {
		return false, err
	}
	return presenter.Store.InitializeTargets([]domain.Target{
		{ID: engineID, Name: "Local images", Kind: "engine"},
		{ID: folderID, Name: "Home folder", Kind: "filesystem", Address: home},
	})
}
