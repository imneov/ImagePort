package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// commanderFixture configures independent locations without contacting a registry or daemon.
func commanderFixture(test *testing.T) *commanderView {
	test.Helper()
	view := testDesktop(test).commander
	view.targets = []domain.Target{{ID: "local", Name: "Local", Kind: "engine"}, {ID: "remote", Name: "Registry", Kind: "registry", Address: "https://registry.example.com"}}
	for index, pane := range view.panes {
		pane.targetID, pane.targetAddress = view.targets[index].ID, view.targets[index].Address
		pane.reloadOptions()
	}
	return view
}

// TestCommanderRendersReceivedRows checks API-derived items actually reach visible Fyne row renderers.
func TestCommanderRendersReceivedRows(test *testing.T) {
	view := commanderFixture(test)
	pane := view.panes[1]
	pane.applyPage("", "", locationResponse{page: domain.LocationPage{Entries: []domain.LocationEntry{{ID: "project:team", Name: "team", Kind: "project", Path: "team"}}, Next: "opaque-next", Total: 83, HasTotal: true}})
	found := visit(pane.content, func(object fyne.CanvasObject) bool {
		label, ok := object.(*widget.Label)
		return ok && label.Text == "team"
	})
	if found == nil {
		test.Fatal("received project count exists but its row is invisible")
	}
	pane.filter.SetText("team")
	if pane.pages[""].Next != "opaque-next" || !strings.Contains(pane.summary.Text, "83 total") {
		test.Fatal("filter discarded server pagination")
	}
	pane.applyPage("", "opaque-next", locationResponse{page: domain.LocationPage{Entries: []domain.LocationEntry{{ID: "project:other", Name: "other", Kind: "project", Path: "other"}}, Total: 83, HasTotal: true}})
	pane.filter.SetText("")
	if len(pane.children[""]) != 2 || pane.pages[""].Next != "" {
		test.Fatal("continuation replaced rows or retained a stale load-more control")
	}
}

// TestCommanderActivePaneAndLanguage retain independent IDs, selection, paths and command direction.
func TestCommanderActivePaneAndLanguage(test *testing.T) {
	view := commanderFixture(test)
	left, right := view.panes[0], view.panes[1]
	local := domain.LocationEntry{ID: "built:v3", Name: "v3", Path: "built:v3", Kind: "version", Repository: "built", Local: domain.LocalSource{ImageID: "sha256:pinned"}}
	left.register("", []domain.LocationEntry{local})
	right.register("", []domain.LocationEntry{{ID: "other", Kind: "version", Path: "team/unrelated:old", Repository: "team/unrelated"}})
	right.selectNode("other")
	left.selectNode(local.ID)
	left.path.SetText("built")
	left.filter.SetText("built")
	if view.copy.Disabled() || !strings.Contains(view.copy.Text, "Registry") {
		test.Fatal("active left selection did not copy to right location")
	}
	view.desktop.changeLanguage("zh-Hans")
	if left.selected != local.ID || right.selected != "other" || left.path.Text != "built" || left.filter.Text != "built" || left.targetID != "local" {
		test.Fatal("translation changed user intent")
	}
	view.activate(1)
	if view.copy.Disabled() || !strings.Contains(view.copy.Text, "Local") {
		test.Fatal("active right selection did not reverse copy direction")
	}
	view.targets = view.targets[:1]
	right.reloadOptions()
	if right.selected == "other" {
		test.Fatal("removed target retained actionable old content")
	}
}

// TestCopyReviewPreservesSourceAndRevokesConsent checks naming, explicit projects and conflict edits.
func TestCopyReviewPreservesSourceAndRevokesConsent(test *testing.T) {
	view := commanderFixture(test)
	review := &copyReview{workspace: view, target: view.targets[1], source: workspaceSource{
		target: view.targets[0], entry: domain.LocationEntry{Path: "built:v3", Local: domain.LocalSource{ImageID: "sha256:pinned"}}, local: domain.LocalImage{Platform: "linux/arm64"}}}
	review.show("team", "")
	defer review.popup.Hide()
	spec, err := review.specification()
	if err != nil || spec.Target != "team/built:v3" || spec.Operation != "push" || spec.Local.ImageID != "sha256:pinned" {
		test.Fatalf("copy intent: %+v %v", spec, err)
	}
	spec.PreviousDigest = "sha256:old"
	review.confirmObserved(spec, review.preview.Text)
	if !review.replace.Visible() || review.replace.Checked {
		test.Fatal("replacement was implicitly authorized")
	}
	review.replace.SetChecked(true)
	review.tag.SetText("v4")
	if review.replace.Checked || review.replace.Visible() || review.observedReference != "" {
		test.Fatal("edited destination retained overwrite consent")
	}
	review.project.SetText("")
	if _, err := review.specification(); err == nil {
		test.Fatal("missing project was accepted")
	}
}

