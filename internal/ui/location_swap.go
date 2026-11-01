package ui

import "github.com/imneov/ImagePort/internal/domain"

// paneSnapshot carries browsing state without moving widgets or their side-specific callbacks.
type paneSnapshot struct {
	targetID, accountID, rootKind, rootPath, selected, path, filter string
	engine                                                          domain.Engine
	nodes                                                           map[string]domain.LocationEntry
	children                                                        map[string][]string
	parents                                                         map[string]string
	loaded                                                          map[string]bool
	pages                                                           map[string]domain.LocationPage
	branches                                                        []string
}

// snapshot captures loaded content and expanded branches for an explicit location exchange.
func (pane *locationPane) snapshot() paneSnapshot {
	state := paneSnapshot{targetID: pane.targetID, accountID: pane.accountID, rootKind: pane.rootKind, rootPath: pane.rootPath,
		selected: pane.selected, path: pane.path.Text, filter: pane.filter.Text,
		engine: pane.engine, nodes: pane.nodes, children: pane.children, parents: pane.parents, loaded: pane.loaded, pages: pane.pages}
	for id := range pane.nodes {
		if pane.tree.IsBranchOpen(id) {
			state.branches = append(state.branches, id)
		}
	}
	return state
}

// restore swaps received content, discarding old reads and resuming unfinished expanded containers.
func (pane *locationPane) restore(state paneSnapshot) {
	pane.generation++
	pane.tree.CloseAllBranches()
	pane.targetID, pane.accountID, pane.rootKind, pane.rootPath = state.targetID, state.accountID, state.rootKind, state.rootPath
	target, _ := pane.workspace.target(state.targetID)
	pane.targetAddress, pane.initialized = target.Address, true
	pane.nodes, pane.children, pane.parents, pane.loaded, pane.pages = state.nodes, state.children, state.parents, state.loaded, state.pages
	pane.pending = make(map[string]bool)
	pane.engine, pane.selected = state.engine, state.selected
	pane.path.SetText(state.path)
	pane.filter.SetText(state.filter)
	pane.reloadOptions()
	pane.render()
	for _, id := range state.branches {
		pane.tree.OpenBranch(id)
	}
	if state.selected != "" {
		pane.tree.Select(state.selected)
	}
	if !pane.loaded[""] {
		pane.load("", "")
	}
	pane.remember()
}
