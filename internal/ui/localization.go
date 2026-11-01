package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// tr translates a message template using the desktop's selected application language.
func (desktop *Desktop) tr(message string, arguments ...any) string {
	return desktop.catalog.Text(message, arguments...)
}

// bindText retains a static-view refresh callback and applies its initial translation.
func (desktop *Desktop) bindText(refresh func()) {
	desktop.translations = append(desktop.translations, refresh)
	refresh()
}

// label creates a translated static-view label; transient dialogs use tr directly.
func (desktop *Desktop) label(message string) *widget.Label {
	label := widget.NewLabel("")
	desktop.bindText(func() { label.SetText(desktop.tr(message)) })
	return label
}

// button creates a static-view action whose caption follows language changes.
func (desktop *Desktop) button(message string, tapped func()) *widget.Button {
	button := widget.NewButton("", tapped)
	desktop.bindText(func() { button.SetText(desktop.tr(message)) })
	return button
}

// card translates static-view titles and subtitles while retaining its content.
func (desktop *Desktop) card(title, subtitle string, content fyne.CanvasObject) *widget.Card {
	card := widget.NewCard("", "", content)
	desktop.bindText(func() { card.SetTitle(desktop.tr(title)); card.SetSubTitle(desktop.tr(subtitle)) })
	return card
}

// placeholder translates an entry prompt without modifying the user's entered text.
func (desktop *Desktop) placeholder(entry *widget.Entry, message string) {
	desktop.bindText(func() { entry.SetPlaceHolder(desktop.tr(message)) })
}

// heading returns a translated page title and descriptive introduction.
func (desktop *Desktop) heading(title, description string) fyne.CanvasObject {
	heading := widget.NewRichText()
	heading.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: "", Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Bold: true}, SizeName: theme.SizeNameHeadingText}}}
	desktop.bindText(func() { heading.Segments[0].(*widget.TextSegment).Text = desktop.tr(title); heading.Refresh() })
	text := desktop.label(description)
	text.Wrapping = fyne.TextWrapWord
	return container.NewVBox(heading, text)
}

// setStatus retains the source message so activity text can change language in place.
func (desktop *Desktop) setStatus(message string) {
	desktop.statusMessage = message
	desktop.status.SetText(desktop.translateDetails(message))
}

// translateDetails localizes application-owned lines while preserving raw diagnostics.
func (desktop *Desktop) translateDetails(message string) string {
	if translated := desktop.tr(message); translated != message {
		return translated
	}
	lines := strings.Split(message, "\n")
	for index, line := range lines {
		if translated := desktop.tr(line); translated != line {
			lines[index] = translated
			continue
		}
		parts := strings.Split(line, "; ")
		for partIndex, part := range parts {
			parts[partIndex] = desktop.tr(part)
		}
		lines[index] = strings.Join(parts, "; ")
	}
	return strings.Join(lines, "\n")
}

// showError translates safe application errors without changing persisted diagnostics.
func (desktop *Desktop) showError(err error) {
	message := desktop.translateDetails(err.Error())
	popup := dialog.NewInformation(desktop.tr("Error"), message, desktop.window)
	popup.SetDismissText(desktop.tr("OK"))
	popup.Show()
}

// inform opens a translated transient information dialog with an explicit button label.
func (desktop *Desktop) inform(title, message string) {
	popup := dialog.NewInformation(desktop.tr(title), desktop.tr(message), desktop.window)
	popup.SetDismissText(desktop.tr("OK"))
	popup.Show()
}

// confirm opens a translated transient confirmation with explicit affirmative/negative labels.
func (desktop *Desktop) confirm(title, message string, confirmed func(bool)) {
	popup := dialog.NewConfirm(desktop.tr(title), desktop.tr(message), confirmed, desktop.window)
	popup.SetConfirmText(desktop.tr("Confirm"))
	popup.SetDismissText(desktop.tr("Cancel"))
	popup.Show()
}

// changeLanguage persists selection and refreshes text without rebuilding views.
func (desktop *Desktop) changeLanguage(code string) {
	if code == desktop.language {
		return
	}
	if err := desktop.catalog.SetLanguage(code); err != nil {
		desktop.showError(err)
		return
	}
	desktop.language = code
	desktop.app.Preferences().SetString("language", code)
	for _, refresh := range desktop.translations {
		refresh()
	}
	desktop.setStatus(desktop.statusMessage)
	if desktop.taskList != nil {
		desktop.taskList.Refresh()
	}
	desktop.updateTaskDetail()
	desktop.breadcrumb.SetText(desktop.tr(pageNames[desktop.currentPage]))
	desktop.updateActivity()
	desktop.refreshRecentTasks()
	if desktop.drawerTitle != nil {
		desktop.drawerTitle.SetText(desktop.tr(desktop.drawerTitleMessage))
	}
	if desktop.drawerRefresh != nil {
		desktop.drawerRefresh()
	}
}

// languageSetting keeps bilingual options recognizable in either selected language.
func (desktop *Desktop) languageSetting() fyne.CanvasObject {
	selector := widget.NewSelect([]string{"English", "简体中文"}, nil)
	syncing := false
	desktop.bindText(func() {
		syncing = true
		option := "English"
		if desktop.language == "zh-Hans" {
			option = "简体中文"
		}
		selector.SetSelected(option)
		syncing = false
	})
	selector.OnChanged = func(value string) {
		if syncing {
			return
		}
		code := "en"
		if value == "简体中文" {
			code = "zh-Hans"
		}
		desktop.changeLanguage(code)
	}
	return desktop.card("Language", "Choose your interface language. Changes apply immediately and are remembered.", selector)
}

// form translates captions at opening; modal input blocks language changes.
func (desktop *Desktop) form(title, confirm, dismiss string, items []*widget.FormItem, callback func(bool)) *dialog.FormDialog {
	return dialog.NewForm(desktop.tr(title), desktop.tr(confirm), desktop.tr(dismiss), items, callback, desktop.window)
}

// openFile opens a file chooser with application-controlled action captions.
func (desktop *Desktop) openFile(callback func(fyne.URIReadCloser, error)) {
	chooser := dialog.NewFileOpen(callback, desktop.window)
	chooser.SetTitleText(desktop.tr("Choose file"))
	chooser.SetConfirmText(desktop.tr("Open"))
	chooser.SetDismissText(desktop.tr("Cancel"))
	chooser.Show()
}

// openFolder opens a folder chooser with application-controlled action captions.
func (desktop *Desktop) openFolder(callback func(fyne.ListableURI, error)) {
	chooser := dialog.NewFolderOpen(callback, desktop.window)
	chooser.SetTitleText(desktop.tr("Choose folder"))
	chooser.SetConfirmText(desktop.tr("Open"))
	chooser.SetDismissText(desktop.tr("Cancel"))
	chooser.Show()
}
