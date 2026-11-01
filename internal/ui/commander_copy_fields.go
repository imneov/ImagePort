package ui

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// contentChoices offers all remote platforms while requiring a single platform for engine loads.
func (review *copyReview) contentChoices() {
	desktop := review.workspace.desktop
	if review.source.target.Kind == "filesystem" {
		options := make([]string, len(review.source.archive.Entries))
		for index, entry := range review.source.archive.Entries {
			options[index] = entry.Name
			if entry.Name == "" {
				options[index] = desktop.tr("Unnamed image")
			}
		}
		review.entry.SetOptions(options)
		review.entry.PlaceHolder = desktop.tr("Choose an image inside the archive")
		review.entry.OnChanged = func(string) { review.selectArchiveEntry() }
		if len(options) == 1 {
			review.entry.SetSelectedIndex(0)
		}
		return
	}
	options := []string{}
	if len(review.source.sources) > 0 && review.source.sources[0].IndexDigest != "" && review.target.Kind != "engine" {
		options = append(options, desktop.tr("All source platforms"))
	}
	for _, source := range review.source.sources {
		options = append(options, source.Platform)
	}
	if review.source.local.Platform != "" {
		options = []string{review.source.local.Platform}
	}
	review.platform.SetOptions(options)
	review.platform.PlaceHolder = desktop.tr("Choose a platform")
	if len(options) == 1 || review.target.Kind != "engine" {
		review.platform.SetSelectedIndex(0)
	}
}

// selectArchiveEntry updates the exact selected root and prevents implicit multi-entry imports.
func (review *copyReview) selectArchiveEntry() {
	index := review.entry.SelectedIndex()
	if index < 0 || index >= len(review.source.archive.Entries) {
		return
	}
	entry := review.source.archive.Entries[index]
	name, tag := imageNames(entry.Name)
	if tag == "" {
		name = strings.TrimSuffix(filepath.Base(review.source.archive.Path), ".tar")
		tag = entry.Name
	}
	review.name.SetText(name)
	review.tag.SetText(tag)
	if review.target.Kind == "engine" {
		review.platform.SetOptions(entry.Platforms)
		review.platform.ClearSelected()
		if len(entry.Platforms) == 1 {
			review.platform.SetSelectedIndex(0)
		}
	}
	if review.preview != nil {
		review.invalidate()
	}
}

// fields includes only inputs that affect the current transfer route.
func (review *copyReview) fields() fyne.CanvasObject {
	desktop := review.workspace.desktop
	fields := widget.NewForm()
	if review.target.Kind == "registry" {
		review.project.SetPlaceHolder(desktop.tr("Choose a destination project"))
		fields.Append(desktop.tr("Project"), review.projectChoice)
	}
	if review.source.target.Kind == "filesystem" {
		fields.Append(desktop.tr("Image inside archive"), review.entry)
	}
	if review.target.Kind == "filesystem" {
		fields.Append(desktop.tr("File name"), review.filename)
	} else {
		fields.Append(desktop.tr("Image name"), review.name)
		fields.Append(desktop.tr("Version name (tag)"), review.tag)
	}
	if review.source.target.Kind != "filesystem" || review.target.Kind == "engine" {
		fields.Append(desktop.tr("Platform"), review.platform)
	}
	if review.target.Kind == "registry" {
		signIn := widget.NewButton(desktop.tr("Sign in"), func() {
			review.workspace.accountDialog(review.target, review.workspace.account(review.accountID), func(profile domain.Profile) {
				review.accountID = profile.ID
				review.reloadAccounts()
				review.invalidate()
			})
		})
		fields.Append(desktop.tr("Destination account"), container.NewBorder(nil, nil, nil, signIn, review.account))
	}
	return fields
}

// reloadAccounts uses stable IDs for the destination's independent access choice.
func (review *copyReview) reloadAccounts() {
	accounts := review.workspace.paneAccounts(review.target.ID)
	options := []string{review.workspace.desktop.tr("Anonymous")}
	review.accountIDs = []string{""}
	selected := 0
	labels := accountOptions(accounts)
	for index, profile := range accounts {
		if profile.Username == "" {
			continue
		}
		options = append(options, labels[index])
		review.accountIDs = append(review.accountIDs, profile.ID)
		if profile.ID == review.accountID {
			selected = len(options) - 1
		}
	}
	review.account.OnChanged = nil
	review.account.SetOptions(options)
	review.account.SetSelectedIndex(selected)
	review.account.OnChanged = func(string) {
		review.accountID = review.accountIDs[review.account.SelectedIndex()]
		review.invalidate()
	}
}

// bindChanges revokes overwrite consent whenever naming, platform or content changes.
func (review *copyReview) bindChanges() {
	for _, entry := range []*widget.Entry{review.project, review.name, review.tag, review.filename} {
		entry.OnChanged = func(string) { review.invalidate() }
	}
	review.platform.OnChanged = func(string) { review.invalidate() }
}

// invalidate clears observed target state so an earlier consent cannot authorize a new target.
func (review *copyReview) invalidate() {
	review.revision++
	review.observedReference, review.previous = "", ""
	review.replace.SetChecked(false)
	review.replace.Hide()
	review.feedback.Hide()
	review.refreshPreview()
}

// refreshPreview displays the exact reference or file path that the confirmation will affect.
func (review *copyReview) refreshPreview() {
	if review.preview == nil {
		return
	}
	reference := review.name.Text + ":" + review.tag.Text
	if review.target.Kind == "registry" {
		reference = strings.TrimPrefix(strings.TrimPrefix(review.target.Address, "https://"), "http://") + "/" + review.project.Text + "/" + reference
	}
	if review.target.Kind == "filesystem" {
		reference = filepath.Join(review.outputDirectory, review.filename.Text)
	}
	review.preview.SetText(reference)
}
