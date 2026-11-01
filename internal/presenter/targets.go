package presenter

import (
	"errors"
	"net/url"
	"strings"

	"github.com/imneov/ImagePort/internal/domain"
)

// SaveTarget normalizes and saves one location without copying access credentials.
func (presenter *Presenter) SaveTarget(target domain.Target) (domain.Target, error) {
	presenter.targetsMu.Lock()
	defer presenter.targetsMu.Unlock()
	target.Name, target.Address = strings.TrimSpace(target.Name), strings.TrimSpace(target.Address)
	if target.Kind == "registry" {
		if !strings.Contains(target.Address, "://") {
			target.Address = "https://" + target.Address
		}
		target.Address = strings.TrimSuffix(target.Address, "/")
	}
	if err := domain.ValidateTarget(target); err != nil {
		return target, err
	}
	targets, err := presenter.Store.Targets()
	if err != nil {
		return target, err
	}
	for _, saved := range targets {
		if saved.ID != target.ID && saved.Kind == target.Kind && saved.Address == target.Address {
			return target, errors.New("This location is already saved; edit the existing target")
		}
		if saved.ID == target.ID && saved.Kind != target.Kind {
			return target, errors.New("Create a new target when changing its location type")
		}
		if saved.ID == target.ID && saved.Address != target.Address {
			accounts, err := presenter.TargetAccounts(target.ID)
			if err != nil {
				return target, err
			}
			if len(accounts) > 0 {
				return target, errors.New("Create a new registry target when changing an address that has saved accounts")
			}
		}
	}
	if target.ID == "" {
		target.ID, err = domain.NewID()
		if err != nil {
			return target, err
		}
	}
	return target, presenter.Store.SaveTarget(target)
}

// TargetAccounts returns only identities belonging to the selected registry target.
func (presenter *Presenter) TargetAccounts(targetID string) ([]domain.Profile, error) {
	profiles, err := presenter.Store.Profiles()
	if err != nil {
		return nil, err
	}
	var result []domain.Profile
	for _, profile := range profiles {
		if profile.TargetID == targetID {
			result = append(result, profile)
		}
	}
	return result, nil
}

// AccessProfile binds saved credentials to an endpoint, or returns anonymous access.
func (presenter *Presenter) AccessProfile(target domain.Target, accountID string) (domain.Profile, error) {
	parsed, err := url.Parse(target.Address)
	if err != nil {
		return domain.Profile{}, err
	}
	if accountID == "" {
		return domain.Profile{Name: target.Name, Host: parsed.Host, HTTP: parsed.Scheme == "http", TargetID: target.ID}, nil
	}
	profile, err := presenter.Store.Profile(accountID)
	if err != nil {
		return profile, err
	}
	if profile.TargetID != target.ID || profile.Host != parsed.Host || profile.HTTP != (parsed.Scheme == "http") {
		return profile, errors.New("The access account does not belong to this registry target")
	}
	return profile, nil
}

// RemoveTarget deletes its account secrets and configuration, never content or tasks.
func (presenter *Presenter) RemoveTarget(id string) error {
	presenter.targetsMu.Lock()
	defer presenter.targetsMu.Unlock()
	accounts, err := presenter.TargetAccounts(id)
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if err := presenter.RemoveAccount(account); err != nil {
			return err
		}
	}
	return presenter.Store.DeleteTarget(id)
}

// RemoveAccount removes one identity after clearing only its own protected secret.
func (presenter *Presenter) RemoveAccount(profile domain.Profile) error {
	presenter.credentialsMu.Lock()
	defer presenter.credentialsMu.Unlock()
	current, err := presenter.Store.Profile(profile.ID)
	if err != nil {
		return err
	}
	profile = current
	if profile.SecretRef != "" {
		if err := presenter.Vault.Delete(profile.SecretRef, profile.SessionOnly); err != nil {
			return err
		}
	}
	return presenter.Store.DeleteProfile(profile.ID)
}

// validateAccountTarget prevents credentials from being rebound to another saved endpoint.
func (presenter *Presenter) validateAccountTarget(profile domain.Profile) error {
	if profile.TargetID == "" {
		return nil
	}
	targets, err := presenter.Store.Targets()
	if err != nil {
		return err
	}
	for _, target := range targets {
		if target.ID != profile.TargetID {
			continue
		}
		parsed, err := url.Parse(target.Address)
		if err != nil {
			return err
		}
		if target.Kind == "registry" && parsed.Host == profile.Host && (parsed.Scheme == "http") == profile.HTTP {
			return nil
		}
	}
	return errors.New("The access account does not belong to this registry target")
}
