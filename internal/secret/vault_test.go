package secret_test

import (
	"testing"

	"github.com/imneov/ImagePort/internal/secret"
)

// TestSessionLifecycle never touches the OS keyring and verifies explicit erasure.
func TestSessionLifecycle(test *testing.T) {
	vault := secret.New()
	if err := vault.Set("profile", "session-secret", true); err != nil {
		test.Fatal(err)
	}
	value, err := vault.Get("profile", true)
	if err != nil || value != "session-secret" {
		test.Fatal("session credential unavailable")
	}
	vault.Clear()
	if _, err := vault.Get("profile", true); err == nil {
		test.Fatal("session credential survived clear")
	}
	if err := vault.Delete("profile", true); err != nil {
		test.Fatal(err)
	}
}
