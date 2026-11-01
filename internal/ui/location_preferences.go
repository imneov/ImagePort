package ui

import (
	"fmt"
	"github.com/imneov/ImagePort/internal/domain"
)

// preferenceKey scopes an independent pane's remembered location and access account.
func (pane *locationPane) preferenceKey(field string) string {
	return fmt.Sprintf("commander-pane-%d-%s", pane.side, field)
}

// remember persists opaque target/account IDs only; credentials remain in the vault.
func (pane *locationPane) remember() {
	preferences := pane.workspace.desktop.app.Preferences()
	preferences.SetString(pane.preferenceKey("target"), pane.targetID)
	preferences.SetString(pane.preferenceKey("account-"+pane.targetID), pane.accountID)
}

// rememberedAccount restores a target-specific access choice without coupling the other pane.
func (pane *locationPane) rememberedAccount(target string) string {
	return pane.workspace.desktop.app.Preferences().String(pane.preferenceKey("account-" + target))
}

// account returns saved public metadata for an existing sign-in choice.
func (view *commanderView) account(id string) domain.Profile {
	for _, profile := range view.accounts {
		if profile.ID == id {
			return profile
		}
	}
	return domain.Profile{}
}

// accountOptions disambiguates duplicate display names while retaining stable account IDs.
func accountOptions(accounts []domain.Profile) []string {
	options := make([]string, len(accounts))
	counts := make(map[string]int)
	for index, account := range accounts {
		options[index] = account.Name + " · " + account.Username
		counts[options[index]]++
	}
	for index, account := range accounts {
		if counts[options[index]] > 1 {
			options[index] += " · " + account.ID
		}
	}
	return options
}
