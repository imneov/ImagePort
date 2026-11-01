package ui

import (
	"context"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// targetManager owns one shared location catalog and a separate account editor.
type targetManager struct {
	workspace *commanderView
	selected  string
	listing   *widget.List
	details   *fyne.Container
	popup     *dialog.CustomDialog
	draft     domain.Target
	name      *widget.Entry
	address   *widget.Entry
}

// openManager opens the Commander-style saved-target list and configuration pane.
func (view *commanderView) openManager(id string) {
	if id == "" {
		id = view.panes[view.active].targetID
	}
	manager := &targetManager{workspace: view, selected: id, details: container.NewStack()}
	view.manager = manager
	manager.listing = widget.NewList(func() int { return len(view.targets) }, func() fyne.CanvasObject {
		label := widget.NewLabel("\n")
		label.Truncation = fyne.TextTruncateEllipsis
		return label
	}, func(index widget.ListItemID, object fyne.CanvasObject) {
		target := view.targets[index]
		object.(*widget.Label).SetText(target.Name + "\n" + view.desktop.tr(targetKind(target.Kind)))
	})
	manager.listing.OnSelected = func(index widget.ListItemID) {
		manager.selected = view.targets[index].ID
		manager.edit(view.targets[index])
	}
	add := widget.NewButtonWithIcon(view.desktop.tr("New target"), theme.ContentAddIcon(), manager.newTarget)
	left := container.NewBorder(nil, add, nil, nil, manager.listing)
	split := container.NewHSplit(left, manager.details)
	split.Offset = .27
	manager.popup = dialog.NewCustom(view.desktop.tr("Manage targets"), view.desktop.tr("Close"), split, view.desktop.window)
	manager.popup.SetOnClosed(func() { view.manager = nil })
	manager.popup.Resize(fyne.NewSize(940, 650))
	manager.refresh()
	manager.popup.Show()
}

// refresh keeps the edited target selected after catalog saves without changing either browser.
func (manager *targetManager) refresh() {
	manager.listing.Refresh()
	for index, target := range manager.workspace.targets {
		if target.ID == manager.selected {
			manager.listing.Select(index)
			manager.edit(target)
			return
		}
	}
	manager.newTarget()
}

// newTarget offers exactly the supported location types before showing configuration.
func (manager *targetManager) newTarget() {
	manager.selected = ""
	manager.listing.UnselectAll()
	choices := container.NewGridWithColumns(2)
	icons := map[string]fyne.Resource{"registry": theme.StorageIcon(), "engine": theme.ComputerIcon(), "filesystem": theme.FolderIcon()}
	for _, kind := range []string{"registry", "engine", "filesystem"} {
		kind := kind
		choices.Add(widget.NewButtonWithIcon(manager.workspace.desktop.tr(targetKind(kind)), icons[kind], func() { manager.edit(domain.Target{Kind: kind}) }))
	}
	manager.details.Objects = []fyne.CanvasObject{container.NewBorder(widget.NewLabel(manager.workspace.desktop.tr("New target")), nil, nil, nil, container.NewVBox(choices))}
	manager.details.Refresh()
}

// edit shows metadata on the right and the selected registry's separate access accounts.
func (manager *targetManager) edit(target domain.Target) {
	manager.draft = target
	desktop := manager.workspace.desktop
	manager.name, manager.address = widget.NewEntry(), widget.NewEntry()
	manager.name.SetText(target.Name)
	manager.address.SetText(target.Address)
	addressLabel := map[string]string{"registry": "Registry address", "filesystem": "Folder path", "engine": "Docker context"}[target.Kind]
	if target.Kind == "registry" {
		manager.address.SetPlaceHolder("https://registry.example.com")
	}
	if target.Kind == "engine" {
		manager.address.SetPlaceHolder(desktop.tr("Current Docker context"))
	}
	address := fyne.CanvasObject(manager.address)
	if target.Kind == "filesystem" {
		choose := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
			desktop.openFolder(func(uri fyne.ListableURI, err error) {
				if err != nil {
					desktop.showError(err)
				} else if uri != nil {
					manager.address.SetText(uri.Path())
				}
			})
		})
		address = container.NewBorder(nil, nil, nil, choose, manager.address)
	}
	form := widget.NewForm(widget.NewFormItem(desktop.tr("Display name"), manager.name), widget.NewFormItem(desktop.tr(addressLabel), address))
	content := container.NewVBox(widget.NewLabel(desktop.tr(targetKind(target.Kind))), form)
	if target.Kind == "registry" && target.ID != "" {
		content.Add(manager.accounts(target))
	}
	save := widget.NewButton(desktop.tr("Save target"), manager.save)
	save.Importance = widget.HighImportance
	actions := container.NewHBox(layout.NewSpacer(), save)
	if target.ID != "" {
		remove := widget.NewButton(desktop.tr("Remove target"), manager.remove)
		actions.Objects = append([]fyne.CanvasObject{remove}, actions.Objects...)
	}
	manager.details.Objects = []fyne.CanvasObject{inset(container.NewBorder(nil, actions, nil, nil, container.NewVScroll(content)), 12)}
	manager.details.Refresh()
}

