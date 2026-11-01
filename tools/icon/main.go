// Command icon regenerates the application icon from its checked-in Go source.
package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

// main writes a deterministic icon; run from the repository root.
func main() {
	if err := generate("assets/icon.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// generate draws a stack of image layers using the application's visual palette.
func generate(filename string) error {
	canvas := image.NewRGBA(image.Rect(0, 0, 512, 512))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{20, 39, 57, 255}), image.Point{}, draw.Src)
	for _, layer := range []struct {
		top   int
		shade color.RGBA
	}{
		{300, color.RGBA{15, 119, 116, 255}}, {216, color.RGBA{21, 156, 147, 255}}, {132, color.RGBA{109, 226, 204, 255}},
	} {
		draw.Draw(canvas, image.Rect(96, layer.top, 416, layer.top+64), image.NewUniform(layer.shade), image.Point{}, draw.Src)
		draw.Draw(canvas, image.Rect(124, layer.top+22, 144, layer.top+42), image.NewUniform(color.White), image.Point{}, draw.Src)
	}
	if err := os.MkdirAll("assets", 0755); err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	return errors.Join(png.Encode(file, canvas), file.Close())
}
