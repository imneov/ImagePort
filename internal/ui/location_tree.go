package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"image/color"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// createTree constructs a virtualized hierarchy whose branches load only when opened.
// Filesystem panes browse like a file manager: rows are flat and a click enters the folder.
func (pane *locationPane) createTree() {
	pane.tree = widget.NewTree(pane.visibleChildren, func(id string) bool {
		entry := pane.nodes[id]
		if id == "" {
			return true
		}
		if pane.isFileManager() {
			return false
		}
		return entry.Kind == "project" || entry.Kind == "image" || entry.Kind == "folder"
	}, func(bool) fyne.CanvasObject { return pane.newTappableRow() }, func(id string, _ bool, object fyne.CanvasObject) { pane.updateRow(id, object) })
	pane.tree.OnBranchOpened = func(id string) {
		pane.workspace.activate(pane.side)
		if !pane.loaded[id] && !pane.pending[id] {
			pane.load(id, "")
		}
	}
	pane.tree.OnSelected = pane.selectNode
	pane.tree.OnUnselected = func(id string) {
		pane.selected = ""
		if id != "" {
			pane.tree.RefreshItem(id)
		}
		pane.workspace.updateCommands()
	}
}

// tappableRow is a full-width row that paints an icon and a caption and reports its own taps.
// It replaces Tree selection for row activation: Tree rows cannot contain a widget that
// implements Tappable, because that widget swallows the tap before the tree sees it.
type tappableRow struct {
	widget.BaseWidget
	icon    *widget.Icon
	label   *widget.Label
	onTap   func()
	hovered bool
	// selected drives the selection highlight so the user can see what the
	// commands will act on, mirroring the tree's own selected-row colour.
	selected bool
}

// MouseIn and MouseOut give the row a hover cue, matching the tree's own behaviour.
func (row *tappableRow) MouseIn(*fyne.PointEvent) {
	row.hovered = true
	row.Refresh()
}

// MouseOut clears the hover cue when the pointer leaves the row.
func (row *tappableRow) MouseOut() {
	row.hovered = false
	row.Refresh()
}

// newTappableRow builds a recycled row with a leading type icon and an ellipsised caption.
func (pane *locationPane) newTappableRow() *tappableRow {
	row := &tappableRow{icon: widget.NewIcon(theme.FileIcon()), label: widget.NewLabel("")}
	row.label.Truncation = fyne.TextTruncateEllipsis
	row.ExtendBaseWidget(row)
	return row
}

// CreateRenderer lays out the icon and caption as one full-width clickable row.
func (row *tappableRow) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(color.Transparent)
	background.Hide()
	return &rowRenderer{row: row, background: background,
		objects: []fyne.CanvasObject{background, row.icon, row.label}}
}

// Tapped forwards a row click to the pane so selection and folder entry behave consistently.
func (row *tappableRow) Tapped(*fyne.PointEvent) {
	if row.onTap != nil {
		row.onTap()
	}
}

// TappedSecondary reports the same activation for a secondary click without opening menus.
func (row *tappableRow) TappedSecondary(*fyne.PointEvent) {}

// rowRenderer paints the row background and positions the icon and caption.
type rowRenderer struct {
	row        *tappableRow
	background *canvas.Rectangle
	objects    []fyne.CanvasObject
}

// Layout places the icon at the leading edge and stretches the caption across the row.
// Branch rows hide their icon so the tree's own chevron is the only arrow shown.
func (renderer *rowRenderer) Layout(size fyne.Size) {
	iconSize := theme.IconInlineSize()
	renderer.background.Resize(size)
	offset := float32(0)
	if !renderer.row.icon.Hidden {
		renderer.row.icon.Resize(fyne.NewSize(iconSize, iconSize))
		renderer.row.icon.Move(fyne.NewPos(0, (size.Height-iconSize)/2))
		offset = iconSize + theme.Padding()
	}
	renderer.row.label.Resize(fyne.NewSize(size.Width-offset-theme.Padding(), size.Height))
	renderer.row.label.Move(fyne.NewPos(offset, 0))
}

// MinSize reserves a single text line so rows keep the compact commander density.
func (renderer *rowRenderer) MinSize() fyne.Size {
	iconSize := theme.IconInlineSize()
	height := fyne.Max(renderer.row.label.MinSize().Height, iconSize)
	width := renderer.row.label.MinSize().Width + theme.Padding()
	if !renderer.row.icon.Hidden {
		width += iconSize + theme.Padding()
	}
	return fyne.NewSize(width, height)
}

// Refresh repaints the row background for the selection or hover state.
func (renderer *rowRenderer) Refresh() {
	th := renderer.row.Theme()
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	renderer.background.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	// The row owns its selection paint: the tree's own highlight sits underneath this
	// content, so drawing it here is what makes the selection actually visible.
	switch {
	case renderer.row.selected:
		renderer.background.FillColor = th.Color(theme.ColorNameSelection, variant)
		renderer.background.Show()
	case renderer.row.hovered:
		renderer.background.FillColor = th.Color(theme.ColorNameHover, variant)
		renderer.background.Show()
	default:
		renderer.background.Hide()
	}
	// Re-layout so the background matches the row after being shown from hidden,
	// where it may still carry a zero size and would paint nothing.
	renderer.Layout(renderer.row.Size())
	renderer.background.Refresh()
	renderer.row.icon.Refresh()
	renderer.row.label.Refresh()
}