// accounts lists access identities without inventing permissions or duplicating registry targets.
func (manager *targetManager) accounts(target domain.Target) fyne.CanvasObject {
	desktop := manager.workspace.desktop
	content := container.NewVBox(widget.NewSeparator(), widget.NewLabel(desktop.tr("Access accounts")))
	for _, profile := range manager.workspace.paneAccounts(target.ID) {
		if profile.Username == "" {
			continue
		}
		profile := profile
		label := widget.NewLabel(profile.Name + " · " + profile.Username)
		label.Truncation = fyne.TextTruncateEllipsis
		edit := widget.NewButton(desktop.tr("Edit / sign in"), func() {
			manager.workspace.accountDialog(target, profile, func(domain.Profile) { manager.workspace.reloadTargets() })
		})
		remove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() { manager.removeAccount(profile) })
		content.Add(container.NewBorder(nil, nil, nil, container.NewHBox(edit, remove), label))
	}
	content.Add(widget.NewButtonWithIcon(desktop.tr("Add account"), theme.ContentAddIcon(), func() {
		manager.workspace.accountDialog(target, domain.Profile{}, func(domain.Profile) { manager.workspace.reloadTargets() })
	}))
	return content
}

// save persists validated configuration and then refreshes both target dropdowns.
func (manager *targetManager) save() {
	target := manager.draft
	target.Name, target.Address = strings.TrimSpace(manager.name.Text), strings.TrimSpace(manager.address.Text)
	manager.workspace.desktop.run("Saving target...", func(context.Context) (any, error) {
		return manager.workspace.desktop.presenter.SaveTarget(target)
	}, func(value any) {
		manager.selected = value.(domain.Target).ID
		manager.workspace.reloadTargets()
	})
}

// remove confirms configuration scope and never deletes an image, project or exported file.
func (manager *targetManager) remove() {
	target := manager.draft
	manager.workspace.desktop.confirm("Remove target?", manager.workspace.desktop.tr("Remove this saved target and its accounts? Images and files are kept."), func(confirmed bool) {
		if confirmed {
			manager.workspace.desktop.run("Removing target configuration...", func(context.Context) (any, error) {
				return nil, manager.workspace.desktop.presenter.RemoveTarget(target.ID)
			}, func(any) { manager.selected = ""; manager.workspace.reloadTargets() })
		}
	})
}

// removeAccount removes one saved identity after an explicit credentials-only confirmation.
func (manager *targetManager) removeAccount(profile domain.Profile) {
	manager.workspace.desktop.confirm("Remove account?", manager.workspace.desktop.tr("Remove this saved account and its credentials? Images are kept."), func(confirmed bool) {
		if confirmed {
			manager.workspace.desktop.run("Removing credentials...", func(context.Context) (any, error) {
				return nil, manager.workspace.desktop.presenter.RemoveAccount(profile)
			}, func(any) { manager.workspace.reloadTargets() })
		}
	})
}
