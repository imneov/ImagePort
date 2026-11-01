package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// settingsPage exposes registry trust, credential policy and bounded concurrency.
func (desktop *Desktop) settingsPage(version string) fyne.CanvasObject {
	limit := widget.NewSelect([]string{"1", "2", "3", "4", "5", "6", "7", "8"}, func(text string) {
		value, err := strconv.Atoi(text)
		if err == nil {
			err = desktop.presenter.Tasks.SetConcurrency(value)
		}
		if err != nil {
			desktop.showError(err)
			return
		}
		desktop.app.Preferences().SetInt("concurrency", value)
	})
	limit.SetSelected(strconv.Itoa(desktop.app.Preferences().IntWithFallback("concurrency", 2)))
	concurrency := widget.NewFormItem("", limit)
	concurrencyForm := widget.NewForm(concurrency)
	desktop.bindText(func() { concurrency.Text = desktop.tr("Concurrent tasks"); concurrencyForm.Refresh() })
	security := desktop.label("HTTPS and certificate verification are enabled by default. Import a CA per connection when needed. Passwords stay in the OS keyring or explicitly selected session memory. Proxies use HTTP_PROXY, HTTPS_PROXY and NO_PROXY; do not embed credentials in proxy URLs.")
	security.Wrapping = fyne.TextWrapWord
	return container.NewVScroll(container.NewVBox(desktop.heading("Settings", "Personalize your workspace and transfer preferences."), desktop.languageSetting(),
		desktop.appearanceSetting(),
		desktop.card("Transfers", "Applies to new scheduling; the same target remains serialized", concurrencyForm),
		desktop.card("Security and network", "", security), desktop.card("About ImagePort", version, container.NewVBox(desktop.label("Apache-2.0 · Go / Fyne / regclient\nContainer image management · Windows / macOS / Linux"), desktop.button("Font license", desktop.fontLicense)))))
}

// profileDialog keeps passwords out of Fyne preferences and SQLite metadata.
func (desktop *Desktop) profileDialog(existing domain.Profile) {
	name, host, username := widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	password, ca := widget.NewPasswordEntry(), widget.NewEntry()
	name.SetText(existing.Name)
	host.SetText(existing.Host)
	username.SetText(existing.Username)
	host.SetPlaceHolder("registry.example.com or docker.io")
	password.SetPlaceHolder(desktop.tr("Leave blank to keep existing credentials"))
	ca.SetPlaceHolder(desktop.tr("Optional PEM CA certificate file"))
	session := widget.NewCheck(desktop.tr("Keep credentials for this session only"), nil)
	session.SetChecked(existing.SessionOnly)
	loopback := widget.NewCheck(desktop.tr("Use HTTP for a loopback development registry"), nil)
	loopback.SetChecked(existing.HTTP)
	form := desktop.form("Registry connection", "Save connection", "Cancel", []*widget.FormItem{
		widget.NewFormItem(desktop.tr("Name"), name), widget.NewFormItem(desktop.tr("Host"), host), widget.NewFormItem(desktop.tr("Username"), username),
		widget.NewFormItem(desktop.tr("Password / token"), password), widget.NewFormItem(desktop.tr("CA file"), ca), widget.NewFormItem(desktop.tr("Credentials"), session), widget.NewFormItem(desktop.tr("Development"), loopback),
	}, func(confirmed bool) {
		if !confirmed {
			password.SetText("")
			return
		}
		profile := existing
		profile.Name, profile.Host, profile.Username = name.Text, host.Text, username.Text
		profile.SessionOnly, profile.HTTP = session.Checked, loopback.Checked
		value, caPath := password.Text, ca.Text
		password.SetText("")
		if profile.SessionOnly != existing.SessionOnly && value == "" && existing.SecretRef != "" {
			desktop.showError(errors.New("Enter the password again when changing credential storage"))
			return
		}
		desktop.run("Saving connection and secure credentials...", func(context.Context) (any, error) {
			if caPath != "" {
				data, err := os.ReadFile(caPath)
				if err != nil {
					return nil, fmt.Errorf("read CA certificate: %w", err)
				}
				profile.CAPEM = string(data)
			}
			return desktop.presenter.SaveProfile(profile, value)
		}, func(result any) { desktop.reloadProfiles(result.(domain.Profile).ID) })
	})
	form.Resize(fyne.NewSize(750, 440))
	form.Show()
}
