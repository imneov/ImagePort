package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// createTree constructs a virtualized hierarchy whose branches load only when opened.
func (pane *locationPane) createTree() {
	pane.tree = widget.NewTree(pane.visibleChildren, func(id string) bool {
		entry := pane.nodes[id]
		return id == "" || entry.Kind == "project" || entry.Kind == "image" || entry.Kind == "folder"
	}, func(bool) fyne.CanvasObject { return pane.createRow() }, func(id string, _ bool, object fyne.CanvasObject) { pane.updateRow(id, object) })
	pane.tree.OnBranchOpened = func(id string) {
		pane.workspace.activate(pane.side)
		if !pane.loaded[id] && !pane.pending[id] {
			pane.load(id, "")
		}
	}
	pane.tree.OnSelected = pane.selectNode
}

// createRow returns a full-width name label; the tree supplies expansion and selection controls.
func (pane *locationPane) createRow() fyne.CanvasObject {
	name := widget.NewLabel("")
	name.Truncation = fyne.TextTruncateEllipsis
	return name
}

// updateRow binds the recycled object to id, retaining the full name for truncation and inspection.
func (pane *locationPane) updateRow(id string, object fyne.CanvasObject) {
	object.(*widget.Label).SetText(pane.nodes[id].Name)
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
	if entry.Kind == "more" {
		parent := pane.parents[id]
		pane.load(parent, pane.pages[parent].Next)
		return
	}
	pane.selected = id
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
