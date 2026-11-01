package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// TestRegistryPageKeepsServerRows verifies filtering and bookmarks preserve response/cursor data.
func TestRegistryPageKeepsServerRows(test *testing.T) {
	desktop := testDesktop(test)
	view := desktop.browser
	view.current = domain.Page{Items: []string{"team/api", "team/web", "other/cache"}, Next: "opaque-cursor"}
	view.listedProfile, view.listedMode = "first-registry", "repos"
	view.render("team/")
	if len(view.rows) != 2 || len(view.current.Items) != 3 || view.current.Next != "opaque-cursor" {
		test.Fatal("filter altered the server response")
	}
	view.toggleFavorite("team/api")
	view.favorites = true
	view.render("")
	if len(view.rows) != 1 || view.rows[0] != "team/api" {
		test.Fatal("favorites filter lost the selected repository")
	}
	view.listedProfile = "another-registry"
	view.render("")
	if len(view.rows) != 0 {
		test.Fatal("favorites leaked across connections")
	}
	generation := view.generation
	view.reset()
	if view.generation <= generation || len(view.current.Items) != 0 || view.current.Next != "" {
		test.Fatal("connection change retained stale browsing state")
	}
}

// TestPlatformSelectionIsExplicit verifies even a single platform requires a visible choice.
func TestPlatformSelectionIsExplicit(test *testing.T) {
	desktop := testDesktop(test)
	desktop.selectSource(domain.Profile{}, []domain.Source{{Reference: "registry.example.com/team/api:one", Digest: "sha256:abc", Platform: "linux/arm64"}})
	useImage := buttonByText(test, desktop.drawer, "Use selected image")
	radio := visit(desktop.drawer, func(object fyne.CanvasObject) bool { _, valid := object.(*widget.RadioGroup); return valid }).(*widget.RadioGroup)
	if !useImage.Disabled() || radio.Selected != "" {
		test.Fatal("a platform was implicitly chosen")
	}
	radio.SetSelected(radio.Options[0])
	if useImage.Disabled() {
		test.Fatal("explicit selection did not enable actions")
	}
	radio.SetSelected("")
	if radio.Selected == "" && (!useImage.Disabled()) {
		test.Fatal("deselection retained actionable consent")
	}
	desktop.closeDrawer()
	if desktop.drawer != nil || desktop.drawerRefresh != nil {
		test.Fatal("drawer state survived closing")
	}
}

// TestTransferFiltersAndArchivesUseVerifiedHistory prevents failed outputs becoming local results.
func TestTransferFiltersAndArchivesUseVerifiedHistory(test *testing.T) {
	desktop := testDesktop(test)
	desktop.tasks = []domain.Task{
		{ID: "good", State: domain.Succeeded, Spec: domain.Spec{Operation: "download", Output: "/downloads/good.tar"}},
		{ID: "bad", State: domain.Failed, Spec: domain.Spec{Operation: "download", Output: "/downloads/bad.tar"}},
		{ID: "active", State: domain.CancelRequested},
	}
	desktop.app.Preferences().SetStringList("archive-paths", []string{"/downloads/good.tar", "/imports/checked.tar"})
	desktop.selectedTask = "good"
	desktop.taskFilter = "Needs attention"
	desktop.filterTasks()
	if len(desktop.taskRows) != 1 || desktop.taskRows[0].ID != "bad" || desktop.selectedTask != "good" {
		test.Fatal("filter changed task identity or cancellation semantics")
	}
	desktop.taskFilter = "Active"
	desktop.filterTasks()
	if len(desktop.taskRows) != 1 || desktop.taskRows[0].ID != "active" {
		test.Fatal("cancellation request was treated as completed")
	}
	desktop.archives.refresh()
	if len(desktop.archives.paths) != 2 || strings.Contains(strings.Join(desktop.archives.paths, ","), "bad.tar") {
		test.Fatal("archive history contains failed or duplicate download results")
	}
}

// TestThemeAndNavigationPersistState verifies appearance and menu changes preserve image inputs.
func TestThemeAndNavigationPersistState(test *testing.T) {
	desktop := testDesktop(test)
	desktop.browser.repository.SetText("team/api:stable")
	before := desktop.app.Settings().Theme().Color(theme.ColorNameBackground, theme.VariantLight)
	desktop.toggleTheme()
	after := desktop.app.Settings().Theme().Color(theme.ColorNameBackground, theme.VariantLight)
	if before == after || !desktop.app.Preferences().Bool("dark-theme") {
		test.Fatal("theme variant was not applied and saved")
	}
	for index := range pageNames {
		desktop.show(index)
	}
	desktop.show(imagesPage)
	if desktop.browser.repository.Text != "team/api:stable" || desktop.breadcrumb.Text != "Images" {
		test.Fatal("navigation discarded image input or breadcrumb")
	}
}
