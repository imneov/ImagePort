package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	driver "fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// installShortcuts registers the platform's familiar command-search modifiers.
func (desktop *Desktop) installShortcuts() {
	for key, action := range map[fyne.KeyName]func(){fyne.KeyF3: desktop.commander.inspectSelected, fyne.KeyF5: desktop.commander.copySelected, fyne.KeyF8: desktop.commander.deleteSelected} {
		desktop.window.Canvas().AddShortcut(&driver.CustomShortcut{KeyName: key}, func(fyne.Shortcut) {
			if desktop.window.Canvas().Overlays().Top() == nil {
				action()
			}
		})
	}
}

// commandSearch filters destinations and accepts an explicit registry image reference.
func (desktop *Desktop) commandSearch() {
	query := widget.NewEntry()
	query.SetPlaceHolder(desktop.tr("Search pages or enter an image reference"))
	choices := container.NewVBox()
	var popup *dialog.CustomDialog
	refresh := func(value string) {
		choices.Objects = nil
		for index, name := range pageNames {
			page := index
			if strings.Contains(strings.ToLower(desktop.tr(name)), strings.ToLower(value)) {
				choices.Add(widget.NewButton(desktop.tr(name), func() { popup.Hide(); desktop.show(page) }))
			}
		}
		choices.Refresh()
	}
	resolve := widget.NewButtonWithIcon(desktop.tr("Resolve image & platforms"), theme.SearchIcon(), func() {
		if strings.TrimSpace(query.Text) == "" {
			return
		}
		popup.Hide()
		desktop.showRemoteImages()
		desktop.browser.repository.SetText(query.Text)
		desktop.browser.resolve()
	})
	popup = dialog.NewCustom(desktop.tr("Search"), desktop.tr("Cancel"), container.NewVBox(query, choices, widget.NewSeparator(), resolve), desktop.window)
	query.OnChanged = refresh
	query.OnSubmitted = func(string) { resolve.OnTapped() }
	refresh("")
	popup.Resize(fyne.NewSize(620, 460))
	popup.Show()
	desktop.window.Canvas().Focus(query)
}

// openDrawer displays contextual details while retaining the underlying page and filters.
func (desktop *Desktop) openDrawer(title string, content fyne.CanvasObject, actions fyne.CanvasObject) {
	desktop.closeDrawer()
	close := widget.NewButtonWithIcon("", theme.CancelIcon(), desktop.closeDrawer)
	close.Importance = widget.LowImportance
	heading := widget.NewLabelWithStyle(desktop.tr(title), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	desktop.drawerTitle, desktop.drawerTitleMessage = heading, title
	header := container.NewBorder(nil, nil, nil, close, heading)
	panel := widget.NewCard("", "", container.NewBorder(container.NewVBox(header, widget.NewSeparator()), actions, nil, nil, content))
	desktop.drawer = container.New(&drawerLayout{}, panel)
	desktop.workspace.Add(desktop.drawer)
}

// closeDrawer removes only the contextual panel, leaving persistent screen state intact.
func (desktop *Desktop) closeDrawer() {
	if desktop.drawer != nil && desktop.workspace != nil {
		desktop.workspace.Remove(desktop.drawer)
	}
	desktop.drawer, desktop.drawerRefresh, desktop.drawerTitle = nil, nil, nil
	if desktop.taskProgress != nil {
		desktop.taskProgress.Stop()
		desktop.taskProgress.Hide()
	}
}

// drawerLayout aligns the details card to the right of the workspace.
type drawerLayout struct{}

// Layout sizes a drawer within the available workspace without expanding the window.
func (*drawerLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	width := min(float32(570), size.Width*0.65)
	for _, object := range objects {
		object.Move(fyne.NewPos(size.Width-width, 0))
		object.Resize(fyne.NewSize(width, size.Height))
	}
}

// MinSize allows the overlay to fit the workspace's existing bounds.
func (*drawerLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(0, 0) }
