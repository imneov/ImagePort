package ui

import (
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// tasksPage presents a virtualized transfer table with actual state and stage information.
func (desktop *Desktop) tasksPage() fyne.CanvasObject {
	desktop.taskDetail = widget.NewLabel("")
	desktop.taskDetail.Wrapping = fyne.TextWrapBreak
	desktop.taskProgress = widget.NewProgressBarInfinite()
	desktop.taskProgress.Stop()
	desktop.taskProgress.Hide()
	desktop.taskSummary = widget.NewLabel("")
	desktop.taskList = widget.NewList(func() int { return len(desktop.taskRows) }, desktop.createTaskRow, desktop.updateTaskRow)
	desktop.taskList.OnSelected = func(index widget.ListItemID) {
		if desktop.syncingTaskSelection {
			return
		}
		desktop.selectedTask = desktop.taskRows[index].ID
		desktop.highlightedTask, desktop.highlightedIndex = desktop.selectedTask, index
		desktop.openTaskDrawer()
	}
	filters := container.NewHBox()
	for _, name := range []string{"All transfers", "Active", "Completed", "Needs attention"} {
		filter := name
		filters.Add(desktop.button(name, func() { desktop.taskFilter = filter; desktop.filterTasks() }))
	}
	resume := desktop.button("Resume restored queue", func() {
		desktop.presenter.Tasks.Resume()
		desktop.setStatus("Queue resumed; inputs will be checked before execution")
	})
	header := container.NewBorder(nil, nil, nil, container.NewCenter(resume), desktop.heading("Transfers", "Track downloads, copies and uploads. Unknown totals show the current stage."))
	headings := columns([]float32{4, 1.5, 2.5, 2}, desktop.label("Image / destination"), desktop.label("Operation"), desktop.label("Status / stage"), desktop.label("Created"))
	desktop.bindText(desktop.filterTasks)
	return container.NewBorder(container.NewVBox(header, filters, widget.NewSeparator(), headings), desktop.taskSummary, nil, nil, desktop.taskList)
}

// createTaskRow provides aligned cells for a recycled transfer row.
func (desktop *Desktop) createTaskRow() fyne.CanvasObject {
	cells := make([]fyne.CanvasObject, 4)
	for index := range cells {
		label := widget.NewLabel("")
		label.Truncation = fyne.TextTruncateEllipsis
		cells[index] = label
	}
	return columns([]float32{4, 1.5, 2.5, 2}, cells...)
}

// updateTaskRow shows immutable source/target and durable state for this row identity.
func (desktop *Desktop) updateTaskRow(index widget.ListItemID, object fyne.CanvasObject) {
	item := desktop.taskRows[index]
	cells := object.(*fyne.Container).Objects
	source, target := taskLocations(item)
	values := []string{source + "\n" + target, desktop.tr(item.Spec.Operation), desktop.stateLabel(item.State) + "\n" + desktop.tr(item.Current().Stage), item.CreatedAt.Local().Format("Jan 02\n15:04")}
	for column, value := range values {
		cells[column].(*widget.Label).SetText(value)
	}
}

// taskLocations returns displayable source and destination without including credentials.
func taskLocations(item domain.Task) (string, string) {
	source := item.Spec.Source.Reference
	if item.Spec.Archive.Path != "" {
		source = filepath.Base(item.Spec.Archive.Path)
	}
	if item.Spec.Local.ImageID != "" {
		source = item.Spec.Local.ImageID
	}
	target := item.Spec.Target
	if item.Spec.Output != "" {
		target = item.Spec.Output
	}
	return source, target
}

// refreshTasks retains the previous snapshot on polling errors and refreshes derived views.
func (desktop *Desktop) refreshTasks(items []domain.Task, err error) {
	if err != nil {
		desktop.setStatus(err.Error())
		return
	}
	if desktop.closing {
		return
	}
	desktop.tasks = items
	if desktop.commander != nil {
		desktop.commander.acceptTasks(items)
		return
	}
	desktop.filterTasks()
	desktop.updateTaskDetail()
	desktop.refreshTaskDrawer()
	desktop.updateActivity()
	desktop.refreshRecentTasks()
	if desktop.commander != nil {
		desktop.commander.acceptTasks(items)
	}
	if desktop.presenter.Tasks.Paused() {
		desktop.setStatus("History restored; queued tasks await explicit resumption")
	}
}

// filterTasks derives visible rows without modifying or slicing durable task history.
func (desktop *Desktop) filterTasks() {
	desktop.taskRows = nil
	for _, item := range desktop.tasks {
		active := item.State.Active() || item.State == domain.Queued
		if desktop.taskFilter == "Active" && !active {
			continue
		}
		if desktop.taskFilter == "Completed" && item.State != domain.Succeeded {
			continue
		}
		if desktop.taskFilter == "Needs attention" && (active || item.State == domain.Succeeded) {
			continue
		}
		desktop.taskRows = append(desktop.taskRows, item)
	}
	if desktop.taskList != nil {
		desktop.taskList.Refresh()
		desktop.syncTaskSelection()
		desktop.taskSummary.SetText(desktop.tr("%d visible / %d transfers", len(desktop.taskRows), len(desktop.tasks)))
	}
}

// openTaskDrawer exposes inspection and safe task controls alongside the transfers table.
func (desktop *Desktop) openTaskDrawer() {
	desktop.updateTaskDetail()
	cancel := widget.NewButton(desktop.tr("Request cancellation"), func() { desktop.taskAction("cancel") })
	retry := widget.NewButton(desktop.tr("Retry"), func() { desktop.taskAction("retry") })
	verify := widget.NewButton(desktop.tr("Verify interrupted result"), func() { desktop.taskAction("reconcile") })
	open := widget.NewButtonWithIcon(desktop.tr("Open download folder"), theme.FolderOpenIcon(), desktop.openResultFolder)
	copy := widget.NewButtonWithIcon(desktop.tr("Copy task details"), theme.ContentCopyIcon(), func() { desktop.window.Clipboard().SetContent(desktop.taskDetail.Text) })
	actions := container.NewVBox(container.NewGridWithColumns(3, cancel, retry, verify), open, copy)
	desktop.openDrawer("Transfer details", container.NewBorder(desktop.taskProgress, nil, nil, nil, container.NewVScroll(desktop.taskDetail)), actions)
	desktop.drawerRefresh = func() {
		cancel.SetText(desktop.tr("Request cancellation"))
		retry.SetText(desktop.tr("Retry"))
		verify.SetText(desktop.tr("Verify interrupted result"))
		open.SetText(desktop.tr("Open download folder"))
		copy.SetText(desktop.tr("Copy task details"))
		for _, item := range desktop.tasks {
			if item.ID != desktop.selectedTask {
				continue
			}
			cancel.Disable()
			retry.Disable()
			verify.Disable()
			open.Disable()
			if item.State.Active() || item.State == domain.Queued {
				cancel.Enable()
			}
			if item.State == domain.Cancelled || item.State == domain.Failed {
				retry.Enable()
			}
			if item.State == domain.Interrupted {
				verify.Enable()
			}
			if item.State == domain.Succeeded && item.Spec.Operation == "download" {
				open.Enable()
			}
		}
	}
	desktop.drawerRefresh()
	desktop.updateTaskDetail()
}

// refreshRecentTasks renders at most two real recent tasks with links to their details.
func (desktop *Desktop) refreshRecentTasks() {
	if desktop.recentTasks == nil {
		return
	}
	desktop.recentTasks.Objects = nil
	if len(desktop.tasks) == 0 {
		desktop.recentTasks.Add(widget.NewLabel(desktop.tr("No transfers yet. Downloads and uploads will appear here.")))
	}
	for offset := 0; offset < min(2, len(desktop.tasks)); offset++ {
		item := desktop.tasks[len(desktop.tasks)-1-offset]
		id := item.ID
		source, _ := taskLocations(item)
		label := widget.NewLabel(fmt.Sprintf("%s  ·  %s  ·  %s", desktop.tr(item.Spec.Operation), source, desktop.stateLabel(item.State)))
		label.Truncation = fyne.TextTruncateEllipsis
		action := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() { desktop.show(transfersPage); desktop.selectedTask = id; desktop.openTaskDrawer() })
		desktop.recentTasks.Add(container.NewBorder(nil, nil, nil, action, label))
	}
	desktop.recentTasks.Refresh()
}

// refreshTaskDrawer updates contextual action availability after state transitions.
func (desktop *Desktop) refreshTaskDrawer() {
	if desktop.drawerRefresh != nil && desktop.currentPage == transfersPage {
		desktop.drawerRefresh()
	}
}

// syncTaskSelection remaps only changed identities/indices, preserving the user's scroll position.
func (desktop *Desktop) syncTaskSelection() {
	selected := -1
	for index, item := range desktop.taskRows {
		if item.ID == desktop.selectedTask {
			selected = index
			break
		}
	}
	if selected == desktop.highlightedIndex && desktop.highlightedTask == desktop.selectedTask {
		return
	}
	desktop.syncingTaskSelection = true
	desktop.taskList.UnselectAll()
	if selected >= 0 {
		desktop.taskList.Select(selected)
	}
	desktop.highlightedTask, desktop.highlightedIndex = desktop.selectedTask, selected
	desktop.syncingTaskSelection = false
}
