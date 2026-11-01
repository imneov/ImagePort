package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// TestPaneUsesOnlyTree verifies browsing has no competing list or empty metadata columns.
func TestPaneUsesOnlyTree(test *testing.T) {
	pane := commanderFixture(test).panes[1]
	if visit(pane.content, func(object fyne.CanvasObject) bool {
		_, valid := object.(*widget.List)
		return valid
	}) != nil {
		test.Fatal("pane still exposes a list browser")
	}
	entry := domain.LocationEntry{ID: "image:team/service", Name: "team/service-with-a-long-name", Kind: "image"}
	pane.register("", []domain.LocationEntry{entry})
	row := pane.createRow()
	pane.updateRow(entry.ID, row)
	label, valid := row.(*widget.Label)
	if !valid || label.Text != entry.Name || label.Truncation != fyne.TextTruncateEllipsis {
		test.Fatal("tree row does not reserve its full width for the received name")
	}
	defaultHeight := fyne.MeasureText("M", Theme().Size(theme.SizeNameText), fyne.TextStyle{}).Height + 2*theme.DefaultTheme().Size(theme.SizeNameInnerPadding)
	if row.MinSize().Height >= defaultHeight {
		test.Fatalf("tree row is too tall: %v", row.MinSize())
	}
}

// TestTreeNestedPagination retains branch-specific cursors and totals through filtering and continuation.
func TestTreeNestedPagination(test *testing.T) {
	pane := commanderFixture(test).panes[1]
	parent := "image:team/service"
	pane.register("", []domain.LocationEntry{{ID: parent, Name: "service", Kind: "image", Children: []domain.LocationEntry{}}})
	pane.applyPage(parent, "", locationResponse{page: domain.LocationPage{
		Entries: []domain.LocationEntry{{ID: "version:one", Name: "one", Kind: "version"}}, Next: "opaque-next", Total: 3, HasTotal: true,
	}})
	pane.tree.OpenBranch(parent)
	pane.tree.Select("version:one")
	pane.filter.SetText("one")
	if len(pane.visibleChildren(parent)) != 2 || pane.pages[parent].Next != "opaque-next" || !strings.Contains(pane.summary.Text, "3 total") {
		test.Fatal("filter lost the branch continuation or server total")
	}
	pane.applyPage(parent, "opaque-next", locationResponse{page: domain.LocationPage{
		Entries: []domain.LocationEntry{{ID: "version:two", Name: "two", Kind: "version"}}, Total: 3, HasTotal: true,
	}})
	pane.filter.SetText("")
	if !pane.tree.IsBranchOpen(parent) || pane.selected != "version:one" || len(pane.children[parent]) != 2 || pane.pages[parent].Next != "" {
		test.Fatal("continuation lost tree context, replaced earlier entries or retained load-more")
	}
}

// TestCompactThemePreservesReadableText ensures density comes from spacing rather than smaller fonts.
func TestCompactThemePreservesReadableText(test *testing.T) {
	current := Theme()
	if current.Size(theme.SizeNameText) != 14 {
		test.Fatal("compact theme reduced the readable body font")
	}
	for _, name := range []fyne.ThemeSizeName{theme.SizeNameInnerPadding, theme.SizeNamePadding} {
		if current.Size(name) >= theme.DefaultTheme().Size(name) {
			test.Fatalf("theme did not reduce %s", name)
		}
	}
}
