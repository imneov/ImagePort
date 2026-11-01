package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// registriesPage separates connection management from image browsing.
func (desktop *Desktop) registriesPage() fyne.CanvasObject {
	desktop.registryCards = container.NewVBox()
	add := desktop.button("Add registry", func() { desktop.profileDialog(domain.Profile{}) })
	add.SetIcon(theme.ContentAddIcon())
	add.Importance = widget.HighImportance
	top := container.NewBorder(nil, nil, nil, container.NewCenter(add), desktop.heading("Registries", "Connect to public and private image registries."))
	desktop.bindText(desktop.renderRegistries)
	return container.NewBorder(top, nil, nil, nil, container.NewVScroll(desktop.registryCards))
}

// renderRegistries rebuilds connection cards from saved metadata, never credential values.
func (desktop *Desktop) renderRegistries() {
	if desktop.registryCards == nil {
		return
	}
	desktop.registryCards.Objects = nil
	if len(desktop.profiles) == 0 {
		desktop.registryCards.Add(desktop.emptyState("No registries yet", "Add a registry connection to browse and manage its images.", "Add registry", func() { desktop.profileDialog(domain.Profile{}) }))
	}
	for index, profile := range desktop.profiles {
		selected, current := index, profile
		browse := widget.NewButtonWithIcon(desktop.tr("Browse images"), theme.StorageIcon(), func() {
			desktop.profile.SetSelectedIndex(selected)
			desktop.showRemoteImages()
			desktop.browser.browse()
		})
		edit := widget.NewButton(desktop.tr("Edit / sign in"), func() { desktop.profileDialog(current) })
		check := widget.NewButton(desktop.tr("Check connection"), func() { desktop.checkProfile(current) })
		logout := widget.NewButton(desktop.tr("Sign out"), func() { desktop.signOut(current) })
		policy := desktop.tr("OS keyring")
		if current.SessionOnly {
			policy = desktop.tr("Session memory")
		}
		summary := widget.NewLabel(fmt.Sprintf("%s  ·  %s\n%s", current.Host, desktop.tr("OCI registry"), policy))
		desktop.registryCards.Add(widget.NewCard(current.Name, "", container.NewVBox(summary, container.NewHBox(browse, edit, check, layout.NewSpacer(), logout))))
	}
	desktop.registryCards.Refresh()
}

// checkProfile checks reachability and explicitly distinguishes it from image permissions.
func (desktop *Desktop) checkProfile(profile domain.Profile) {
	desktop.run("Checking registry connectivity...", func(ctx context.Context) (any, error) { return nil, desktop.presenter.Registry.Check(ctx, profile) }, func(any) {
		desktop.inform("Connection check", "The registry is reachable. This does not prove authentication or permission to read or push a particular image.")
	})
}

// signOut requests consent before removing this connection's secure credentials.
func (desktop *Desktop) signOut(profile domain.Profile) {
	desktop.confirm("Sign out", "Delete this connection's session or keyring credentials? Tasks fetch credentials again when needed.", func(confirmed bool) {
		if confirmed {
			desktop.run("Removing credentials...", func(context.Context) (any, error) { return nil, desktop.presenter.Logout(profile) }, func(any) { desktop.reloadProfiles(profile.ID) })
		}
	})
}

// emptyState offers a concrete next action when a collection has no real items.
func (desktop *Desktop) emptyState(title, description, action string, tapped func()) fyne.CanvasObject {
	text := widget.NewLabel(desktop.tr(description))
	text.Wrapping = fyne.TextWrapWord
	button := widget.NewButton(desktop.tr(action), tapped)
	button.SetIcon(theme.ContentAddIcon())
	return container.NewCenter(widget.NewCard(desktop.tr(title), "", container.NewVBox(text, button)))
}
