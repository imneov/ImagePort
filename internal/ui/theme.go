// Package ui implements ImagePort's Go desktop interface using Fyne.
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type portTheme struct {
	fyne.Theme
	variant *fyne.ThemeVariant
}

// Theme returns a neutral application theme supporting both light and dark variants.
func Theme() fyne.Theme { return portTheme{Theme: theme.DefaultTheme()} }

// Color maps semantic surfaces and actions to the approved neutral/slate palette.
func (current portTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if current.variant != nil {
		variant = *current.variant
	}
	if variant == theme.VariantDark {
		switch name {
		case theme.ColorNameBackground:
			return color.NRGBA{R: 15, G: 23, B: 42, A: 255}
		case theme.ColorNameButton:
			return color.NRGBA{R: 30, G: 41, B: 59, A: 255}
		case theme.ColorNamePrimary:
			return color.NRGBA{R: 96, G: 165, B: 250, A: 255}
		}
	} else {
		switch name {
		case theme.ColorNamePrimary:
			return color.NRGBA{R: 37, G: 99, B: 235, A: 255}
		case theme.ColorNameBackground:
			return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		case theme.ColorNameButton, theme.ColorNameInputBackground:
			return color.NRGBA{R: 248, G: 250, B: 252, A: 255}
		case theme.ColorNameSeparator, theme.ColorNameInputBorder:
			return color.NRGBA{R: 226, G: 232, B: 240, A: 255}
		case theme.ColorNameForeground:
			return color.NRGBA{R: 30, G: 41, B: 59, A: 255}
		}
	}
	return current.Theme.Color(name, variant)
}

// Size provides compact app controls and a clear page-title hierarchy.
func (current portTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return 14
	}
	if name == theme.SizeNameHeadingText {
		return 26
	}
	if name == theme.SizeNameInnerPadding {
		return 6
	}
	if name == theme.SizeNamePadding {
		return 3
	}
	return current.Theme.Size(name)
}
