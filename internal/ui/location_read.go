package ui

import (
	"context"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"github.com/imneov/ImagePort/internal/domain"
)

// locationResponse keeps failures inside the affected pane rather than replacing its neighbor.
type locationResponse struct {
	page domain.LocationPage
	err  error
}

// refresh invalidates pending responses and rereads only this location.
func (pane *locationPane) refresh() {
	pane.workspace.activate(pane.side)
	pane.resetData()
	pane.tree.CloseAllBranches()
	pane.render()
	pane.load("", "")
}

// load captures the target, account, path and generation before starting a background read.
func (pane *locationPane) load(parent, cursor string) {
	if pane.pending[parent] {
		return
	}
	target, valid := pane.workspace.target(pane.targetID)
	if !valid {
		return
	}
	kind, path := pane.rootKind, pane.rootPath
	if parent != "" {
		entry := pane.nodes[parent]
		kind, path = entry.Kind, entry.Path
	}
	account, generation := pane.accountID, pane.generation
	pane.pending[parent] = true
	pane.notice.SetText(pane.workspace.desktop.tr("Loading..."))
	pane.notice.Show()
	service := pane.workspace.desktop.presenter
	service.Work(func(ctx context.Context) (any, error) {
		profile := domain.Profile{}
		if target.Kind == "registry" {
			var err error
			profile, err = service.AccessProfile(target, account)
			if err != nil {
				return locationResponse{err: err}, nil
			}
		}
		page, err := service.BrowseLocation(ctx, target, profile, kind, path, cursor)
		return locationResponse{page, err}, nil
	}, func(value any, err error) {
		fyne.Do(func() {
			if pane.workspace.desktop.closing || pane.generation != generation {
				return
			}
			response := locationResponse{err: err}
			if err == nil {
				response = value.(locationResponse)
			}
			pane.applyPage(parent, cursor, response)
		})
	})
}

// applyPage appends received items and retains the server's cursor/total without repagination.
func (pane *locationPane) applyPage(parent, cursor string, response locationResponse) {
	delete(pane.pending, parent)
	if response.err != nil {
		pane.notice.SetText(pane.workspace.desktop.translateDetails(response.err.Error()))
		pane.notice.Show()
		return
	}
	pane.notice.Hide()
	moreID := "more:" + parent
	children := pane.children[parent][:0]
	for _, id := range pane.children[parent] {
		if id != moreID {
			children = append(children, id)
		}
	}
	pane.children[parent] = children
	delete(pane.nodes, moreID)
	pane.pages[parent], pane.loaded[parent] = response.page, true
	if response.page.Engine.Identity != "" {
		pane.engine = response.page.Engine
	}
	pane.register(parent, response.page.Entries)
	if response.page.Next != "" {
		pane.register(parent, []domain.LocationEntry{{ID: moreID, Name: pane.workspace.desktop.tr("Load more"), Kind: "more"}})
	}
	if cursor == "" && len(response.page.Entries) == 0 {
		pane.notice.SetText(pane.workspace.desktop.tr("This location is empty"))
		pane.notice.Show()
	}
	pane.render()
	pane.workspace.updateCommands()
}

// openPath browses a known repository or directory when catalog enumeration is restricted.
func (pane *locationPane) openPath(input string) {
	target, valid := pane.workspace.target(pane.targetID)
	if !valid {
		return
	}
	input = strings.TrimSpace(input)
	switch target.Kind {
	case "registry":
		host := strings.TrimPrefix(strings.TrimPrefix(target.Address, "https://"), "http://")
		path := strings.TrimPrefix(strings.TrimPrefix(input, target.Address), "/")
		path = strings.TrimPrefix(path, host+"/")
		if path == "" {
			pane.rootKind, pane.rootPath = "root", ""
		} else {
			pane.rootKind, pane.rootPath = "image", path
			if strings.Contains(path[strings.LastIndex(path, "/")+1:], ":") || strings.Contains(path, "@") {
				pane.rootKind = "reference"
			}
		}
	case "filesystem":
		pane.rootKind, pane.rootPath = "folder", input
	case "engine":
		pane.filter.SetText(input)
		return
	}
	pane.refresh()
}

// up returns to a parent container while staying within the selected target.
func (pane *locationPane) up() {
	pane.workspace.activate(pane.side)
	target, valid := pane.workspace.target(pane.targetID)
	if !valid {
		return
	}
	if target.Kind == "filesystem" && pane.rootPath != "" && pane.rootPath != target.Address {
		pane.rootPath = filepath.Dir(pane.rootPath)
	} else {
		pane.rootKind, pane.rootPath = "root", ""
	}
	pane.path.SetText(target.Address)
	if pane.rootPath != "" {
		pane.path.SetText(pane.rootPath)
	}
	pane.refresh()
}

// container returns the selected container or nearest ancestor of the requested kind.
func (pane *locationPane) container(kind string) domain.LocationEntry {
	id := pane.selected
	for id != "" {
		entry := pane.nodes[id]
		if entry.Kind == kind {
			return entry
		}
		id = pane.parents[id]
	}
	return domain.LocationEntry{}
}
