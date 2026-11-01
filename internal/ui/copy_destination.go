package ui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// buildTarget creates persistent editable names; validation errors never discard these inputs.
func (view *imageCopyView) buildTarget() {
	desktop := view.desktop
	view.repository, view.tag = widget.NewEntry(), widget.NewEntry()
	view.directory, view.filename = widget.NewEntry(), widget.NewEntry()
	desktop.placeholder(view.repository, "project/image")
	desktop.placeholder(view.tag, "Destination tag")
	desktop.placeholder(view.directory, "Destination folder")
	desktop.placeholder(view.filename, "Filename")
	view.directory.SetText(desktop.app.Preferences().String("download-directory"))
	for _, entry := range []*widget.Entry{view.repository, view.tag, view.directory, view.filename} {
		entry.OnChanged = func(string) { view.updatePreview() }
	}
	view.targetBody = container.NewVBox()
}

// isArchive reports the special destination after saved registry options.
func (view *imageCopyView) isArchive() bool {
	return view.sourceProfile.ID != "" && view.destination.SelectedIndex() == len(view.profileIDs)
}

// renderDestination shows only the fields relevant to the selected kind of location.
func (view *imageCopyView) renderDestination() {
	if view.updating || view.targetBody == nil {
		return
	}
	desktop := view.desktop
	fields := []fyne.CanvasObject{desktop.copyLabel("Selected image"), view.selected, widget.NewSeparator()}
	if view.local.ImageID != "" {
		fields = append(fields, widget.NewButton(desktop.tr("Image details"), func() { desktop.localImageDetails(view.local.Engine, view.localImage) }))
	}
	if view.isArchive() {
		pick := widget.NewButton(desktop.tr("Choose folder"), view.chooseFolder)
		fields = append(fields, desktop.copyLabel("Destination folder"), container.NewBorder(nil, nil, nil, pick, view.directory), desktop.copyLabel("Filename"), view.filename)
	} else {
		fields = append(fields, desktop.copyLabel("Repository name"), view.repository, desktop.copyLabel("Destination tag"), view.tag)
	}
	fields = append(fields, desktop.copyLabel("Destination"), view.preview, view.hint)
	view.targetBody.Objects = fields
	view.targetBody.Refresh()
	view.updatePreview()
}

// copyLabel translates a rebuilt destination label without registering stale widgets.
func (desktop *Desktop) copyLabel(message string) *widget.Label {
	return widget.NewLabel(desktop.tr(message))
}

// updatePreview makes the exact copy endpoints visible and gates incomplete input.
func (view *imageCopyView) updatePreview() {
	if view.copy == nil {
		return
	}
	desktop := view.desktop
	view.copy.Disable()
	view.selected.SetText(desktop.tr("Choose an image from the source location."))
	view.preview.SetText("")
	view.hint.SetText(desktop.tr("The source image and its existing tags are retained."))
	selected := view.local.ImageID != "" || view.source.Digest != ""
	if selected {
		identity := view.source.Digest
		if view.local.ImageID != "" {
			identity = view.local.ImageID
		}
		view.selected.SetText(view.source.Reference + "\n" + view.source.Platform + "\n" + identity)
	}
	if view.isArchive() {
		view.preview.SetText(filepath.Join(view.directory.Text, view.filename.Text))
		if selected && strings.TrimSpace(view.directory.Text) != "" && strings.TrimSpace(view.filename.Text) != "" {
			view.copy.Enable()
		}
		return
	}
	index := view.destination.SelectedIndex()
	if index < 0 || index >= len(desktop.profiles) {
		return
	}
	view.preview.SetText(desktop.profiles[index].Host)
	if view.repository.Text != "" {
		view.preview.SetText(desktop.profiles[index].Host + "/" + view.repository.Text + ":" + view.tag.Text)
	}
	if selected && strings.TrimSpace(view.repository.Text) != "" && strings.TrimSpace(view.tag.Text) != "" {
		view.copy.Enable()
	}
}

// chooseFolder uses a native-independent Fyne chooser without creating an output file.
func (view *imageCopyView) chooseFolder() {
	view.desktop.openFolder(func(uri fyne.ListableURI, err error) {
		if err != nil {
			view.desktop.showError(err)
			return
		}
		if uri != nil {
			view.directory.SetText(uri.Path())
		}
	})
}

// submit translates copy intent into a pinned task after validating explicit destination parts.
func (view *imageCopyView) submit() {
	if view.local.ImageID == "" && view.source.Digest == "" {
		return
	}
	if view.isArchive() {
		view.submitArchive()
		return
	}
	index := view.destination.SelectedIndex()
	if index < 0 || index >= len(view.desktop.profiles) {
		return
	}
	name, tag := strings.TrimSpace(view.repository.Text), strings.TrimSpace(view.tag.Text)
	if name == "" || tag == "" || strings.ContainsAny(name, ":@") || strings.ContainsAny(tag, ":/@") {
		view.desktop.showError(errors.New("Enter a repository path and a separate tag"))
		return
	}
	profile := view.desktop.profiles[index]
	spec := domain.Spec{Operation: "copy", ProfileID: profile.ID, Target: name + ":" + tag, Source: view.source}
	if view.local.ImageID != "" {
		spec.Operation, spec.Local = "push", view.local
	} else {
		spec.SourceProfileID, spec.SourceRegistryHost = view.sourceProfile.ID, view.sourceProfile.Host
	}
	view.desktop.confirmRemote(profile, spec)
}

// submitArchive queues an explicitly named OCI tar with the same selected source identity.
func (view *imageCopyView) submitArchive() {
	name, directory := strings.TrimSpace(view.filename.Text), strings.TrimSpace(view.directory.Text)
	if directory == "" || name == "" || filepath.Base(name) != name || strings.ContainsAny(name, "\\/") {
		view.desktop.showError(errors.New("Choose a folder and enter a filename without path separators"))
		return
	}
	profile, source, output := view.sourceProfile, view.source, filepath.Join(directory, name)
	view.desktop.run("Queueing download...", func(context.Context) (any, error) {
		return view.desktop.presenter.Download(profile, source, output)
	}, func(any) { view.desktop.show(transfersPage) })
}
