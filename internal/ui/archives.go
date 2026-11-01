package ui

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// archivesView lists real inspected paths and verified downloads, without scanning arbitrary files.
type archivesView struct {
	desktop     *Desktop
	filter      *widget.Entry
	listing     *widget.List
	paths, rows []string
	summary     *widget.Label
	empty       fyne.CanvasObject
}

// archivesPage exposes known image archives and reinspection before any upload.
func (desktop *Desktop) archivesPage() fyne.CanvasObject {
	view := &archivesView{desktop: desktop, filter: widget.NewEntry(), summary: widget.NewLabel("")}
	desktop.archives = view
	desktop.placeholder(view.filter, "Search local archives")
	view.filter.OnChanged = func(string) { view.refresh() }
	view.listing = widget.NewList(func() int { return len(view.rows) }, func() fyne.CanvasObject {
		label := widget.NewLabel("")
		label.Truncation = fyne.TextTruncateEllipsis
		inspect := widget.NewButtonWithIcon("", theme.SearchIcon(), nil)
		return container.NewBorder(nil, nil, nil, inspect, label)
	}, func(index widget.ListItemID, object fyne.CanvasObject) {
		path := view.rows[index]
		row := object.(*fyne.Container)
		row.Objects[0].(*widget.Label).SetText(filepath.Base(path) + "\n" + path)
		row.Objects[1].(*widget.Button).OnTapped = func() { desktop.importDialog(path) }
	})
	view.listing.OnSelected = func(index widget.ListItemID) { desktop.importDialog(view.rows[index]) }
	add := desktop.button("Open archive", func() { desktop.importDialog("") })
	add.SetIcon(theme.FolderOpenIcon())
	add.Importance = widget.HighImportance
	header := container.NewBorder(nil, nil, nil, container.NewCenter(add), desktop.heading("Local archives", "Verified downloads and archives you have inspected. Files remain in their original locations."))
	description := desktop.label("Download an image or open an existing OCI / Docker archive to inspect it.")
	description.Wrapping = fyne.TextWrapWord
	view.empty = container.NewCenter(desktop.card("No local archives yet", "", container.NewVBox(description, desktop.button("Open archive", func() { desktop.importDialog("") }))))
	content := container.NewStack(view.empty, view.listing)
	desktop.bindText(view.refresh)
	return container.NewBorder(container.NewVBox(header, view.filter), view.summary, nil, nil, content)
}

// refresh merges persisted inspected paths with successful download results by absolute path.
func (view *archivesView) refresh() {
	view.paths = nil
	seen := make(map[string]bool)
	add := func(path string) {
		if path != "" && !seen[path] {
			seen[path] = true
			view.paths = append(view.paths, path)
		}
	}
	for _, path := range view.desktop.app.Preferences().StringList("archive-paths") {
		add(path)
	}
	for _, item := range view.desktop.tasks {
		if item.State == domain.Succeeded && item.Spec.Operation == "download" {
			add(item.Spec.Output)
		}
	}
	view.rows = nil
	for _, path := range view.paths {
		if strings.Contains(strings.ToLower(path), strings.ToLower(view.filter.Text)) {
			view.rows = append(view.rows, path)
		}
	}
	view.summary.SetText(view.desktop.tr("%d local archives · reinspection is required before upload", len(view.paths)))
	if view.listing != nil {
		view.listing.Refresh()
		if len(view.paths) == 0 {
			view.listing.Hide()
			view.empty.Show()
		} else {
			view.empty.Hide()
			view.listing.Show()
		}
	}
}

// rememberArchive persists only a successfully inspected local filename, never file contents.
func (desktop *Desktop) rememberArchive(path string) {
	paths := desktop.app.Preferences().StringList("archive-paths")
	for _, existing := range paths {
		if existing == path {
			return
		}
	}
	desktop.app.Preferences().SetStringList("archive-paths", append(paths, path))
	desktop.archives.refresh()
}
