//go:build ci

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/imneov/ImagePort/internal/presenter"
)

// newApplication provides Fyne's official software renderer for CI smoke checks.
func newApplication() fyne.App { return test.NewApp() }

// waitForSmoke lets the inline software dispatcher finish initial location reads before capture.
func waitForSmoke(service *presenter.Presenter) error { service.WaitReads(); return nil }
