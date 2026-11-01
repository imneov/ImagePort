package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// transferQueue creates a compact durable-task list, hidden when there is no current work.
func (view *commanderView) transferQueue() *fyne.Container {
	view.queueList = widget.NewList(func() int { return len(view.queueRows) }, view.createQueueRow, func(index widget.ListItemID, object fyne.CanvasObject) {
		view.updateQueueRow(view.queueRows[index], object)
	})
	view.queueList.OnSelected = func(index widget.ListItemID) { view.showTask(view.queueRows[index].ID) }
	header := container.NewBorder(nil, nil, view.desktop.label("Transfers"), view.desktop.button("View history", view.showHistory))
	content := container.NewBorder(header, nil, nil, nil, view.queueList)
	queue := container.New(&queueLayout{}, content)
	queue.Hide()
	return queue
}

// createQueueRow supplies compact source/destination/status cells and a detail action.
func (view *commanderView) createQueueRow() fyne.CanvasObject {
	name, route, status := widget.NewLabel(""), widget.NewLabel(""), widget.NewLabel("")
	name.Truncation, route.Truncation, status.Truncation = fyne.TextTruncateEllipsis, fyne.TextTruncateEllipsis, fyne.TextTruncateEllipsis
	return container.New(&queueColumns{}, name, route, status, widget.NewButtonWithIcon("", theme.InfoIcon(), nil))
}

// updateQueueRow shows real stage/state information, never simulated percentages.
func (view *commanderView) updateQueueRow(item domain.Task, object fyne.CanvasObject) {
	cells := object.(*fyne.Container).Objects
	source, target := taskLocations(item)
	cells[0].(*widget.Label).SetText(source)
	cells[1].(*widget.Label).SetText(target)
	cells[2].(*widget.Label).SetText(view.desktop.stateLabel(item.State) + " · " + view.desktop.tr(item.Current().Stage))
	cells[3].(*widget.Button).OnTapped = func() { view.showTask(item.ID) }
}

// refreshQueue updates visible work and refreshes an affected pane only when it is still idle there.
func (view *commanderView) refreshQueue() {
	items, err := view.desktop.presenter.Tasks.List()
	if err != nil {
		view.desktop.showError(err)
		return
	}
	view.acceptTasks(items)
}

// acceptTasks refreshes terminal changes without navigating or clearing the user's current selection.
func (view *commanderView) acceptTasks(items []domain.Task) {
	view.desktop.tasks = items
	view.queueRows = nil
	for _, item := range items {
		if item.State.Active() || item.State == domain.Queued || len(view.queueRows) < 5 {
			view.queueRows = append(view.queueRows, item)
		}
		if previous, seen := view.seen[item.ID]; seen && previous != item.State && item.State == domain.Succeeded {
			for _, pane := range view.panes {
				if pane.targetID == item.Spec.DestinationTargetID {
					pane.notice.SetText(view.desktop.tr("Transfer completed. Refresh to view the result."))
					pane.notice.Show()
				}
			}
		}
		view.seen[item.ID] = item.State
	}
	view.queueList.Refresh()
	if len(view.queueRows) > 0 {
		view.queue.Show()
	} else {
		view.queue.Hide()
	}
	view.desktop.activity.SetText(view.desktop.tr("Transfers") + fmt.Sprintf(" · %d", len(items)))
}

// showHistory displays durable task history without replacing the two locations.
func (view *commanderView) showHistory() {
	items := append([]domain.Task(nil), view.desktop.tasks...)
	list := widget.NewList(func() int { return len(items) }, view.createQueueRow, func(index widget.ListItemID, object fyne.CanvasObject) { view.updateQueueRow(items[index], object) })
	list.OnSelected = func(index widget.ListItemID) { view.showTask(items[index].ID) }
	resume := widget.NewButton(view.desktop.tr("Resume restored queue"), func() { view.desktop.presenter.Tasks.Resume(); view.refreshQueue() })
	if !view.desktop.presenter.Tasks.Paused() {
		resume.Hide()
	}
	content := container.NewBorder(resume, nil, nil, nil, list)
	if len(items) == 0 {
		content = container.NewStack(container.NewCenter(widget.NewLabel(view.desktop.tr("No transfers"))))
	}
	popup := dialog.NewCustom(view.desktop.tr("Transfers"), view.desktop.tr("Close"), content, view.desktop.window)
	popup.Resize(fyne.NewSize(980, 500))
	popup.Show()
}

