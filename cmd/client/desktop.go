//go:build !ci

package main

import (
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/imneov/ImagePort/internal/presenter"
)

// newApplication initializes the native Go/Fyne desktop driver.
func newApplication() fyne.App { return app.NewWithID("io.github.imneov.ImagePort") }

// waitForSmoke rejects synchronous capture on the native event thread; use the official CI renderer.
func waitForSmoke(*presenter.Presenter) error {
	return errors.New("Use -tags ci for --smoke; launch without --smoke for the native desktop")
}
