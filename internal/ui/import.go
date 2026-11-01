package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
)

// importView owns archive inspection and selection state without SDK dependencies.
type importView struct {
	desktop      *Desktop
	path, target *widget.Entry
	entries      *widget.Select
	details      *widget.Label
	inspection   domain.Archive
	generation   uint64
	stage        *widget.Label
	uploadButton *widget.Button
	popup        *dialog.CustomDialog
	destination  *widget.Select
	profiles     []domain.Profile
}

// importDialog starts a fresh inspection flow; filenames may come from archive history.
func (desktop *Desktop) importDialog(filename string) {
	view := &importView{desktop: desktop, path: widget.NewEntry(), target: widget.NewEntry(), entries: widget.NewSelect(nil, nil)}
	view.path.SetPlaceHolder(desktop.tr("Choose a docker-save tar or OCI Layout tar"))
	view.target.SetPlaceHolder(desktop.tr("Target repository:tag, for example team/image:v1"))
	view.entries.PlaceHolder = desktop.tr("Inspect the archive, then select one entry")
	view.details = widget.NewLabel("")
	view.details.Wrapping = fyne.TextWrapBreak
	view.stage = widget.NewLabel(desktop.tr("1 · Choose archive   /   2 · Inspect and select   /   3 · Destination"))
	view.uploadButton = widget.NewButton(desktop.tr("Check target and upload"), view.upload)
	view.uploadButton.Importance = widget.HighImportance
	view.uploadButton.Disable()
	view.path.OnChanged = func(string) {
		view.generation++
		view.inspection = domain.Archive{}
		view.entries.ClearSelected()
		view.entries.SetOptions(nil)
		view.uploadButton.Disable()
		view.refreshDetails()
	}
	view.entries.OnChanged = func(value string) {
		view.selectEntry(value)
		if view.entries.SelectedIndex() >= 0 {
			view.uploadButton.Enable()
		}
	}
	view.refreshDetails()
	destination := view.importDestination()
	content := container.NewVBox(view.stage, widget.NewSeparator(),
		widget.NewLabel(desktop.tr("1 · Archive file")), container.NewBorder(nil, nil, nil, widget.NewButton(desktop.tr("Choose file"), view.pick), view.path),
		widget.NewLabel(desktop.tr("2 · Inspect and select an image")), widget.NewButton(desktop.tr("Inspect archive"), view.inspect), view.entries, view.details,
		widget.NewSeparator(), widget.NewLabel(desktop.tr("3 · Destination registry and tag")), destination, view.target)
	view.popup = dialog.NewCustom(desktop.tr("Import archive"), desktop.tr("Cancel"), container.NewBorder(nil, view.uploadButton, nil, nil, container.NewVScroll(content)), desktop.window)
	view.popup.SetOnClosed(func() { view.generation++ })
	view.popup.Resize(fyne.NewSize(850, 650))
	view.popup.Show()
	if filename != "" {
		view.path.SetText(filename)
		view.inspect()
	}
}

// importDestination captures independent destination profiles for this modal flow.
func (view *importView) importDestination() fyne.CanvasObject {
	view.profiles = append([]domain.Profile(nil), view.desktop.profiles...)
	view.destination = widget.NewSelect(profileOptions(view.profiles), nil)
	view.destination.PlaceHolder = view.desktop.tr("Select a registry connection")
	if index := view.desktop.profile.SelectedIndex(); index >= 0 {
		view.destination.SetSelectedIndex(index)
	}
	return view.destination
}

// destinationProfile resolves the modal's selected connection without consulting source state.
func (view *importView) destinationProfile() (domain.Profile, bool) {
	index := view.destination.SelectedIndex()
	if index < 0 || index >= len(view.profiles) {
		return domain.Profile{}, false
	}
	return view.profiles[index], true
}

// selectEntry displays all platforms within the explicitly selected root.
func (view *importView) selectEntry(string) {
	index := view.entries.SelectedIndex()
	if index < 0 || index >= len(view.inspection.Entries) {
		return
	}
	entry := view.inspection.Entries[index]
	name := entry.Name
	if name == "" {
		name = entry.Digest
	}
	view.details.SetText(view.desktop.tr("Format: %s\nEntry: %s\nDigest: %s\nPlatforms: %s\nFile SHA256: %s\nFile size: %.1f MiB",
		view.inspection.Format, name, entry.Digest, strings.Join(entry.Platforms, ", "), view.inspection.SHA256, float64(view.inspection.Size)/(1<<20)))
}

// pick obtains a local filename and closes the chooser's reader before inspection.
func (view *importView) pick() {
	view.desktop.openFile(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			view.desktop.showError(err)
			return
		}
		if reader == nil {
			return
		}
		filename := reader.URI().Path()
		if err := reader.Close(); err != nil {
			view.desktop.showError(err)
			return
		}
		view.path.SetText(filename)
	})
}

// inspect ignores stale results and never silently selects the first archive entry.
func (view *importView) inspect() {
	filename := view.path.Text
	view.generation++
	generation := view.generation
	view.desktop.runCurrent("Checking archive contents, platforms, and digests...", func() bool { return generation == view.generation }, func(ctx context.Context) (any, error) {
		return archive.Inspect(ctx, filename, view.desktop.presenter.Limits)
	}, func(result any) {
		if generation != view.generation {
			return
		}
		view.inspection = result.(domain.Archive)
		view.desktop.rememberArchive(view.inspection.Path)
		labels := make([]string, len(view.inspection.Entries))
		for index, entry := range view.inspection.Entries {
			name := entry.Name
			if name == "" {
				name = entry.Digest
			}
			labels[index] = fmt.Sprintf("%d · %s · %s", index+1, name, strings.Join(entry.Platforms, ", "))
		}
		view.entries.ClearSelected()
		view.entries.SetOptions(labels)
		view.refreshDetails()
	})
}

// upload captures one validated root and asks for renewed target/overwrite consent.
func (view *importView) upload() {
	profile, valid := view.destinationProfile()
	if !valid {
		view.desktop.inform("Select a connection", "Select a registry on Images, or add a connection in Registries.")
		return
	}
	index := view.entries.SelectedIndex()
	if view.inspection.SHA256 == "" || index < 0 || index >= len(view.inspection.Entries) {
		view.desktop.showError(errors.New("Inspect the archive and select an image entry first"))
		return
	}
	view.desktop.confirmRemoteThen(profile, domain.Spec{Operation: "import", ProfileID: profile.ID, Archive: view.inspection, Entry: view.inspection.Entries[index], Target: view.target.Text}, view.popup.Hide)
}

// refreshDetails re-renders inspected data without changing the selected root.
func (view *importView) refreshDetails() {
	if view.entries.SelectedIndex() >= 0 {
		view.selectEntry("")
		return
	}
	if view.inspection.SHA256 != "" {
		view.details.SetText(view.desktop.tr("Found %d entries; select one. All platforms in the selected entry will be preserved.", len(view.inspection.Entries)))
	} else {
		view.details.SetText(view.desktop.tr("No archive inspected. Ordinary directories and container root filesystem tars are not supported."))
	}
}
