package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/registry"
)

// registryView retains browsing state, server cursors and the identity of in-flight reads.
type registryView struct {
	desktop                                           *Desktop
	repository, filter                                *widget.Entry
	listing                                           *widget.List
	current                                           domain.Page
	rows                                              []string
	mode, listedMode, listedRepository, listedProfile string
	project, listedProject                            string
	generation                                        uint64
	cursors                                           []string
	page                                              int
	previous, nextPage                                *widget.Button
	projectBack                                       *widget.Button
	summary, notice                                   *widget.Label
	favorites                                         bool
	readError                                         string
}

// registryPage presents the approved images workspace using live registry responses.
func (desktop *Desktop) remoteImagesPage() fyne.CanvasObject {
	view := &registryView{desktop: desktop, repository: widget.NewEntry(), filter: widget.NewEntry(), mode: "repos"}
	desktop.browser = view
	desktop.placeholder(view.repository, "library/alpine or a full image reference")
	desktop.placeholder(view.filter, "Filter this page only")
	view.filter.OnChanged = view.render
	view.repository.OnChanged = func(string) { view.generation++ }
	view.repository.OnSubmitted = func(string) { view.resolve() }
	view.summary, view.notice = widget.NewLabel(""), widget.NewLabel("")
	view.notice.Wrapping = fyne.TextWrapWord
	view.listing = widget.NewList(func() int { return len(view.rows) }, view.createRow, view.updateRow)
	view.listing.OnSelected = func(index widget.ListItemID) { view.open(view.rows[index]) }
	desktop.bindText(func() { view.render(view.filter.Text) })
	desktop.recentTasks = container.NewVBox()
	top := container.NewVBox(view.controls(), view.notice)
	headers := columns([]float32{4, 2, 2, 2}, desktop.label("Project / repository / tag"), desktop.label("Type"), desktop.label("Metadata"), desktop.label("Actions"))
	view.previous = desktop.button("Previous", view.back)
	view.nextPage = desktop.button("Next page", view.next)
	view.previous.Disable()
	view.nextPage.Disable()
	footer := container.NewVBox(container.NewHBox(view.summary, layout.NewSpacer(), view.previous, view.nextPage), widget.NewSeparator(), desktop.label("Recent transfers"), desktop.recentTasks)
	return container.NewBorder(top, footer, nil, nil, container.NewBorder(headers, nil, nil, nil, view.listing))
}

// controls retains direct references when a registry disallows catalog enumeration.
func (view *registryView) controls() fyne.CanvasObject {
	desktop := view.desktop
	repos := desktop.button("Browse registry", view.browse)
	back := desktop.button("Project repositories", func() { view.mode = "repos"; view.load("") })
	view.projectBack = back
	back.Disable()
	favorites := desktop.button("Favorites", func() { view.favorites = !view.favorites; view.render(view.filter.Text) })
	tags := desktop.button("List tags", func() { view.mode = "tags"; view.load("") })
	resolve := desktop.button("Select image", view.resolve)
	resolve.Importance = widget.HighImportance
	return container.NewVBox(container.NewHBox(repos, back, favorites), view.repository, container.NewHBox(tags, resolve), view.filter)
}

// browse starts with authorized Harbor projects and falls back only for other products.
func (view *registryView) browse() {
	view.reset()
	view.project, view.mode, view.favorites = "", "projects", false
	view.load("")
}

// readPage selects the correct product API while retaining generic OCI compatibility.
func (view *registryView) readPage(ctx context.Context, profile domain.Profile, mode, project, input, cursor string) (domain.Page, string, error) {
	port := view.desktop.presenter.Registry
	product, supported := port.(registry.ProjectBrowser)
	if mode == "projects" && supported {
		page, err := product.Projects(ctx, profile, cursor, 50)
		if !registry.IsCode(err, "UNSUPPORTED") {
			return page, mode, err
		}
	}
	if mode == "projects" {
		mode = "repos"
	}
	if mode == "repos" {
		if project != "" && supported {
			page, err := product.ProjectRepositories(ctx, profile, project, cursor, 50)
			return page, mode, err
		}
		page, err := port.Repositories(ctx, profile, cursor, 50)
		return page, mode, err
	}
	page, err := port.Tags(ctx, profile, input, cursor, 50)
	return page, mode, err
}

