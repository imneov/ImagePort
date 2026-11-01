package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// locationPane retains browsing, pagination, credentials and selection for one side only.
type locationPane struct {
	activeMarker        *canvas.Rectangle
	initialized         bool
	targetAddress       string
	workspace           *commanderView
	side                int
	targetID, accountID string
	selected            string
	rootKind, rootPath  string
	generation          uint64
	syncing             bool
	nodes               map[string]domain.LocationEntry
	children            map[string][]string
	parents             map[string]string
	loaded, pending     map[string]bool
	pages               map[string]domain.LocationPage
	engine              domain.Engine
	location, account   *widget.Select
	path, filter        *widget.Entry
	notice, summary     *widget.Label
	tree                *widget.Tree
	accountHost         *fyne.Container
	content             fyne.CanvasObject
	targetOptions       []string
	accountIDs          []string
}

// newPane creates independent target, path and tree controls using Fyne's public widgets.
func (view *commanderView) newPane(side int) *locationPane {
	pane := &locationPane{workspace: view, side: side, rootKind: "root"}
	pane.targetID = view.desktop.app.Preferences().String(pane.preferenceKey("target"))
	pane.accountID = pane.rememberedAccount(pane.targetID)
	pane.resetData()
	pane.location = widget.NewSelect(nil, func(string) {
		if !pane.syncing {
			index := pane.location.SelectedIndex()
			if index >= 0 && index < len(view.targets) {
				pane.chooseTarget(view.targets[index].ID, pane.rememberedAccount(view.targets[index].ID))
			}
		}
	})
	pane.account = widget.NewSelect(nil, pane.chooseAccount)
	pane.accountHost = container.NewStack(pane.account)
	pane.path, pane.filter = widget.NewEntry(), widget.NewEntry()
	pane.path.OnSubmitted = pane.openPath
	pane.filter.OnChanged = func(string) { view.activate(side); pane.render() }
	pane.notice, pane.summary = widget.NewLabel(""), widget.NewLabel("")
	pane.notice.Wrapping = fyne.TextWrapWord
	pane.notice.Hide()
	pane.summary.Truncation = fyne.TextTruncateEllipsis
	pane.createTree()
	pane.activeMarker = canvas.NewRectangle(theme.SeparatorColor())
	pane.activeMarker.SetMinSize(fyne.NewSize(0, 2))
	top := container.NewVBox(pane.activeMarker, pane.location, pane.accountHost, pane.navigation(), pane.filter, pane.notice)
	pane.content = container.NewBorder(top, pane.summary, nil, nil, pane.tree)
	return pane
}

// navigation provides an editable concrete path and an explicit parent-folder action.
func (pane *locationPane) navigation() fyne.CanvasObject {
	up := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), pane.up)
	refresh := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), pane.refresh)
	return container.NewBorder(nil, nil, up, refresh, pane.path)
}

// resetData invalidates in-flight reads and removes content from the previous access context.
func (pane *locationPane) resetData() {
	pane.generation++
	pane.nodes, pane.parents = make(map[string]domain.LocationEntry), make(map[string]string)
	pane.children, pane.pages = make(map[string][]string), make(map[string]domain.LocationPage)
	pane.loaded, pane.pending = make(map[string]bool), make(map[string]bool)
	pane.selected = ""
	pane.engine = domain.Engine{}
	if pane.tree != nil {
		pane.tree.UnselectAll()
	}
}

// reloadOptions refreshes shared target names without using display strings as identity.
func (pane *locationPane) reloadOptions() {
	pane.syncing = true
	pane.targetOptions = make([]string, len(pane.workspace.targets))
	selected := -1
	for index, target := range pane.workspace.targets {
		pane.targetOptions[index] = target.Name + " · " + pane.workspace.desktop.tr(targetKind(target.Kind))
		for _, other := range pane.workspace.targets {
			if other.ID != target.ID && other.Name == target.Name && other.Kind == target.Kind {
				pane.targetOptions[index] += " · " + target.Address
				break
			}
		}
		if target.ID == pane.targetID {
			selected = index
		}
	}
	pane.location.SetOptions(pane.targetOptions)
	pane.location.PlaceHolder = pane.workspace.desktop.tr("Choose a target")
	if selected >= 0 {
		pane.location.SetSelectedIndex(selected)
	} else {
		pane.location.ClearSelected()
	}
	previousAccount := pane.accountID
	pane.reloadAccounts()
	pane.path.SetPlaceHolder(pane.workspace.desktop.tr("Location or repository path"))
	pane.filter.SetPlaceHolder(pane.workspace.desktop.tr("Filter this view"))
	pane.syncing = false
	if selected < 0 {
		pane.targetID, pane.accountID, pane.targetAddress = "", "", ""
		pane.resetData()
		pane.render()
		if len(pane.workspace.targets) > 0 {
			pane.selectInitialTarget()
		}
	} else if target := pane.workspace.targets[selected]; pane.targetAddress != target.Address || !pane.initialized {
		pane.chooseTarget(target.ID, pane.accountID)
	} else if previousAccount != pane.accountID {
		pane.refresh()
	}
	pane.workspace.updateCommands()
}