// showTask presents captured inputs and real task controls in a modal above the workspace.
func (view *commanderView) showTask(id string) {
	item, err := view.desktop.presenter.Tasks.Get(id)
	if err != nil {
		view.desktop.showError(err)
		return
	}
	source, target := taskLocations(item)
	text := source + "\n→ " + target + "\n" + view.desktop.stateLabel(item.State) + " · " + view.desktop.tr(item.Current().Stage)
	text += "\n" + view.desktop.tr("Platform") + ": " + item.Spec.Source.Platform
	text += view.taskAccounts(item.Spec)
	for _, attempt := range item.Attempts {
		text += fmt.Sprintf("\n\n#%d · %s\n%s\n%s", attempt.Number, view.desktop.stateLabel(attempt.State), view.desktop.translateDetails(attempt.Result), view.desktop.translateDetails(attempt.Error))
	}
	details := widget.NewLabel(text)
	details.Wrapping = fyne.TextWrapBreak
	actions := container.NewHBox()
	var popup *dialog.CustomDialog
	if item.State.Active() || item.State == domain.Queued {
		actions.Add(widget.NewButton(view.desktop.tr("Request cancellation"), func() { view.taskControl(id, "cancel"); popup.Hide() }))
	}
	if item.State == domain.Failed || item.State == domain.Cancelled {
		actions.Add(widget.NewButton(view.desktop.tr("Retry"), func() { popup.Hide(); view.retryTask(item) }))
	}
	if item.State == domain.Interrupted {
		actions.Add(widget.NewButton(view.desktop.tr("Verify interrupted result"), func() { view.taskControl(id, "verify"); popup.Hide() }))
	}
	if item.State == domain.Succeeded && !strings.HasPrefix(item.Spec.Operation, "delete-") {
		actions.Add(widget.NewButton(view.desktop.tr("Open destination"), func() { popup.Hide(); view.openDestination(item) }))
	}
	popup = dialog.NewCustom(view.desktop.tr("Transfer details"), view.desktop.tr("Close"), container.NewBorder(nil, actions, nil, nil, container.NewVScroll(details)), view.desktop.window)
	popup.Resize(fyne.NewSize(720, 500))
	popup.Show()
}

// taskControl performs cancellation or read-only reconciliation through the scheduler.
func (view *commanderView) taskControl(id, action string) {
	view.desktop.run("Updating task...", func(ctx context.Context) (any, error) {
		if action == "cancel" {
			return nil, view.desktop.presenter.Tasks.Cancel(id)
		}
		return nil, view.desktop.presenter.Tasks.Reconcile(ctx, id)
	}, func(any) { view.refreshQueue() })
}

// openDestination opens the captured target explicitly, rather than following later pane choices.
func (view *commanderView) openDestination(item domain.Task) {
	target, valid := view.target(item.Spec.DestinationTargetID)
	if !valid {
		view.desktop.inform("Target unavailable", "This task's target is no longer configured")
		return
	}
	pane := view.panes[1-view.active]
	account := ""
	if target.Kind == "registry" {
		account = item.Spec.ProfileID
	}
	pane.chooseTarget(target.ID, account)
	if target.Kind == "filesystem" && item.Spec.Output != "" {
		pane.openPath(filepath.Dir(item.Spec.Output))
	}
}

// queueLayout reserves a small fixed-height transfer area without constraining its width.
type queueLayout struct{}

// Layout fills the queue with its sole bordered content object.
func (*queueLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, object := range objects {
		object.Move(fyne.NewPos(0, 0))
		object.Resize(size)
	}
}

// MinSize allocates two compact rows below the image browser.
func (*queueLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(0, 140) }

// queueColumns allocates name, exact target, stage and action columns responsively.
type queueColumns struct{}

// Layout distributes queue cells while keeping the detail control usable.
func (*queueColumns) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	widths := []float32{size.Width * .30, size.Width*.45 - 36, size.Width * .25, 36}
	offset := float32(0)
	for index, object := range objects {
		object.Move(fyne.NewPos(offset, 0))
		object.Resize(fyne.NewSize(widths[index], size.Height))
		offset += widths[index]
	}
}

// MinSize keeps queue rows compact and prevents long references from widening the window.
func (*queueColumns) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(560, 36) }

// accountCaption translates anonymous access while keeping the captured username literal.
func (view *commanderView) accountCaption(username string) string {
	if username == "" {
		return view.desktop.tr("Anonymous")
	}
	return username
}

// taskAccounts shows captured access identities only for registry endpoints involved in this route.
func (view *commanderView) taskAccounts(spec domain.Spec) string {
	if !spec.AccessPinned {
		return ""
	}
	text := ""
	if spec.SourceProfileID != "" {
		text += "\n" + view.desktop.tr("Source account") + ": " + view.accountCaption(spec.SourceUsername)
	}
	if spec.ProfileID != "" && spec.Operation != "download" && spec.Operation != "pull" {
		text += "\n" + view.desktop.tr("Destination account") + ": " + view.accountCaption(spec.ProfileUsername)
	}
	return text
}
