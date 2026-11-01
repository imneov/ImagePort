package ui

import (
	"context"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/imneov/ImagePort/internal/domain"
)

// inspectSelected reads selected content on demand and retains the two-pane browsing context.
func (view *commanderView) inspectSelected() {
	pane := view.panes[view.active]
	entry, valid := pane.nodes[pane.selected]
	if !valid {
		return
	}
	target, _ := view.target(pane.targetID)
	account, generation := pane.accountID, pane.generation
	if entry.Kind != "version" && entry.Kind != "archive" {
		view.desktop.inform("Details", entry.Path)
		return
	}
	view.desktop.runCurrent("Inspecting selected image...", func() bool { return pane.generation == generation }, func(ctx context.Context) (any, error) {
		return view.readSource(ctx, target, account, entry)
	}, func(value any) {
		source := value.(workspaceSource)
		text := source.target.Name + " / " + source.entry.Path
		for _, manifest := range source.sources {
			text += "\n\n" + manifest.Platform + "\n" + manifest.Digest + "\n" + humanSize(manifest.Size)
		}
		if source.local.ID != "" {
			text += "\n\n" + source.local.Platform + "\n" + source.local.ID + "\n" + humanSize(source.local.Size)
		}
		for _, image := range source.archive.Entries {
			text += "\n\n" + image.Name + "\n" + strings.Join(image.Platforms, ", ")
		}
		label := widget.NewLabel(text)
		label.Wrapping = fyne.TextWrapBreak
		popup := dialog.NewCustom(view.desktop.tr("Image details"), view.desktop.tr("Close"), container.NewVScroll(label), view.desktop.window)
		popup.Resize(fyne.NewSize(700, 440))
		popup.Show()
	})
}

// deleteSelected captures an exact alias/archive and confirms its limited scope before scheduling.
func (view *commanderView) deleteSelected() {
	if view.cut.Disabled() {
		return
	}
	pane := view.panes[view.active]
	entry := pane.nodes[pane.selected]
	target, _ := view.target(pane.targetID)
	account, generation := pane.accountID, pane.generation
	view.desktop.runCurrent("Inspecting selected image...", func() bool { return pane.generation == generation }, func(ctx context.Context) (any, error) {
		return view.readSource(ctx, target, account, entry)
	}, func(value any) {
		source := value.(workspaceSource)
		spec := domain.Spec{AccessPinned: true, ProfileUsername: source.profile.Username, SourceTargetID: target.ID, DestinationTargetID: target.ID, Local: entry.Local, Target: entry.Path, Archive: source.archive}
		spec.Operation = "delete-file"
		if target.Kind == "engine" {
			spec.Operation = "delete-local"
		}
		if target.Kind == "registry" && len(source.sources) > 0 {
			spec.Operation, spec.ProfileID, spec.RegistryHost = "delete-remote", source.profile.ID, source.profile.Host
			spec.Source = source.sources[0]
			if spec.Source.IndexDigest != "" {
				spec.Source.Digest, spec.Source.Platform = spec.Source.IndexDigest, "all"
			}
			spec.PreviousDigest = spec.Source.Digest
		}
		message := target.Name + " / " + entry.Path + "\n\n" + view.desktop.tr("Only this version name or archive file will be removed. Copies in other locations are kept.")
		view.desktop.confirm("Delete selected item?", message, func(confirmed bool) {
			if confirmed {
				view.desktop.run("Queueing image cleanup...", func(context.Context) (any, error) { return view.desktop.presenter.SubmitWorkspace(spec) }, func(any) { view.refreshQueue() })
			}
		})
	})
}

// retryTask renews destination consent without substituting a new source or access identity.
func (view *commanderView) retryTask(item domain.Task) {
	view.desktop.run("Rechecking retry target...", func(ctx context.Context) (any, error) {
		previous, err := view.desktop.presenter.RecheckTaskTarget(ctx, item)
		return previous, err
	}, func(value any) {
		previous := value.(string)
		source, target := taskLocations(item)
		message := source + "\n→ " + target
		if previous != "" && !strings.HasPrefix(item.Spec.Operation, "delete-") {
			message += "\n\n" + view.desktop.tr("This retry will replace the current destination version.")
		}
		view.desktop.confirm("Confirm new retry attempt", message, func(confirmed bool) {
			if confirmed {
				view.desktop.run("Queueing retry...", func(context.Context) (any, error) { return nil, view.desktop.presenter.Tasks.Retry(item.ID, previous) }, func(any) { view.refreshQueue() })
			}
		})
	})
}