// reset invalidates results when the active connection changes.
func (view *registryView) reset() {
	view.generation++
	view.current = domain.Page{}
	view.readError = ""
	view.listedProfile, view.listedRepository, view.listedMode = "", "", ""
	view.cursors, view.page = nil, 0
	view.render(view.filter.Text)
}

// load starts a first or next server page; the UI never repaginates response items.
func (view *registryView) load(cursor string) {
	page := 0
	if cursor != "" {
		page = view.page + 1
	}
	view.loadPage(cursor, page)
}

// loadPage captures request identity, keeping stale data and errors out of the current view.
func (view *registryView) loadPage(cursor string, pageNumber int) {
	profile, valid := view.desktop.connection()
	if !valid {
		return
	}
	input, mode, project := strings.TrimSpace(view.repository.Text), view.mode, view.project
	if profile.ID != view.listedProfile || project != view.listedProject || mode != view.listedMode || (mode == "tags" && input != view.listedRepository) {
		view.reset()
	}
	view.readError = ""
	view.generation++
	generation := view.generation
	type response struct {
		page domain.Page
		err  error
		mode string
	}
	view.notice.SetText(view.desktop.tr("Reading registry page..."))
	view.desktop.run("Reading registry page...", func(ctx context.Context) (any, error) {
		page, actualMode, err := view.readPage(ctx, profile, mode, project, input, cursor)
		return response{page, err, actualMode}, nil
	}, func(result any) {
		active := view.desktop.sourceProfileIndex()
		if generation != view.generation || active < 0 || active >= len(view.desktop.profiles) || view.desktop.profiles[active].ID != profile.ID {
			return
		}
		value := result.(response)
		if value.err != nil {
			view.readError = view.desktop.tr("Repository listing is unavailable or not permitted. Enter a known repository path to list tags, or resolve a full image reference.") + "\n" + view.desktop.translateDetails(value.err.Error())
			view.render(view.filter.Text)
			return
		}
		view.current, view.listedProfile, view.listedRepository, view.listedMode = value.page, profile.ID, input, mode
		view.mode, view.listedMode, view.listedProject = value.mode, value.mode, project
		if pageNumber < len(view.cursors) {
			view.cursors = view.cursors[:pageNumber]
		}
		view.cursors = append(view.cursors, cursor)
		view.page = pageNumber
		view.render(view.filter.Text)
	})
}

// render filters only the received page; counts always describe the page, never a server total.
func (view *registryView) render(query string) {
	view.rows = nil
	for _, value := range view.current.Items {
		if !strings.Contains(strings.ToLower(value), strings.ToLower(query)) {
			continue
		}
		if view.favorites && !view.isFavorite(value) {
			continue
		}
		view.rows = append(view.rows, value)
	}
	if view.listing == nil {
		return
	}
	view.listing.UnselectAll()
	view.listing.Refresh()
	view.summary.SetText(view.desktop.tr("Page %d · %d visible / %d received", view.page+1, len(view.rows), len(view.current.Items)))
	if view.projectBack != nil {
		view.projectBack.Disable()
		if view.project != "" {
			view.projectBack.Enable()
		}
	}
	if view.current.HasTotal {
		view.summary.SetText(view.desktop.tr("Page %d · %d visible · %d total", view.page+1, len(view.rows), view.current.Total))
	}
	if view.readError != "" {
		view.notice.SetText(view.readError)
	} else if view.listedProfile == "" {
		view.notice.SetText(view.desktop.tr("Select a registry, then browse repositories or enter a known image reference."))
	} else if len(view.current.Items) == 0 {
		view.notice.SetText(view.desktop.tr("No results on this server page. Try a known repository or image reference."))
	} else {
		view.notice.SetText(view.desktop.tr("Open a project, repository or tag to browse its images."))
		if view.listedProject != "" {
			view.notice.SetText(view.desktop.tr("Project: %s", view.listedProject))
		}
	}
	if view.previous != nil {
		view.previous.Disable()
		view.nextPage.Disable()
		if view.page > 0 {
			view.previous.Enable()
		}
		if view.current.Next != "" {
			view.nextPage.Enable()
		}
	}
}

// next follows the registry cursor only while the requested context still matches.
func (view *registryView) next() {
	if view.current.Next != "" && view.contextMatches() {
		view.load(view.current.Next)
	}
}

// back revisits an earlier server cursor without applying client-side slicing.
func (view *registryView) back() {
	if view.page > 0 && view.contextMatches() {
		view.loadPage(view.cursors[view.page-1], view.page-1)
	}
}

