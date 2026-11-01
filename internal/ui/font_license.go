package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/assets/fonts"
)

// fontLicense makes the bundled font's copyright and license available in distributed applications.
func (desktop *Desktop) fontLicense() {
	text := widget.NewLabel(string(fonts.License))
	text.Wrapping = fyne.TextWrapWord
	popup := dialog.NewCustom(desktop.tr("Font license"), desktop.tr("Close"), container.NewVScroll(text), desktop.window)
	popup.Resize(fyne.NewSize(760, 600))
	popup.Show()
}
