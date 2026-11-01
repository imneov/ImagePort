package presenter

import (
	"errors"
	"os"
	"strings"

	"github.com/distribution/reference"
	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
)

// SubmitWorkspace validates a captured Commander operation before durable scheduling.
func (presenter *Presenter) SubmitWorkspace(spec domain.Spec) (string, error) {
	switch spec.Operation {
	case "copy", "push", "import":
		return presenter.SubmitRemote(spec)
	case "download", "pull", "delete-remote":
		profile, err := presenter.Store.Profile(spec.ProfileID)
		if err != nil {
			return "", err
		}
		if profile.Host != spec.RegistryHost {
			return "", errors.New("The connection address changed; check the target again")
		}
		if spec.Operation == "delete-remote" {
			spec.Target, err = adapter.Canonical(profile, spec.Target, true)
			if err != nil {
				return "", err
			}
		}
		if spec.Source.Digest == "" || spec.Source.Platform == "" {
			return "", errors.New("Select a platform first")
		}
	case "export-local", "delete-local", "copy-local":
		if err := domain.ValidateLocalSource(spec.Local); err != nil {
			return "", err
		}
	case "load", "delete-file":
		if spec.Archive.SHA256 == "" {
			return "", errors.New("Inspect the archive and select an entry first")
		}
	default:
		return "", errors.New("Invalid operation type")
	}
	if spec.Output != "" {
		path, err := archive.Destination(spec.Output)
		if err != nil {
			return "", err
		}
		if _, err := os.Lstat(path); err == nil {
			return "", errors.New("The file already exists; choose another filename")
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		spec.Output = path
	}
	if err := validateEngineDestination(spec); err != nil {
		return "", err
	}
	return presenter.Tasks.Submit(spec)
}

// validateEngineDestination rejects unpinned engines and implicit local target tags.
func validateEngineDestination(spec domain.Spec) error {
	if spec.Operation != "load" && spec.Operation != "pull" && spec.Operation != "copy-local" {
		return nil
	}
	if spec.DestinationEngine.Identity == "" || spec.DestinationEngine.Host == "" {
		return errors.New("Refresh the destination local image engine before copying")
	}
	parsed, err := reference.ParseNormalizedNamed(spec.Target)
	if err != nil || strings.Contains(spec.Target, "@") {
		return errors.New("Enter a valid image reference with an explicit tag")
	}
	if _, valid := parsed.(reference.Tagged); !valid {
		return errors.New("Enter a valid image reference with an explicit tag")
	}
	return nil
}
