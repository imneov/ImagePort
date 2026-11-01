package ui

import (
	"fyne.io/fyne/v2"
	"github.com/imneov/ImagePort/assets/fonts"
)

var regularFont = fyne.NewStaticResource("NotoSansCJKsc-Regular.otf", fonts.Regular)
var boldFont = fyne.NewStaticResource("NotoSansCJKsc-Bold.otf", fonts.Bold)

// Font supplies Chinese and Latin glyphs without relying on installed system fonts.
// Symbol and monospace styles retain the toolkit's specialized faces.
func (current portTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Symbol || style.Monospace {
		return current.Theme.Font(style)
	}
	if style.Bold {
		return boldFont
	}
	return regularFont
}
