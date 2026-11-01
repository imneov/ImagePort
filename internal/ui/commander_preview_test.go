package ui

import (
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/imneov/ImagePort/internal/domain"
)

// TestCommanderPreview renders review artifacts with deterministic sample content, never private registry data.
// IMAGEPORT_UI_SCREENSHOT_DIR enables artifacts; regular tests verify row and action behavior elsewhere.
func TestCommanderPreview(test *testing.T) {
	directory := os.Getenv("IMAGEPORT_UI_SCREENSHOT_DIR")
	if directory == "" {
		test.Skip("set IMAGEPORT_UI_SCREENSHOT_DIR to capture native-widget previews")
	}
	view := commanderFixture(test)
	image := domain.LocationEntry{ID: "image:built", Kind: "image", Name: "built", Path: "built", Children: []domain.LocationEntry{{ID: "built:v3", Kind: "version", Name: "v3", Path: "built:v3", Repository: "built", Size: 12 << 20}}}
	view.panes[0].applyPage("", "", locationResponse{page: domain.LocationPage{Entries: []domain.LocationEntry{image}}})
	remote := domain.LocationEntry{ID: "project:team", Kind: "project", Name: "team", Path: "team", Children: []domain.LocationEntry{{ID: "image:team/built", Kind: "image", Name: "built", Path: "team/built", Children: []domain.LocationEntry{{ID: "version:team/built:v2", Kind: "version", Name: "v2", Path: "team/built:v2", Repository: "team/built"}}}}}
	view.panes[1].applyPage("", "", locationResponse{page: domain.LocationPage{Entries: []domain.LocationEntry{remote}}})
	view.panes[0].tree.OpenBranch("image:built")
	view.panes[1].tree.OpenBranch("project:team")
	view.panes[1].tree.OpenBranch("image:team/built")
	view.panes[0].tree.Select("built:v3")
	captureCommander(test, view, directory, "native-workspace.png")
	view.accounts = []domain.Profile{{ID: "reader", TargetID: "remote", Name: "Read access", Username: "reader"}, {ID: "publisher", TargetID: "remote", Name: "Publish access", Username: "publisher"}}
	view.openManager("remote")
	captureCommander(test, view, directory, "native-targets.png")
	view.manager.popup.Hide()
	review := &copyReview{workspace: view, target: view.targets[1], projectNames: []string{"team"}, source: workspaceSource{target: view.targets[0], entry: domain.LocationEntry{Path: "built:v3"}, local: domain.LocalImage{Platform: "linux/arm64"}}}
	review.show("team", "")
	captureCommander(test, view, directory, "native-copy.png")
	review.popup.Hide()
	view.desktop.changeLanguage("zh-Hans")
	captureCommander(test, view, directory, "native-workspace-zh.png")
	view.desktop.window.Resize(fyne.NewSize(1024, 680))
	captureCommander(test, view, directory, "native-workspace-compact-zh.png")
	view.desktop.toggleTheme()
	captureCommander(test, view, directory, "native-workspace-compact-dark-zh.png")
}

// captureCommander exports a Fyne software-rendered window and propagates file/encoding errors.
func captureCommander(test *testing.T, view *commanderView, directory, name string) {
	test.Helper()
	if err := os.MkdirAll(directory, 0755); err != nil {
		test.Fatal(err)
	}
	view.desktop.window.Content().Refresh()
	file, err := os.Create(filepath.Join(directory, name))
	if err != nil {
		test.Fatal(err)
	}
	if err := errors.Join(png.Encode(file, view.desktop.window.Canvas().Capture()), file.Close()); err != nil {
		test.Fatal(err)
	}
}
