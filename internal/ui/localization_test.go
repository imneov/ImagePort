package ui

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/presenter"
	"github.com/imneov/ImagePort/internal/secret"
	"github.com/imneov/ImagePort/internal/store"
	"github.com/imneov/ImagePort/internal/task"
)

// testDesktop assembles real presentation/storage services under Fyne's test driver.
func testDesktop(test *testing.T) *Desktop {
	test.Helper()
	app := fyneTest.NewApp()
	app.Settings().SetTheme(Theme())
	database, err := store.Open(filepath.Join(test.TempDir(), "ui.db"))
	if err != nil {
		test.Fatal(err)
	}
	manager, err := task.New(database, nil, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		test.Fatal(err)
	}
	vault := secret.New()
	service := presenter.New(database, vault, adapter.New(vault), manager)
	// The headless driver dispatches inline; stop polling for deterministic UI-only checks.
	service.Close()
	window := app.NewWindow("ImagePort")
	desktop, err := New(app, window, service, "test")
	if err != nil {
		test.Fatal(err)
	}
	// Legacy component regressions instantiate their own views; the production window only builds Commander.
	desktop.pages = []fyne.CanvasObject{desktop.registryPage(), desktop.registriesPage(), desktop.archivesPage(), desktop.tasksPage(), desktop.pages[preferencesPage]}
	window.Show()
	test.Cleanup(func() {
		service.Close()
		if err := manager.Close(context.Background()); err != nil {
			test.Error(err)
		}
		if err := database.Close(); err != nil {
			test.Error(err)
		}
		window.SetCloseIntercept(nil)
		window.Close()
		app.Quit()
	})
	return desktop
}

// visit walks containers and public widget renderers to locate controls as users see them.
func visit(object fyne.CanvasObject, match func(fyne.CanvasObject) bool) fyne.CanvasObject {
	if match(object) {
		return object
	}
	var children []fyne.CanvasObject
	if container, ok := object.(*fyne.Container); ok {
		children = container.Objects
	}
	if widget, ok := object.(fyne.Widget); ok {
		children = fyneTest.WidgetRenderer(widget).Objects()
	}
	for _, child := range children {
		if child == object {
			continue
		}
		if found := visit(child, match); found != nil {
			return found
		}
	}
	return nil
}

// buttonByText finds a static screen action by its visible caption.
func buttonByText(test *testing.T, root fyne.CanvasObject, text string) *widget.Button {
	test.Helper()
	object := visit(root, func(object fyne.CanvasObject) bool {
		button, ok := object.(*widget.Button)
		return ok && button.Text == text
	})
	if object == nil {
		test.Fatalf("button not found: %s", text)
	}
	return object.(*widget.Button)
}

// TestLanguageSwitchRetainsState drives the selector and verifies all screen captions/inputs.
func TestLanguageSwitchRetainsState(test *testing.T) {
	desktop := testDesktop(test)
	input := desktop.browser.repository
	fyneTest.Type(input, "team/image:v1")
	desktop.tasks = []domain.Task{{ID: "existing", Spec: domain.Spec{Operation: "download", Source: domain.Source{Digest: "sha256:abc"}}, State: domain.Running, Attempts: []domain.Attempt{{Number: 1, State: domain.Running, Stage: "transfer"}}}}
	desktop.selectedTask = "existing"
	fyneTest.Tap(buttonByText(test, desktop.window.Content(), "Settings"))
	selector := visit(desktop.pages[preferencesPage], func(object fyne.CanvasObject) bool {
		selectBox, ok := object.(*widget.Select)
		return ok && len(selectBox.Options) == 2
	}).(*widget.Select)
	selector.SetSelected("简体中文")
	buttonByText(test, desktop.window.Content(), "管理目标")
	buttonByText(test, desktop.pages[registriesPage], "添加仓库")
	if input.Text != "team/image:v1" || desktop.selectedTask != "existing" {
		test.Fatal("language switch discarded input or selection")
	}
	if !strings.Contains(desktop.taskDetail.Text, "传输") || !strings.Contains(desktop.taskDetail.Text, "下载") {
		test.Fatal("task values were not translated")
	}
	if desktop.app.Preferences().String("language") != "zh-Hans" {
		test.Fatal("language not persisted")
	}
	if desktop.commander == nil || desktop.window.Canvas().Overlays().Top() == nil {
		test.Fatal("switch navigated away from settings")
	}
	selector.SetSelected("English")
	buttonByText(test, desktop.window.Content(), "Manage targets")
	if input.Text != "team/image:v1" || !strings.Contains(desktop.taskDetail.Text, "transfer") {
		test.Fatal("switch back lost state")
	}
}

// TestSavedLanguageAndDialogs verifies startup selection and translated validation feedback.
func TestSavedLanguageAndDialogs(test *testing.T) {
	desktop := testDesktop(test)
	desktop.changeLanguage("zh-Hans")
	window := desktop.app.NewWindow("restored")
	restored, err := New(desktop.app, window, desktop.presenter, "test")
	if err != nil {
		test.Fatal(err)
	}
	defer func() { window.SetCloseIntercept(nil); window.Close() }()
	if restored.language != "zh-Hans" {
		test.Fatal("saved language ignored")
	}
	buttonByText(test, restored.window.Content(), "设置")
	restored.connection()
	overlays := window.Canvas().Overlays().List()
	if len(overlays) == 0 {
		test.Fatal("missing connection guidance")
	}
	buttonByText(test, overlays[len(overlays)-1], "确定")
}

// TestCatalogFailureGuidance retains both the fallback path and actionable auth error.
func TestCatalogFailureGuidance(test *testing.T) {
	desktop := testDesktop(test)
	desktop.changeLanguage("zh-Hans")
	message := "Repository listing is unavailable or not permitted. Enter a known repository path to list tags, or resolve a full image reference.\n\nCredentials are missing or expired; enter the password again"
	translated := desktop.translateDetails(message)
	if strings.Contains(translated, "Repository listing") || strings.Contains(translated, "Credentials are") {
		test.Fatal("catalog fallback or credential guidance was lost")
	}
}
