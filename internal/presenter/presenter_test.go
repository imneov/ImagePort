package presenter_test

import (
	"path/filepath"
	"testing"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/presenter"
	"github.com/imneov/ImagePort/internal/secret"
	"github.com/imneov/ImagePort/internal/store"
)

// TestCredentialRotationAndLogout keeps one secret reference and removes it on logout.
func TestCredentialRotationAndLogout(test *testing.T) {
	database, err := store.Open(filepath.Join(test.TempDir(), "profiles.db"))
	if err != nil {
		test.Fatal(err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			test.Error(err)
		}
	}()
	vault := secret.New()
	service := presenter.New(database, vault, nil, nil)
	defer service.Close()
	profile, err := service.SaveProfile(domain.Profile{Name: "Example", Host: "registry.example.com", Username: "user", SessionOnly: true}, "first-secret")
	if err != nil {
		test.Fatal(err)
	}
	originalRef := profile.SecretRef
	profile, err = service.SaveProfile(profile, "second-secret")
	if err != nil {
		test.Fatal(err)
	}
	if profile.SecretRef != originalRef {
		test.Fatal("credential rotation orphaned the old reference")
	}
	value, err := vault.Get(profile.SecretRef, true)
	if err != nil || value != "second-secret" {
		test.Fatal("credential not rotated")
	}
	if err := service.Logout(profile); err != nil {
		test.Fatal(err)
	}
	if _, err := vault.Get(profile.SecretRef, true); err == nil {
		test.Fatal("logout retained credentials")
	}
	saved, err := database.Profile(profile.ID)
	if err != nil {
		test.Fatal(err)
	}
	if saved.SecretRef != "" || saved.Username != "" {
		test.Fatal("logout left account metadata active")
	}
}

// TestSessionSignInAfterRestart restores expired session credentials using new input.
func TestSessionSignInAfterRestart(test *testing.T) {
	database, err := store.Open(filepath.Join(test.TempDir(), "profiles.db"))
	if err != nil {
		test.Fatal(err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			test.Error(err)
		}
	}()
	vault := secret.New()
	service := presenter.New(database, vault, nil, nil)
	defer service.Close()
	profile, err := service.SaveProfile(domain.Profile{Name: "Example", Host: "registry.example.com", Username: "user", SessionOnly: true}, "original")
	if err != nil {
		test.Fatal(err)
	}
	vault.Clear()
	profile, err = service.SaveProfile(profile, "replacement")
	if err != nil {
		test.Fatal(err)
	}
	value, err := vault.Get(profile.SecretRef, true)
	if err != nil || value != "replacement" {
		test.Fatal("could not sign in after session expiry")
	}
}

// TestRegistryAuthorityCannotChangeAfterConsent prevents a stale dialog redirecting writes.
func TestRegistryAuthorityCannotChangeAfterConsent(test *testing.T) {
	database, err := store.Open(filepath.Join(test.TempDir(), "profiles.db"))
	if err != nil {
		test.Fatal(err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			test.Error(err)
		}
	}()
	service := presenter.New(database, secret.New(), nil, nil)
	defer service.Close()
	profile, err := service.SaveProfile(domain.Profile{Name: "Example", Host: "new.example.com"}, "")
	if err != nil {
		test.Fatal(err)
	}
	_, err = service.SubmitRemote(domain.Spec{Operation: "import", ProfileID: profile.ID, RegistryHost: "old.example.com", Target: "old.example.com/repo:tag"})
	if err == nil {
		test.Fatal("stale registry authority accepted")
	}
}
