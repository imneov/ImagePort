package ui

import (
	"context"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/distribution/reference"
	"github.com/imneov/ImagePort/internal/domain"
)

// workspaceSource is the captured, inspected source used throughout a copy review.
type workspaceSource struct {
	target  domain.Target
	profile domain.Profile
	entry   domain.LocationEntry
	sources []domain.Source
	archive domain.Archive
	local   domain.LocalImage
}

// copyReview owns captured source/destination intent independently of later pane navigation.
type copyReview struct {
	workspace                    *commanderView
	source                       workspaceSource
	target                       domain.Target
	engine                       domain.Engine
	accountID                    string
	project, name, tag, filename *widget.Entry
	platform, entry, account     *widget.Select
	accountIDs                   []string
	projectChoice                *widget.SelectEntry
	projectNames                 []string
	preview, feedback            *widget.Label
	replace                      *widget.Check
	submit                       *widget.Button
	popup                        *dialog.CustomDialog
	previous, observedReference  string
	outputDirectory              string
	checking                     bool
	closed                       bool
	revision                     uint64
}

// copySelected resolves only the selected image, keeping all blocking work off the UI goroutine.
func (view *commanderView) copySelected() {
	if view.copy.Disabled() {
		return
	}
	pane, destination := view.panes[view.active], view.panes[1-view.active]
	sourceTarget, _ := view.target(pane.targetID)
	target, _ := view.target(destination.targetID)
	entry, account, generation := pane.nodes[pane.selected], pane.accountID, pane.generation
	destinationGeneration := destination.generation
	engine, targetAccount := destination.engine, destination.accountID
	project := destination.container("project").Path
	if project == "" && target.Kind == "registry" {
		if prefix, _, found := strings.Cut(destination.container("image").Path, "/"); found {
			project = prefix
		}
	}
	folder := destination.container("folder").Path
	if folder == "" {
		folder = destination.rootPath
	}
	view.desktop.runCurrent("Inspecting selected image...", func() bool { return pane.generation == generation && destination.generation == destinationGeneration }, func(ctx context.Context) (any, error) {
		return view.readSource(ctx, sourceTarget, account, entry)
	}, func(value any) {
		review := &copyReview{workspace: view, source: value.(workspaceSource), target: target, engine: engine, accountID: targetAccount}
		for _, entry := range destination.nodes {
			if entry.Kind == "project" {
				review.projectNames = append(review.projectNames, entry.Path)
			}
		}
		review.show(project, folder)
	})
}

// readSource pins remote manifests, daemon IDs or validated archive identity before confirmation.
func (view *commanderView) readSource(ctx context.Context, target domain.Target, account string, entry domain.LocationEntry) (workspaceSource, error) {
	result := workspaceSource{target: target, entry: entry}
	service := view.desktop.presenter
	var err error
	switch target.Kind {
	case "registry":
		result.profile, err = service.TaskProfile(target, account)
		if err == nil {
			result.sources, err = service.Registry.Resolve(ctx, result.profile, entry.Path)
		}
	case "engine":
		result.local, err = service.Local.Inspect(ctx, entry.Local.Engine, entry.Local.ImageID)
	case "filesystem":
		result.archive, err = service.InspectArchive(ctx, entry.Path)
	}
	return result, err
}

// show displays source, exact destination naming and supported content choices in one review.
func (review *copyReview) show(project, folder string) {
	desktop := review.workspace.desktop
	review.projectChoice = widget.NewSelectEntry(review.projectNames)
	review.project = &review.projectChoice.Entry
	review.name, review.tag, review.filename = widget.NewEntry(), widget.NewEntry(), widget.NewEntry()
	review.platform, review.entry = widget.NewSelect(nil, nil), widget.NewSelect(nil, nil)
	review.preview, review.feedback = widget.NewLabel(""), widget.NewLabel("")
	review.preview.Wrapping, review.feedback.Wrapping = fyne.TextWrapBreak, fyne.TextWrapWord
	review.feedback.Hide()
	review.replace = widget.NewCheck(desktop.tr("Replace this version at the destination"), nil)
	review.replace.Hide()
	review.project.SetText(project)
	review.defaultNames()
	if folder == "" {
		folder = review.target.Address
	}
	review.outputDirectory = folder
	review.contentChoices()
	review.account = widget.NewSelect(nil, nil)
	review.reloadAccounts()
	review.submit = widget.NewButton(desktop.tr("Start copy"), review.checkAndSubmit)
	review.submit.Importance = widget.HighImportance
	cancel := widget.NewButton(desktop.tr("Cancel"), func() { review.popup.Hide() })
	summary := widget.NewLabel(review.source.target.Name + " / " + review.source.entry.Path + "  →  " + review.target.Name)
	summary.Wrapping = fyne.TextWrapBreak
	content := container.NewVBox(summary, review.fields(), review.preview, review.feedback, review.replace)
	review.popup = dialog.NewCustomWithoutButtons(desktop.tr("Copy image"), container.NewBorder(nil, container.NewHBox(widget.NewLabel(desktop.tr("The original stays where it is.")), cancel, review.submit), nil, nil, container.NewVScroll(content)), desktop.window)
	review.bindChanges()
	review.refreshPreview()
	review.popup.Resize(fyne.NewSize(720, 620))
	review.popup.SetOnClosed(func() { review.closed = true })
	review.popup.Show()
}

// defaultNames derives names from source content; an unrelated target selection cannot rename it.
func (review *copyReview) defaultNames() {
	name, tag := imageNames(review.source.entry.Path)
	if review.source.target.Kind == "filesystem" {
		name, tag = "", ""
	}
	review.name.SetText(name)
	review.tag.SetText(tag)
	filename := strings.ReplaceAll(name, "/", "-") + "-" + tag + ".tar"
	if name == "" {
		filename = "image.tar"
	}
	review.filename.SetText(filename)
}

// imageNames preserves the image path below the source namespace and its explicit tag.
func imageNames(input string) (string, string) {
	parsed, err := reference.ParseNormalizedNamed(input)
	if err != nil {
		return "", ""
	}
	name := reference.Path(parsed)
	if _, relative, namespaced := strings.Cut(name, "/"); namespaced {
		name = relative
	}
	tag := ""
	if tagged, valid := parsed.(reference.Tagged); valid {
		tag = tagged.Tag()
	}
	return name, tag
}
