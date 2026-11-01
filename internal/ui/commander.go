package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// commanderView owns two independent locations and commands for the active pane.
type commanderView struct {
	desktop            *Desktop
	panes              [2]*locationPane
	active             int
	targets            []domain.Target
	accounts           []domain.Profile
	inspect, copy, cut *widget.Button
	queue              *fyne.Container
	queueList          *widget.List
	queueRows          []domain.Task
	seen               map[string]domain.State
	manager            *targetManager
	seeding            bool
	seeded             bool
}

// commanderShell builds the approved toolbar, contiguous panes and shared operation bar.
func (desktop *Desktop) commanderShell(version string) fyne.CanvasObject {
	view := &commanderView{desktop: desktop, seen: make(map[string]domain.State)}
	desktop.commander = view
	desktop.breadcrumb = widget.NewLabel("")
	desktop.activity = desktop.button("Transfers", view.showHistory)
	desktop.activity.SetIcon(theme.DownloadIcon())
	view.panes[0], view.panes[1] = view.newPane(0), view.newPane(1)
	split := container.NewHSplit(view.panes[0].content, view.panes[1].content)
	split.Offset = 0.5
	view.inspect = desktop.button("Inspect · F3", view.inspectSelected)
	view.copy = desktop.button("Copy · F5", view.copySelected)
	view.cut = desktop.button("Delete · F8", view.deleteSelected)
	commands := container.NewGridWithColumns(3, view.inspect, view.copy, view.cut)
	view.queue = view.transferQueue()
	desktop.workspace = container.NewStack(container.NewBorder(nil, view.queue, nil, nil, split))
	desktop.bindText(func() { view.translate(); view.updateCommands() })
	return container.NewBorder(container.NewVBox(view.toolbar(version), widget.NewSeparator()), container.NewVBox(widget.NewSeparator(), commands), nil, nil, desktop.workspace)
}

// toolbar keeps location configuration and application preferences within easy reach.
func (view *commanderView) toolbar(version string) fyne.CanvasObject {
	desktop := view.desktop
	brand := widget.NewLabelWithStyle("ImagePort", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	swap := desktop.button("Swap panes", view.swap)
	manage := desktop.button("Manage targets", func() { view.openManager("") })
	manage.SetIcon(theme.ComputerIcon())
	settings := desktop.button("Settings", func() {
		popup := dialog.NewCustom(desktop.tr("Settings"), desktop.tr("Close"), desktop.pages[preferencesPage], desktop.window)
		popup.Resize(fyne.NewSize(720, 620))
		popup.Show()
	})
	language := widget.NewButton("", func() {
		code := "zh-Hans"
		if desktop.language == code {
			code = "en"
		}
		desktop.changeLanguage(code)
	})
	desktop.bindText(func() {
		caption := "EN"
		if desktop.language == "zh-Hans" {
			caption = "中"
		}
		language.SetText(caption)
	})
	appearance := widget.NewButtonWithIcon("", theme.ColorPaletteIcon(), desktop.toggleTheme)
	left := container.NewHBox(brand, swap, desktop.activity)
	right := container.NewHBox(manage, settings, language, appearance)
	return inset(container.NewBorder(nil, nil, left, right, layout.NewSpacer()), 4)
}

// reloadTargets loads the shared catalog while retaining each pane's independent identity.
func (view *commanderView) reloadTargets() {
	type snapshot struct {
		targets  []domain.Target
		accounts []domain.Profile
	}
	view.desktop.run("Loading targets...", func(context.Context) (any, error) {
		targets, err := view.desktop.presenter.Store.Targets()
		if err != nil {
			return nil, err
		}
		accounts, err := view.desktop.presenter.Store.Profiles()
		return snapshot{targets, accounts}, err
	}, func(value any) {
		result := value.(snapshot)
		view.targets, view.accounts = result.targets, result.accounts
		view.desktop.profiles = result.accounts
		view.seedLocations()
		for _, pane := range view.panes {
			pane.reloadOptions()
		}
		if view.manager != nil {
			view.manager.refresh()
		}
	})
}

// seedLocations creates OS-derived initial locations once, without probing a daemon or creating files.
func (view *commanderView) seedLocations() {
	if view.seeded || view.seeding {
		return
	}
	view.seeding = true
	view.desktop.presenter.Work(func(context.Context) (any, error) { return view.desktop.presenter.EnsureLocalTargets() }, func(value any, err error) {
		fyne.Do(func() {
			view.seeding = false
			if view.desktop.closing {
				return
			}
			if err != nil {
				view.desktop.showError(err)
				return
			}
			view.seeded = true
			if value.(bool) {
				view.reloadTargets()
			}
		})
	})
}

// target returns the shared metadata for an opaque selected target ID.
func (view *commanderView) target(id string) (domain.Target, bool) {
	for _, target := range view.targets {
		if target.ID == id {
			return target, true
		}
	}
	return domain.Target{}, false
}

// paneAccounts filters identities without turning accounts into duplicate locations.
func (view *commanderView) paneAccounts(id string) []domain.Profile {
	var accounts []domain.Profile
	for _, account := range view.accounts {
		if account.TargetID == id {
			accounts = append(accounts, account)
		}
	}
	return accounts
}

// activate changes command ownership without altering either location or selection.
func (view *commanderView) activate(side int) {
	view.active = side
	for index, pane := range view.panes {
		if pane.activeMarker == nil {
			continue
		}
		pane.activeMarker.FillColor = theme.SeparatorColor()
		if index == side {
			pane.activeMarker.FillColor = theme.PrimaryColor()
		}
		pane.activeMarker.Refresh()
	}
	view.updateCommands()
}

// swap exchanges location state while keeping widgets, focus and callbacks on their own side.
func (view *commanderView) swap() {
	left, right, active := view.panes[0].snapshot(), view.panes[1].snapshot(), view.active
	view.panes[0].restore(right)
	view.panes[1].restore(left)
	view.activate(active)
}

// translate updates existing pane labels while preserving paths and selected content.
func (view *commanderView) translate() {
	for _, pane := range view.panes {
		pane.reloadOptions()
		pane.render()
	}
}

// updateCommands binds the shared bottom bar to the active source and opposite destination.
func (view *commanderView) updateCommands() {
	if view.inspect == nil {
		return
	}
	pane, other := view.panes[view.active], view.panes[1-view.active]
	entry, selected := pane.nodes[pane.selected]
	view.inspect.Disable()
	view.copy.Disable()
	view.cut.Disable()
	view.copy.SetText(view.desktop.tr("Copy · F5"))
	if selected && entry.Kind != "more" {
		view.inspect.Enable()
	}
	if !selected || (entry.Kind != "version" && entry.Kind != "archive") {
		return
	}
	source, sourceAvailable := view.target(pane.targetID)
	if !sourceAvailable {
		return
	}
	destination, available := view.target(other.targetID)
	if available && !(source.Kind == "filesystem" && destination.Kind == "filesystem") {
		view.copy.SetText(view.desktop.tr("Copy to %s · F5", destination.Name))
		view.copy.Enable()
	}
	if entry.Kind == "archive" || (source.Kind != "filesystem" && entry.Repository != "") {
		view.cut.Enable()
	}
}
