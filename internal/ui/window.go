package ui

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/localization"
	"github.com/imneov/ImagePort/internal/presenter"
)

// Desktop owns view state only; SDK clients and transfer goroutines stay below it.
type Desktop struct {
	window               fyne.Window
	app                  fyne.App
	presenter            *presenter.Presenter
	profiles             []domain.Profile
	profile              *widget.Select
	status               *widget.Label
	body                 *fyne.Container
	pages                []fyne.CanvasObject
	tasks                []domain.Task
	taskList             *widget.List
	taskDetail           *widget.Label
	taskProgress         *widget.ProgressBarInfinite
	selectedTask         string
	closing              bool
	catalog              *localization.Catalog
	language             string
	translations         []func()
	statusMessage        string
	breadcrumb           *widget.Label
	activity             *widget.Button
	sidebar              fyne.CanvasObject
	workspace            *fyne.Container
	navigationButtons    []*widget.Button
	currentPage          int
	browser              *registryView
	localImages          *localImagesView
	copyView             *imageCopyView
	archives             *archivesView
	registryCards        *fyne.Container
	taskFilter           string
	syncingTaskSelection bool
	highlightedTask      string
	highlightedIndex     int
	taskRows             []domain.Task
	taskSummary          *widget.Label
	drawer               *fyne.Container
	drawerRefresh        func()
	drawerTitle          *widget.Label
	drawerTitleMessage   string
	recentTasks          *fyne.Container
	commander            *commanderView
}

// New creates the two-pane desktop, loads targets and subscribes to durable tasks.
func New(app fyne.App, window fyne.Window, service *presenter.Presenter, version string) (*Desktop, error) {
	if err := registerToolkitLocale(); err != nil {
		return nil, err
	}
	language := app.Preferences().StringWithFallback("language", "en")
	if language != "zh-Hans" {
		language = "en"
	}
	catalog, err := localization.New(language)
	if err != nil {
		return nil, err
	}
	desktop := &Desktop{app: app, window: window, presenter: service, catalog: catalog, language: language}
	desktop.restoreAppearance()
	desktop.status = widget.NewLabel("")
	desktop.setStatus("Ready · Image management")
	desktop.profile = widget.NewSelect(nil, nil)
	desktop.bindText(func() {
		desktop.profile.PlaceHolder = desktop.tr("Select a registry connection")
		desktop.profile.Refresh()
	})
	desktop.bindText(func() { window.SetTitle(desktop.tr("ImagePort · Image Management")) })
	desktop.pages = make([]fyne.CanvasObject, len(pageNames))
	desktop.pages[preferencesPage] = desktop.settingsPage(version)
	desktop.body = container.NewStack()
	window.SetContent(desktop.shell(version))
	window.Resize(fyne.NewSize(1360, 880))

	desktop.installShortcuts()
	desktop.commander.reloadTargets()
	service.Observe(func(items []domain.Task, err error) { fyne.Do(func() { desktop.refreshTasks(items, err) }) })
	window.SetCloseIntercept(desktop.requestClose)
	return desktop, nil
}

// run sends read/form work to the presenter and marshals every result through fyne.Do.
func (desktop *Desktop) run(label string, work func(context.Context) (any, error), done func(any)) {
	desktop.runCurrent(label, func() bool { return true }, work, done)
}

// runCurrent applies validity on the UI thread before presenting either a result or error.
func (desktop *Desktop) runCurrent(label string, valid func() bool, work func(context.Context) (any, error), done func(any)) {
	desktop.setStatus(label)
	desktop.presenter.Work(work, func(result any, err error) {
		fyne.Do(func() {
			if desktop.closing || !valid() {
				return
			}
			if err != nil {
				desktop.setStatus("Operation incomplete")
				desktop.showError(err)
				return
			}
			desktop.setStatus("Ready")
			if done != nil {
				done(result)
			}
		})
	})
}

// connection returns the selected profile or asks the user to add/select one.
func (desktop *Desktop) connection() (domain.Profile, bool) {
	index := desktop.sourceProfileIndex()
	if index < 0 || index >= len(desktop.profiles) {
		desktop.inform("Select a connection", "Select a registry on Images, or add a connection in Registries.")
		return domain.Profile{}, false
	}
	return desktop.profiles[index], true
}

// sourceProfileIndex treats the visible location as authoritative during remote browsing.
func (desktop *Desktop) sourceProfileIndex() int {
	if desktop.copyView != nil && desktop.copyView.sourceLocation.SelectedIndex() > 0 {
		return desktop.copyView.sourceLocation.SelectedIndex() - 1
	}
	return desktop.profile.SelectedIndex()
}

// reloadProfiles refreshes metadata asynchronously without reading secret values.
func (desktop *Desktop) reloadProfiles(selected string) {
	desktop.run("Loading registry profiles...", func(context.Context) (any, error) { return desktop.presenter.Store.Profiles() }, func(result any) {
		desktop.profiles = result.([]domain.Profile)
		options := profileOptions(desktop.profiles)
		desktop.profile.SetOptions(options)
		desktop.renderRegistries()
		desktop.copyView.reloadLocations()
		for index, profile := range desktop.profiles {
			if profile.ID == selected {
				desktop.profile.SetSelectedIndex(index)
				return
			}
		}
		if len(options) > 0 {
			desktop.profile.SetSelectedIndex(0)
		}
	})
}

// requestClose confirms active work, then lets the presenter cancel without blocking UI.
func (desktop *Desktop) requestClose() {
	items, err := desktop.presenter.Tasks.List()
	active := false
	for _, item := range items {
		active = active || item.State.Active() || item.State == domain.Queued
	}
	message := "Close ImagePort?"
	if err != nil {
		message += "\n" + err.Error()
	}
	if active {
		message = "Tasks are running or queued. Closing requests cancellation. Unknown results need verification after restart, and queued tasks wait for manual resumption."
	}
	desktop.confirm("Exit", message, func(confirmed bool) {
		if !confirmed {
			return
		}
		desktop.run("Requesting tasks to stop...", func(context.Context) (any, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			return nil, desktop.presenter.Tasks.Close(ctx)
		}, func(any) { desktop.closing = true; desktop.window.SetCloseIntercept(nil); desktop.window.Close() })
	})
}
