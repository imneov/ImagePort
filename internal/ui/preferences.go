package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// appearanceSetting exposes the same persistent theme preference as the toolbar.
func (desktop *Desktop) appearanceSetting() fyne.CanvasObject {
	change := desktop.button("Toggle light / dark", desktop.toggleTheme)
	return desktop.card("Appearance", "Switch between light and dark themes.", change)
}

// downloadSetting stores a user-selected default folder without creating or truncating files.
func (desktop *Desktop) downloadSetting() fyne.CanvasObject {
	directory := widget.NewEntry()
	desktop.placeholder(directory, "Destination folder")
	directory.SetText(desktop.app.Preferences().String("download-directory"))
	directory.OnChanged = func(path string) { desktop.app.Preferences().SetString("download-directory", path) }
	pick := desktop.button("Choose folder", func() {
		desktop.openFolder(func(uri fyne.ListableURI, err error) {
			if err != nil {
				desktop.showError(err)
				return
			}
			if uri != nil {
				directory.SetText(uri.Path())
			}
		})
	})
	return desktop.card("Default download folder", "Each download can choose a different destination.", container.NewBorder(nil, nil, nil, pick, directory))
}