// selectInitialTarget prefers local images on the left and a registry/folder on the right.
func (pane *locationPane) selectInitialTarget() {
	preferred := "engine"
	if pane.side == 1 {
		preferred = "registry"
	}
	for _, target := range pane.workspace.targets {
		if target.Kind == preferred {
			pane.chooseTarget(target.ID, "")
			return
		}
	}
	for _, target := range pane.workspace.targets {
		if pane.side == 0 || target.Kind == "filesystem" {
			pane.chooseTarget(target.ID, "")
			return
		}
	}
}

// chooseTarget changes only this pane, preserving another pane's selected account and content.
func (pane *locationPane) chooseTarget(id, account string) {
	target, valid := pane.workspace.target(id)
	if !valid {
		return
	}
	pane.initialized = true
	pane.targetID, pane.accountID, pane.targetAddress = id, account, target.Address
	pane.rootKind, pane.rootPath = "root", ""
	pane.resetData()
	pane.filter.SetText("")
	pane.reloadOptions()
	pane.path.SetText(target.Address)
	pane.remember()
	pane.workspace.activate(pane.side)
	pane.refresh()
}

// reloadAccounts renders a separate access selector only for remote registry targets.
func (pane *locationPane) reloadAccounts() {
	target, _ := pane.workspace.target(pane.targetID)
	pane.accountHost.Objects = nil
	if target.Kind != "registry" {
		caption := target.Address
		if target.Kind == "engine" && caption == "" {
			caption = pane.workspace.desktop.tr("Current Docker context")
		}
		pane.accountHost.Add(widget.NewLabel(caption))
		return
	}
	accounts := pane.workspace.paneAccounts(target.ID)
	options := []string{pane.workspace.desktop.tr("Anonymous")}
	pane.accountIDs = []string{""}
	selected := 0
	labels := accountOptions(accounts)
	for index, account := range accounts {
		if account.Username == "" {
			continue
		}
		options = append(options, labels[index])
		pane.accountIDs = append(pane.accountIDs, account.ID)
		if account.ID == pane.accountID {
			selected = len(options) - 1
		}
	}
	pane.accountID = pane.accountIDs[selected]
	pane.account.SetOptions(options)
	pane.account.SetSelectedIndex(selected)
	signIn := widget.NewButton(pane.workspace.desktop.tr("Sign in"), func() {
		pane.workspace.accountDialog(target, pane.workspace.account(pane.accountID), func(profile domain.Profile) { pane.chooseTarget(target.ID, profile.ID) })
	})
	pane.accountHost.Add(container.NewBorder(nil, nil, nil, signIn, pane.account))
}

// chooseAccount invalidates both content and outstanding requests when access identity changes.
func (pane *locationPane) chooseAccount(string) {
	if pane.syncing {
		return
	}
	index := pane.account.SelectedIndex()
	if index < 0 || index >= len(pane.accountIDs) {
		return
	}
	pane.accountID = pane.accountIDs[index]
	pane.remember()
	pane.rootKind, pane.rootPath = "root", ""
	pane.refresh()
}

// targetKind maps stable location kinds to translated user-facing names.
func targetKind(kind string) string {
	return map[string]string{"registry": "Image registry", "engine": "Local images", "filesystem": "Folder / disk"}[kind]
}

// matches returns whether an item or one of its loaded descendants matches the filter.
func (pane *locationPane) matches(id, query string) bool {
	entry := pane.nodes[id]
	if query == "" || entry.Kind == "more" || strings.Contains(strings.ToLower(entry.Name+" "+entry.Path), query) {
		return true
	}
	for _, child := range pane.children[id] {
		if pane.matches(child, query) {
			return true
		}
	}
	return false
}
