package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// localImagesView retains the active engine and its actual complete image listing.
type localImagesView struct {
	desktop         *Desktop
	engine          domain.Engine
	images, rows    []domain.LocalImage
	filter          *widget.Entry
	listing         *widget.List
	status, summary *widget.Label
	generation      uint64
	loaded          bool
	failure         string
}

// localImagesPage exposes refresh, local filtering, tagging and registry push actions.
func (desktop *Desktop) localImagesPage() fyne.CanvasObject {
	view := &localImagesView{desktop: desktop, filter: widget.NewEntry(), status: widget.NewLabel(""), summary: widget.NewLabel("")}
	desktop.localImages = view
	desktop.placeholder(view.filter, "Filter local image tags or IDs")
	view.filter.OnChanged = func(string) { view.render() }
	view.status.Wrapping = fyne.TextWrapWord
	view.listing = widget.NewList(func() int { return len(view.rows) }, view.createRow, view.updateRow)
	view.listing.OnSelected = func(index widget.ListItemID) { view.open(view.rows[index]) }
	refresh := desktop.button("Refresh local images", view.refresh)
	refresh.SetIcon(theme.ViewRefreshIcon())
	refresh.Importance = widget.HighImportance
	title := container.NewBorder(nil, nil, nil, container.NewCenter(refresh), desktop.heading("Local images", "Choose an image to copy. Its original tags are retained."))
	top := container.NewVBox(title, view.status, view.filter, widget.NewSeparator())
	headers := columns([]float32{5, 3, 2, 2}, desktop.label("Image tags"), desktop.label("Local image ID"), desktop.label("Size"), desktop.label("Actions"))
	desktop.bindText(view.render)
	return container.NewBorder(top, view.summary, nil, nil, container.NewBorder(headers, nil, nil, nil, view.listing))
}

// refresh reads the configured Docker context without assuming a daemon is running.
func (view *localImagesView) refresh() {
	view.generation++
	generation := view.generation
	view.status.SetText(view.desktop.tr("Reading local Docker images..."))
	type response struct {
		images domain.LocalImages
		err    error
	}
	view.desktop.runCurrent("Reading local Docker images...", func() bool { return generation == view.generation }, func(ctx context.Context) (any, error) {
		images, err := view.desktop.presenter.Local.Images(ctx)
		return response{images, err}, nil
	}, func(result any) {
		value := result.(response)
		view.engine, view.images, view.loaded = value.images.Engine, value.images.Images, true
		view.failure = ""
		if value.err != nil {
			view.failure = value.err.Error()
		}
		view.render()
	})
}

// render filters the complete local response and distinguishes an empty engine from failure.
func (view *localImagesView) render() {
	query := strings.ToLower(view.filter.Text)
	view.rows = nil
	for _, image := range view.images {
		if strings.Contains(strings.ToLower(strings.Join(image.Tags, " ")+" "+image.ID), query) {
			view.rows = append(view.rows, image)
		}
	}
	if view.listing == nil {
		return
	}
	view.listing.UnselectAll()
	view.listing.Refresh()
	message := view.desktop.tr("Refresh to connect to the Docker context configured on this machine.")
	if view.loaded {
		message = view.desktop.tr("Docker context: %s · %s", view.engine.Context, view.engine.Host)
		if len(view.images) == 0 {
			message += "\n" + view.desktop.tr("No local images found. Build or pull an image, then refresh.")
		}
	}
	if view.failure != "" {
		message = view.desktop.translateDetails(view.failure)
	}
	view.status.SetText(message)
	view.summary.SetText(view.desktop.tr("%d visible / %d local images", len(view.rows), len(view.images)))
}

// createRow creates virtualized local image metadata and an explicit detail action.
func (view *localImagesView) createRow() fyne.CanvasObject {
	name, identity, size := widget.NewLabel(""), widget.NewLabel(""), widget.NewLabel("")
	name.Truncation, identity.Truncation = fyne.TextTruncateEllipsis, fyne.TextTruncateEllipsis
	return columns([]float32{5, 3, 2, 2}, name, identity, size, widget.NewButtonWithIcon("", theme.NavigateNextIcon(), nil))
}

// updateRow binds the selected image value rather than a recycled widget's index.
func (view *localImagesView) updateRow(index widget.ListItemID, object fyne.CanvasObject) {
	image := view.rows[index]
	cells := object.(*fyne.Container).Objects
	name := strings.Join(image.Tags, ", ")
	if name == "" {
		name = view.desktop.tr("Untagged image")
	}
	cells[0].(*widget.Label).SetText(name)
	cells[1].(*widget.Label).SetText(image.ID)
	cells[2].(*widget.Label).SetText(fmt.Sprintf("%.1f MiB", float64(image.Size)/(1024*1024)))
	cells[3].(*widget.Button).OnTapped = func() { view.open(image) }
}

// open inspects the selected immutable ID before setting the copy source.
func (view *localImagesView) open(image domain.LocalImage) {
	view.generation++
	view.desktop.copyView.local, view.desktop.copyView.source = domain.LocalSource{}, domain.Source{}
	view.desktop.copyView.updatePreview()
	engine, generation := view.engine, view.generation
	view.desktop.runCurrent("Inspecting local image...", func() bool { return generation == view.generation && view.desktop.currentPage == imagesPage }, func(ctx context.Context) (any, error) {
		return view.desktop.presenter.Local.Inspect(ctx, engine, image.ID)
	}, func(result any) {
		selected := result.(domain.LocalImage)
		view.desktop.copyView.selectLocal(engine, selected)
	})
}

// localImageDetails exposes optional metadata and local alias management outside the copy flow.
func (desktop *Desktop) localImageDetails(engine domain.Engine, image domain.LocalImage) {
	contextLabel, idLabel, tagsLabel, platformLabel := widget.NewLabel(""), widget.NewLabel(""), widget.NewLabel(""), widget.NewLabel("")
	identity := widget.NewLabel(image.ID)
	identity.Wrapping = fyne.TextWrapBreak
	tags := widget.NewLabel(strings.Join(image.Tags, "\n"))
	tags.Wrapping = fyne.TextWrapBreak
	content := container.NewVBox(contextLabel, idLabel, identity, tagsLabel, tags, platformLabel, widget.NewLabel(image.Platform))
	tag := widget.NewButton(desktop.tr("Add tag"), func() { desktop.localTagDialog(engine, image) })
	desktop.openDrawer("Local image details", content, tag)
	desktop.drawerRefresh = func() {
		contextLabel.SetText(desktop.tr("Docker context: %s · %s", engine.Context, engine.Host))
		idLabel.SetText(desktop.tr("Local image ID"))
		tagsLabel.SetText(desktop.tr("Image tags"))
		platformLabel.SetText(desktop.tr("Platform"))
		tag.SetText(desktop.tr("Add tag"))
	}
	desktop.drawerRefresh()
}