// contextMatches prevents using a cursor for a different connection or repository.
func (view *registryView) contextMatches() bool {
	profile, valid := view.desktop.connection()
	matches := valid && profile.ID == view.listedProfile && view.project == view.listedProject && view.mode == view.listedMode && (view.mode != "tags" || strings.TrimSpace(view.repository.Text) == view.listedRepository)
	if !matches {
		view.desktop.setStatus("The connection or path changed; reload the first page")
	}
	return matches
}

// open lists a repository's tags or inspects an explicitly clicked tag.
func (view *registryView) open(value string) {
	if view.listedMode == "projects" {
		view.project, view.mode = value, "repos"
		view.load("")
	} else if view.listedMode == "repos" {
		view.repository.SetText(value)
		view.mode = "tags"
		view.load("")
	} else {
		view.resolveReference(view.listedRepository + ":" + value)
	}
}

// resolve resolves the entered image reference without inferring a local CPU platform.
func (view *registryView) resolve() { view.resolveReference(view.repository.Text) }

// resolveReference captures the connection and ignores results after it changes.
func (view *registryView) resolveReference(reference string) {
	profile, valid := view.desktop.connection()
	if !valid {
		return
	}
	view.generation++
	generation := view.generation
	view.desktop.runCurrent("Resolving digests and available platforms...", func() bool { return generation == view.generation && view.desktop.currentPage == imagesPage }, func(ctx context.Context) (any, error) {
		return view.desktop.presenter.Registry.Resolve(ctx, profile, strings.TrimSpace(reference))
	}, func(result any) {
		if generation == view.generation {
			view.desktop.selectSource(profile, result.([]domain.Source))
		}
	})
}

// createRow creates reusable columns for a virtualized repository/tag row.
func (view *registryView) createRow() fyne.CanvasObject {
	name, kind, metadata := widget.NewLabel(""), widget.NewLabel(""), widget.NewLabel("")
	name.Truncation, metadata.Truncation = fyne.TextTruncateEllipsis, fyne.TextTruncateEllipsis
	name.TextStyle.Bold = true
	actions := container.NewHBox(widget.NewButton("☆", nil), widget.NewButtonWithIcon("", theme.NavigateNextIcon(), nil))
	return columns([]float32{4, 2, 2, 2}, name, kind, metadata, actions)
}

// updateRow binds actions to the current response item instead of a recycled row index.
func (view *registryView) updateRow(index widget.ListItemID, object fyne.CanvasObject) {
	value := view.rows[index]
	cells := object.(*fyne.Container).Objects
	cells[0].(*widget.Label).SetText(value)
	kind := "Repository"
	if view.listedMode == "projects" {
		kind = "Project"
	}
	if view.listedMode == "tags" {
		kind = "Image tag"
	}
	cells[1].(*widget.Label).SetText(view.desktop.tr(kind))
	cells[2].(*widget.Label).SetText(view.desktop.tr("Inspect to load"))
	actions := cells[3].(*fyne.Container).Objects
	favorite := actions[0].(*widget.Button)
	favorite.OnTapped = func() { view.toggleFavorite(value) }
	favorite.SetText("☆")
	favorite.Importance = widget.LowImportance
	if view.isFavorite(value) {
		favorite.SetText("★")
		favorite.Importance = widget.HighImportance
	}
	favorite.Refresh()
	actions[1].(*widget.Button).OnTapped = func() { view.open(value) }
}

// favoriteKey namespaces a favorite by saved connection, listing mode and repository.
func (view *registryView) favoriteKey(value string) string {
	repository := ""
	if view.listedMode == "tags" {
		repository = view.listedRepository
	}
	return fmt.Sprintf("%s/%s/%s/%s", view.listedProfile, view.listedMode, repository, value)
}

// isFavorite returns whether this exact connection/repository item is bookmarked.
func (view *registryView) isFavorite(value string) bool {
	return view.desktop.app.Preferences().Bool("favorite:" + view.favoriteKey(value))
}

// toggleFavorite persists a bookmark without storing credentials or changing remote state.
func (view *registryView) toggleFavorite(value string) {
	key := "favorite:" + view.favoriteKey(value)
	view.desktop.app.Preferences().SetBool(key, !view.isFavorite(value))
	view.render(view.filter.Text)
}
