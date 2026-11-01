// Package secret keeps passwords outside persistent metadata and task snapshots.
package secret

import (
	"errors"
	"sync"

	"github.com/zalando/go-keyring"
)

const service = "io.github.imneov.ImagePort"

// ErrNotFound distinguishes absent credentials from a locked or unavailable store.
var ErrNotFound = errors.New("Credentials are missing or expired; enter the password again")

// Vault combines an OS credential store with explicitly selected session memory.
type Vault struct {
	mu      sync.RWMutex
	session map[string]string
}

// New returns an empty session vault; it does not probe or unlock the OS keyring.
func New() *Vault { return &Vault{session: make(map[string]string)} }

// Set stores value under id; sessionOnly never falls back to a plaintext file.
func (vault *Vault) Set(id, value string, sessionOnly bool) error {
	if sessionOnly {
		vault.mu.Lock()
		defer vault.mu.Unlock()
		vault.session[id] = value
		return nil
	}
	if err := keyring.Set(service, id, value); err != nil {
		return errors.New("The system keyring is unavailable; grant access or explicitly choose session-only storage")
	}
	return nil
}

// Get retrieves id using the profile's explicit storage policy.
func (vault *Vault) Get(id string, sessionOnly bool) (string, error) {
	if id == "" {
		return "", nil
	}
	if sessionOnly {
		vault.mu.RLock()
		defer vault.mu.RUnlock()
		value, found := vault.session[id]
		if !found {
			return "", ErrNotFound
		}
		return value, nil
	}
	value, err := keyring.Get(service, id)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", errors.New("Cannot read the system keyring; sign in again or grant access")
	}
	return value, nil
}

// Delete removes a profile's secret; missing credentials count as signed out.
func (vault *Vault) Delete(id string, sessionOnly bool) error {
	if id == "" {
		return nil
	}
	if sessionOnly {
		vault.mu.Lock()
		defer vault.mu.Unlock()
		delete(vault.session, id)
		return nil
	}
	if err := keyring.Delete(service, id); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return errors.New("Cannot delete credentials from the system keyring")
	}
	return nil
}

// Clear drops all session references when the application exits.
func (vault *Vault) Clear() {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	clear(vault.session)
}
