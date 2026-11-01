package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// imageCopyView owns one immutable selection and independent source/destination locations.
type imageCopyView struct {
	desktop                              *Desktop
	sourceLocation, destination          *widget.Select
	sourceBody, targetBody               *fyne.Container
	localBody, remoteBody                fyne.CanvasObject
	selected, preview, hint              *widget.Label
	repository, tag, directory, filename *widget.Entry
	copy                                 *widget.Button
	local                                domain.LocalSource
	localImage                           domain.LocalImage
	source                               domain.Source
	sourceProfile                        domain.Profile
	profileIDs                           []string
	updating                             bool
}

// registryPage presents copying between locations as the primary image workflow.
func (desktop *Desktop) registryPage() fyne.CanvasObject {
	view := &imageCopyView{desktop: desktop, selected: widget.NewLabel(""), preview: widget.NewLabel(""), hint: widget.NewLabel("")}
	desktop.copyView = view
	view.sourceLocation = widget.NewSelect(nil, view.changeLocation)
	view.destination = widget.NewSelect(nil, func(string) { view.renderDestination() })
	desktop.bindText(func() { view.destination.PlaceHolder = desktop.tr("Choose a destination"); view.destination.Refresh() })
	view.localBody, view.remoteBody = desktop.localImagesPage(), desktop.remoteImagesPage()
	view.sourceBody = container.NewStack(view.localBody)
	view.selected.Wrapping, view.preview.Wrapping, view.hint.Wrapping = fyne.TextWrapBreak, fyne.TextWrapBreak, fyne.TextWrapWord
	view.copy = desktop.button("Copy image", view.submit)
	view.copy.SetIcon(theme.ContentCopyIcon())
	view.copy.Importance = widget.HighImportance
	view.buildTarget()
	left := container.NewBorder(container.NewVBox(desktop.label("From"), view.sourceLocation, widget.NewSeparator()), nil, nil, nil, view.sourceBody)
	actions := container.NewVBox(view.copy, desktop.button("Add registry", func() { desktop.profileDialog(domain.Profile{}) }))
	right := container.NewBorder(container.NewVBox(desktop.label("To"), view.destination, widget.NewSeparator()), actions, nil, nil, container.NewVScroll(view.targetBody))
	split := container.NewHSplit(left, inset(right, 12))
	split.Offset = .62
	heading := desktop.heading("Copy images", "Choose an image, choose its destination, then copy.")
	desktop.bindText(view.reloadLocations)
	return container.NewBorder(heading, nil, nil, nil, split)
}

// reloadLocations refreshes translated location names while retaining profile IDs and selection.
func (view *imageCopyView) reloadLocations() {
	if view.sourceLocation == nil {
		return
	}
	sourceID, targetID := view.locationID(view.sourceLocation.SelectedIndex()-1), view.locationID(view.destination.SelectedIndex())
	archive := view.isArchive()
	view.updating = true
	options := []string{view.desktop.tr("Local Docker images")}
	view.profileIDs = nil
	for _, profile := range view.desktop.profiles {
		view.profileIDs = append(view.profileIDs, profile.ID)

	}
	options = append(options, profileOptions(view.desktop.profiles)...)
	view.sourceLocation.SetOptions(options)
	view.sourceLocation.SetSelectedIndex(view.profileIndex(sourceID) + 1)
	view.destination.SetOptions(view.destinationOptions(options[1:]))
	selected := view.profileIndex(targetID)
	if archive && view.sourceProfile.ID != "" {
		selected = len(view.profileIDs)
	}
	if selected < 0 && len(view.destination.Options) > 0 {
		selected = 0
	}
	if selected >= 0 {
		view.destination.SetSelectedIndex(selected)
	} else {
		view.destination.ClearSelected()
	}
	view.updating = false
	view.renderDestination()
}

// locationID returns the stable saved profile ID for a selector index, or empty for local.
func (view *imageCopyView) locationID(index int) string {
	if index < 0 || index >= len(view.profileIDs) {
		return ""
	}
	return view.profileIDs[index]
}

// profileIndex looks up a saved ID without depending on translated display strings.
func (view *imageCopyView) profileIndex(id string) int {
	for index, value := range view.profileIDs {
		if value == id {
			return index
		}
	}
	return -1
}

// destinationOptions offers archive output only when an immutable remote source exists.
func (view *imageCopyView) destinationOptions(profiles []string) []string {
	options := append([]string(nil), profiles...)
	if view.sourceProfile.ID != "" {
		options = append(options, view.desktop.tr("Local archive (.tar)"))
	}
	return options
}

// changeLocation invalidates selections and pending reads before displaying another source.
func (view *imageCopyView) changeLocation(string) {
	if view.updating {
		return
	}
	view.desktop.closeDrawer()
	view.desktop.browser.generation++
	view.desktop.localImages.generation++
	view.local, view.source, view.sourceProfile = domain.LocalSource{}, domain.Source{}, domain.Profile{}
	index := view.sourceLocation.SelectedIndex()
	view.sourceBody.Objects = []fyne.CanvasObject{view.localBody}
	if index > 0 {
		view.sourceBody.Objects = []fyne.CanvasObject{view.remoteBody}
		view.desktop.profile.SetSelectedIndex(index - 1)
		view.desktop.browser.browse()
	} else {
		view.desktop.localImages.refresh()
	}
	view.sourceBody.Refresh()
	view.reloadLocations()
}

// showRemoteImages selects the profile's source location for navigation commands.
func (desktop *Desktop) showRemoteImages() {
	desktop.show(imagesPage)
	index := desktop.profile.SelectedIndex()
	if index >= 0 {
		desktop.copyView.sourceLocation.SetSelectedIndex(index + 1)
	}
}

// selectLocal pins the image ID and engine without changing any daemon tags.
func (view *imageCopyView) selectLocal(engine domain.Engine, image domain.LocalImage) {
	view.local = domain.LocalSource{Engine: engine, ImageID: image.ID}
	view.localImage = image
	view.sourceProfile = domain.Profile{}
	view.source = domain.Source{Reference: strings.Join(image.Tags, ", "), Platform: image.Platform}
	view.suggestName("")
	if len(image.Tags) > 0 {
		view.suggestName(image.Tags[0])
	}
	view.reloadLocations()
}

// selectRemote retains the explicitly selected platform and its immutable manifest.
func (view *imageCopyView) selectRemote(profile domain.Profile, source domain.Source) {
	view.local, view.sourceProfile, view.source = domain.LocalSource{}, profile, source
	view.suggestName(source.Reference)
	view.filename.SetText("image-" + strings.ReplaceAll(strings.Split(source.Platform, ",")[0], "/", "-") + ".tar")
	view.reloadLocations()
	view.desktop.closeDrawer()
}

// suggestName derives editable defaults only from an actual selected image reference.
func (view *imageCopyView) suggestName(reference string) {
	reference, _, _ = strings.Cut(reference, "@")
	first, rest, full := strings.Cut(reference, "/")
	if full && (strings.ContainsAny(first, ".:") || first == "localhost") {
		reference = rest
	}
	name, tag := reference, ""
	if split := strings.LastIndex(reference, ":"); split > strings.LastIndex(reference, "/") {
		name, tag = reference[:split], reference[split+1:]
	}
	view.repository.SetText(name)
	view.tag.SetText(tag)
}