// Objects returns the painted objects in draw order.
func (renderer *rowRenderer) Objects() []fyne.CanvasObject { return renderer.objects }

// Destroy releases no external resources.
func (renderer *rowRenderer) Destroy() {}

// updateRow binds a recycled row to its entry and forwards taps back to this pane.
func (pane *locationPane) updateRow(id string, object fyne.CanvasObject) {
	row, ok := object.(*tappableRow)
	if !ok {
		return
	}
	entry := pane.nodes[id]
	row.label.SetText(entry.Name)
	if icon := pane.rowIcon(entry); icon != nil {
		row.icon.SetResource(icon)
		row.icon.Show()
	} else {
		row.icon.Hide()
	}
	row.selected = pane.selected == id
	row.onTap = func() { pane.selectNode(id) }
	row.Refresh()
}

// rowIcon chooses a distinct glyph per entry kind without inventing remote metadata.
// Branch entries return nil: the tree already paints its own expand chevron, and a second
// glyph here would render two arrows side by side.
func (pane *locationPane) rowIcon(entry domain.LocationEntry) fyne.Resource {
	switch entry.Kind {
	case "parent":
		return theme.NavigateBackIcon()
	case "folder":
		return theme.FolderIcon()
	case "project", "image":
		return nil
	case "archive":
		return theme.DocumentIcon()
	case "more":
		return theme.MoreHorizontalIcon()
	default:
		return theme.FileIcon()
	}
}

// isFileManager reports whether this pane browses a folder as a flat, click-to-enter list.
func (pane *locationPane) isFileManager() bool {
	target, valid := pane.workspace.target(pane.targetID)
	return valid && target.Kind == "filesystem"
}

// enterFolder navigates into a directory the way a file manager does, staying under the root.
func (pane *locationPane) enterFolder(path string) {
	if _, valid := pane.workspace.target(pane.targetID); !valid {
		return
	}
	pane.rootKind, pane.rootPath = "folder", path
	pane.path.SetText(path)
	pane.filter.SetText("")
	pane.refresh()
}

// visibleChildren filters received nodes while retaining pagination controls and parent context.
func (pane *locationPane) visibleChildren(parent string) []string {
	var result []string
	query := strings.ToLower(pane.filter.Text)
	for _, id := range pane.children[parent] {
		if pane.matches(id, query) {
			result = append(result, id)
		}
	}
	return result
}

// selectNode keeps the selection local and updates shared commands without opening details.
func (pane *locationPane) selectNode(id string) {
	entry, valid := pane.nodes[id]
	if !valid {
		return
	}
	pane.workspace.activate(pane.side)
	if pane.isFileManager() && (entry.Kind == "folder" || entry.Kind == "parent") {
		pane.enterFolder(entry.Path)
		return
	}
	if entry.Kind == "more" {
		parent := pane.parents[id]
		pane.load(parent, pane.pages[parent].Next)
		return
	}
	previous := pane.selected
	pane.selected = id
	// Rebind the affected rows so the highlight moves with the selection, and keep the
	// tree's own selection in sync for keyboard navigation and accessibility.
	pane.tree.Select(id)
	if previous != "" && previous != id {
		pane.tree.RefreshItem(previous)
	}
	pane.tree.RefreshItem(id)
	pane.renderSummary()
	pane.workspace.updateCommands()
}

// render updates the current presentation without discarding selection or search text.
func (pane *locationPane) render() {
	if pane.tree == nil {
		return
	}
	pane.tree.Refresh()
	pane.renderSummary()
}

// renderSummary displays only actual selected content and received/server counts.
func (pane *locationPane) renderSummary() {
	if pane.summary == nil {
		return
	}
	parent := ""
	selected := pane.nodes[pane.selected]
	if pane.selected != "" {
		parent = pane.parents[pane.selected]
	}
	page := pane.pages[parent]
	count := 0
	for _, id := range pane.visibleChildren(parent) {
		if pane.nodes[id].Kind != "more" {
			count++
		}
	}
	text := pane.workspace.desktop.tr("%d items", count)
	if page.HasTotal {
		text = pane.workspace.desktop.tr("%d visible · %d total", count, page.Total)
	}
	if selected.ID != "" {
		text = selected.Path + " · " + text
	}
	pane.summary.SetText(text)
}

// humanSize formats byte counts for rows without claiming remote sizes before inspection.
func humanSize(size int64) string {
	if size < 1<<20 {
		return fmt.Sprintf("%.1f KB", float64(size)/(1<<10))
	}
	if size < 1<<30 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1<<20))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1<<30))
}

// register adds detached entries and preloaded local-image children under a stable parent.
func (pane *locationPane) register(parent string, entries []domain.LocationEntry) {
	for _, entry := range entries {
		if _, exists := pane.nodes[entry.ID]; !exists {
			pane.children[parent] = append(pane.children[parent], entry.ID)
		}
		pane.nodes[entry.ID], pane.parents[entry.ID] = entry, parent
		if entry.Children != nil {
			pane.loaded[entry.ID] = true
			pane.register(entry.ID, entry.Children)
		}
	}
}
