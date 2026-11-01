package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
)

// inset returns content with equal padding on all four sides, measured in logical pixels.
func inset(content fyne.CanvasObject, padding float32) fyne.CanvasObject {
	return container.New(layout.NewCustomPaddedLayout(padding, padding, padding, padding), content)
}

// sidebarLayout reserves a consistent width without imposing its children's minimum height.
type sidebarLayout struct{ width float32 }

// Layout fills size with the sidebar's single content object.
func (current *sidebarLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, object := range objects {
		object.Move(fyne.NewPos(0, 0))
		object.Resize(size)
	}
}

// MinSize returns the sidebar width; the workspace determines the window's height.
func (current *sidebarLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(current.width, 0)
}

// columnsLayout aligns reusable list rows and column headings using relative widths.
type columnsLayout struct{ weights []float32 }

// Layout distributes the available row width according to configured column weights.
func (current *columnsLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	total := float32(0)
	for _, weight := range current.weights {
		total += weight
	}
	offset := float32(0)
	for index, object := range objects {
		width := size.Width * current.weights[index] / total
		object.Move(fyne.NewPos(offset, 0))
		object.Resize(fyne.NewSize(width, size.Height))
		offset += width
	}
}

// MinSize gives virtual list rows a stable readable height without long-text overflow.
func (current *columnsLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(400, 64) }

// columns creates an aligned row; callers must supply one weight per object.
func columns(weights []float32, objects ...fyne.CanvasObject) *fyne.Container {
	return container.New(&columnsLayout{weights: weights}, objects...)
}
