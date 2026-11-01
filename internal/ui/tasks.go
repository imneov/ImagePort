package ui

import (
	"context"
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/storage"
	"github.com/imneov/ImagePort/internal/domain"
)

// updateTaskDetail shows phase, immutable inputs, results and all attempt records.
func (desktop *Desktop) updateTaskDetail() {
	if desktop.taskDetail == nil {
		return
	}
	for _, item := range desktop.tasks {
		if item.ID != desktop.selectedTask {
			continue
		}
		text := desktop.tr("Task %s\nOperation: %s\nState: %s\nSource: %s%s\nTarget: %s%s\nPlatform: %s\nSource index: %s\nSelected manifest: %s\n", item.ID, desktop.tr(item.Spec.Operation), desktop.stateLabel(item.State), item.Spec.Source.Reference, item.Spec.Archive.Path, item.Spec.Target, item.Spec.Output, item.Spec.Source.Platform, item.Spec.Source.IndexDigest, item.Spec.Source.Digest)
		if item.Spec.Operation == "push" {
			text += desktop.tr("Local image ID") + ": " + item.Spec.Local.ImageID + "\n" + desktop.tr("Docker context: %s · %s", item.Spec.Local.Engine.Context, item.Spec.Local.Engine.Host) + "\n"
		}
		for _, attempt := range item.Attempts {
			text += desktop.tr("\nAttempt %d · %s\nStage: %s\nExpected digest: %s\nResult: %s\nDetails: %s\n", attempt.Number, desktop.stateLabel(attempt.State), desktop.tr(attempt.Stage), attempt.ExpectedDigest, desktop.translateDetails(attempt.Result), desktop.translateDetails(attempt.Error))
		}
		desktop.taskDetail.SetText(text)
		if item.State.Active() && desktop.drawer != nil && desktop.currentPage == transfersPage {
			desktop.taskProgress.Show()
			desktop.taskProgress.Start()
		} else {
			desktop.taskProgress.Stop()
			desktop.taskProgress.Hide()
		}
		return
	}
}

// taskAction routes task commands through background application work.
func (desktop *Desktop) taskAction(action string) {
	id := desktop.selectedTask
	if id == "" {
		desktop.inform("Task", "Select a task first")
		return
	}
	if action == "retry" {
		desktop.retryDialog(id)
		return
	}
	desktop.run("Updating task...", func(ctx context.Context) (any, error) {
		if action == "cancel" {
			return nil, desktop.presenter.Tasks.Cancel(id)
		}
		return nil, desktop.presenter.Tasks.Reconcile(ctx, id)
	}, nil)
}

// retryDialog rechecks remote conflicts before adding an attempt with the same source.
func (desktop *Desktop) retryDialog(id string) {
	type consent struct{ previous, target string }
	desktop.run("Rechecking retry target...", func(ctx context.Context) (any, error) {
		item, err := desktop.presenter.Tasks.Get(id)
		if err != nil {
			return nil, err
		}
		if item.State != domain.Failed && item.State != domain.Cancelled {
			return nil, errors.New("Only failed or cancelled tasks can be retried; verify interrupted tasks first")
		}
		if item.Spec.Operation == "download" {
			return consent{}, nil
		}
		profile, err := desktop.presenter.Store.Profile(item.Spec.ProfileID)
		if err != nil {
			return nil, err
		}
		_, previous, err := desktop.presenter.Target(ctx, profile, item.Spec.Target)
		return consent{previous, item.Spec.Target}, err
	}, func(result any) {
		value := result.(consent)
		desktop.confirm("Confirm new retry attempt", desktop.retryMessage(value.target, value.previous), func(confirmed bool) {
			if confirmed {
				desktop.run("Queueing retry...", func(context.Context) (any, error) { return nil, desktop.presenter.Tasks.Retry(id, value.previous) }, nil)
			}
		})
	})
}

// retryMessage renders renewed consent on the UI thread using the current language.
func (desktop *Desktop) retryMessage(target, previous string) string {
	if target == "" {
		return desktop.tr("Create a new attempt with the original manifest digest and platform. Existing destination files will not be overwritten.")
	}
	message := desktop.tr("Keep the original immutable source, even if its tag changed.\nTarget:") + target + desktop.tr("\nCurrent digest:") + previous
	if previous != "" {
		message += desktop.tr("\nWarning: this retry will replace the current target tag.")
	}
	return message
}

// openResultFolder opens only the directory of a successful download task.
func (desktop *Desktop) openResultFolder() {
	item, err := desktop.presenter.Tasks.Get(desktop.selectedTask)
	if err != nil {
		desktop.showError(err)
		return
	}
	if item.State != domain.Succeeded || item.Spec.Output == "" {
		desktop.inform("Download result", "This task has no verified local download result")
		return
	}
	location, err := url.Parse(storage.NewFileURI(filepath.Dir(item.Spec.Output)).String())
	if err == nil {
		err = desktop.app.OpenURL(location)
	}
	if err != nil {
		desktop.showError(err)
	}
}

// stateLabel translates stable machine states without conflating cancellation phases.
func (desktop *Desktop) stateLabel(state domain.State) string {
	labels := map[domain.State]string{domain.Queued: "Queued", domain.Running: "Running", domain.CancelRequested: "Cancellation requested", domain.Succeeded: "Verified success", domain.Failed: "Failed", domain.Cancelled: "Cancelled", domain.Interrupted: "Interrupted / verification needed"}
	if label, found := labels[state]; found {
		return desktop.tr(label)
	}
	return strings.ToLower(string(state))
}
