package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
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
	}, func(bool) fyne.CanvasObject { return pane.createRow() }, func(id string, _ bool, object fyne.CanvasObject) { pane.updateRow(id, object) })
	pane.tree.OnBranchOpened = func(id string) {
		pane.workspace.activate(pane.side)
		if !pane.loaded[id] && !pane.pending[id] {
			pane.load(id, "")
		}
	}
	pane.tree.OnSelected = pane.selectNode
}

// createRow returns a recycled icon-and-name row.
// A Button paints an icon and a caption in one widget, so virtualised rebinding stays consistent.
func (pane *locationPane) createRow() fyne.CanvasObject {
	button := widget.NewButton("", nil)
	button.Alignment = widget.ButtonAlignLeading
	button.Importance = widget.LowImportance
	return button
}

// updateRow binds the recycled object to id, retaining the full name for truncation and inspection.
func (pane *locationPane) updateRow(id string, object fyne.CanvasObject) {
	button, ok := object.(*widget.Button)
	if !ok {
		return
	}
	entry := pane.nodes[id]
	button.SetText(entry.Name)
	button.SetIcon(pane.rowIcon(entry))
}

// rowIcon chooses a distinct glyph per entry kind without inventing remote metadata.
func (pane *locationPane) rowIcon(entry domain.LocationEntry) fyne.Resource {
	switch entry.Kind {
	case "parent":
		return theme.NavigateBackIcon()
	case "folder", "project":
		return theme.FolderIcon()
	case "image":
		return theme.StorageIcon()
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
