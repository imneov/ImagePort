package ui

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// imageDetails owns platform consent independently of the local machine's architecture.
type imageDetails struct {
	desktop       *Desktop
	profile       domain.Profile
	sources       []domain.Source
	selected      int
	labels        []string
	details       *widget.Label
	notice        *widget.Label
	copyReference *widget.Button
	useImage      *widget.Button
}

// selectSource opens contextual image details with no platform selected, even for one result.
func (desktop *Desktop) selectSource(profile domain.Profile, sources []domain.Source) {
	if len(sources) == 0 {
		desktop.showError(errors.New("No platforms are available"))
		return
	}
	view := &imageDetails{desktop: desktop, profile: profile, sources: sources, selected: -1, labels: sourceLabels(sources), details: widget.NewLabel("")}
	view.details.Wrapping = fyne.TextWrapBreak
	platform := widget.NewRadioGroup(view.labels, view.choose)
	platform.Required = true
	view.useImage = widget.NewButtonWithIcon("", theme.ConfirmIcon(), func() {
		if view.selected >= 0 {
			desktop.copyView.selectRemote(profile, sources[view.selected])
		}
	})
	view.useImage.Importance = widget.HighImportance
	view.useImage.Disable()
	content := container.NewVScroll(container.NewVBox(view.summary(), platform, widget.NewSeparator(), view.details))
	desktop.openDrawer("Image details", content, view.useImage)
	desktop.drawerRefresh = view.refresh
	view.refresh()
}

// choose resets consent before matching a radio value, including an explicit deselection.
func (view *imageDetails) choose(value string) {
	view.selected = -1
	for index, label := range view.labels {
		if label == value {
			view.selected = index
		}
	}
	if view.selected < 0 {
		view.useImage.Disable()
	} else {
		view.useImage.Enable()
	}
	view.refresh()
}

// refresh renders selected immutable metadata without changing platform consent.
func (view *imageDetails) refresh() {
	view.notice.SetText(view.desktop.tr("Select the platform to manage. The client does not choose based on the local CPU."))
	view.copyReference.SetText(view.desktop.tr("Copy reference"))
	view.useImage.SetText(view.desktop.tr("Use selected image"))
	if view.selected < 0 {
		view.details.SetText(view.desktop.tr("Select a platform explicitly"))
		return
	}
	source := view.sources[view.selected]
	view.details.SetText(view.desktop.tr("Source: %s\nSource index: %s\nSelected manifest: %s\nPlatform: %s\nKnown content size: %.1f MiB (excluding archive overhead)", source.Reference, source.IndexDigest, source.Digest, source.Platform, float64(source.Size)/(1<<20)))
}

// shortDigest preserves the hash algorithm and a recognizable prefix for compact rows.
func shortDigest(value string) string {
	if len(value) > 23 {
		return value[:23] + "…"
	}
	return value
}

// summary displays an immutable reference and explains explicit platform selection.
func (view *imageDetails) summary() fyne.CanvasObject {
	desktop, source := view.desktop, view.sources[0]
	reference := widget.NewLabel(source.Reference)
	reference.Wrapping = fyne.TextWrapBreak
	clipboard := widget.NewButtonWithIcon(desktop.tr("Copy reference"), theme.ContentCopyIcon(), func() { desktop.window.Clipboard().SetContent(source.Reference) })
	notice := widget.NewLabel(desktop.tr("Select the platform to manage. The client does not choose based on the local CPU."))
	notice.Wrapping = fyne.TextWrapWord
	view.notice, view.copyReference = notice, clipboard
	return container.NewVBox(reference, clipboard, widget.NewSeparator(), notice)
}

// sourceLabels associates each radio option with one resolved manifest and platform.
func sourceLabels(sources []domain.Source) []string {
	labels := make([]string, len(sources))
	for index, source := range sources {
		labels[index] = source.Platform + " · " + shortDigest(source.Digest)
	}
	return labels
}