// TestArchiveAndPlatformChoicesAreExplicit prevents silently choosing one image/platform from several.
func TestArchiveAndPlatformChoicesAreExplicit(test *testing.T) {
	view := commanderFixture(test)
	review := &copyReview{workspace: view, target: view.targets[0], source: workspaceSource{
		target: domain.Target{Kind: "filesystem"}, archive: domain.Archive{Path: "/images/export.tar", Entries: []domain.ArchiveEntry{{Name: "one:v1", Platforms: []string{"linux/amd64", "linux/arm64"}}, {Name: "two:v1"}}}}}
	review.show("", "")
	defer review.popup.Hide()
	if _, err := review.specification(); err == nil || review.entry.SelectedIndex() != -1 {
		test.Fatal("multi-image archive was implicitly selected")
	}
	review.entry.SetSelectedIndex(0)
	if review.platform.SelectedIndex() != -1 {
		test.Fatal("multi-platform engine load was implicitly selected")
	}
	review.platform.SetSelected("linux/arm64")
	spec, err := review.specification()
	if err != nil || spec.Entry.Name != "one:v1" || spec.Source.Platform != "linux/arm64" || spec.Operation != "load" {
		test.Fatalf("archive selection: %+v %v", spec, err)
	}
}

// TestSwapPreservesBrowsingContext prevents a convenience action from silently returning both panes to roots.
func TestSwapPreservesBrowsingContext(test *testing.T) {
	view := commanderFixture(test)
	left, right := view.panes[0], view.panes[1]
	left.register("", []domain.LocationEntry{{ID: "built:v3", Name: "v3", Kind: "version", Path: "built:v3", Repository: "built"}})
	right.register("", []domain.LocationEntry{{ID: "remote:v2", Name: "v2", Kind: "version", Path: "team/remote:v2", Repository: "team/remote"}})
	left.loaded[""], right.loaded[""] = true, true
	left.selected, right.selected = "built:v3", "remote:v2"
	left.path.SetText("built")
	left.filter.SetText("built")
	right.rootKind, right.rootPath = "project", "team"
	generation := left.generation
	view.activate(0)
	view.swap()
	if left.targetID != "remote" || right.targetID != "local" || right.selected != "built:v3" || right.filter.Text != "built" || right.path.Text != "built" {
		test.Fatal("swap lost location, selection or filter")
	}
	if left.rootKind != "project" || left.rootPath != "team" || left.selected != "remote:v2" || left.generation <= generation || view.active != 0 {
		test.Fatal("swap changed context or retained an old read generation")
	}
}

// TestRefreshReselectsSameNamedVersion clears widget selection as well as the backing source identity.
func TestRefreshReselectsSameNamedVersion(test *testing.T) {
	view := commanderFixture(test)
	pane := view.panes[0]
	entry := domain.LocationEntry{ID: "built:v1", Name: "v1", Kind: "version", Path: "built:v1", Repository: "built"}
	pane.register("", []domain.LocationEntry{entry})
	pane.tree.Select(entry.ID)
	if pane.selected != entry.ID {
		test.Fatal("initial selection missing")
	}
	pane.resetData()
	pane.register("", []domain.LocationEntry{entry})
	pane.render()
	pane.tree.Select(entry.ID)
	if pane.selected != entry.ID {
		test.Fatal("same-named version could not be selected after refresh")
	}
	view.activate(0)
	view.desktop.changeLanguage("zh-Hans")
	if view.active != 0 || pane.selected != entry.ID {
		test.Fatal("translation changed the active pane or tree selection")
	}
}

// TestCopyNamesPreserveNestedImagePaths prevents destination project selection from flattening source image names.
func TestCopyNamesPreserveNestedImagePaths(test *testing.T) {
	for source, expected := range map[string]string{"built:v3": "built", "registry.example.com/team/nested/api:v3": "nested/api"} {
		name, tag := imageNames(source)
		if name != expected || tag != "v3" {
			test.Fatalf("source %s became %s:%s", source, name, tag)
		}
	}
}
