//go:build ci

package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	fyneTest "fyne.io/fyne/v2/test"
	"github.com/imneov/ImagePort/internal/domain"
)

// TestRowTapSelectsAndEntersFolder simulates real row taps, which previously only worked on row padding.
// Trees cannot host a Tappable child, so each row reports its own taps; this guards that contract.
func TestRowTapSelectsAndEntersFolder(test *testing.T) {
	view := commanderFixture(test)
	view.targets = append(view.targets, domain.Target{ID: "fs", Name: "Home", Kind: "filesystem", Address: "/home/kasm-user"})
	pane := view.panes[1]
	pane.chooseTarget("fs", "")

	archive := domain.LocationEntry{ID: "/home/kasm-user/a.tar", Path: "/home/kasm-user/a.tar", Name: "a.tar", Kind: "archive"}
	pane.register("", []domain.LocationEntry{archive})
	row := pane.newTappableRow()
	pane.updateRow(archive.ID, row)
	fyne.NewContainer(row).Resize(fyne.NewSize(300, 30))
	fyneTest.Tap(row)
	test.Logf("after archive tap: selected=%q", pane.selected)
	if pane.selected != archive.ID {
		test.Fatalf("tapping a row did not select it, selected=%q", pane.selected)
	}

	folder := domain.LocationEntry{ID: "/home/kasm-user/docs", Path: "/home/kasm-user/docs", Name: "docs", Kind: "folder"}
	pane.register("", []domain.LocationEntry{folder})
	row2 := pane.newTappableRow()
	pane.updateRow(folder.ID, row2)
	fyneTest.Tap(row2)
	test.Logf("after folder tap: rootPath=%q", pane.rootPath)
	if pane.rootPath != "/home/kasm-user/docs" {
		test.Fatalf("tapping a folder did not enter it, rootPath=%q", pane.rootPath)
	}
}
