package ui

import (
	"context"
	"github.com/imneov/ImagePort/internal/domain"
	"strings"
)

// confirmRemote displays current target digest and obtains consent before enqueueing.
func (desktop *Desktop) confirmRemote(profile domain.Profile, spec domain.Spec) {
	desktop.confirmRemoteThen(profile, spec, nil)
}

// confirmRemoteThen closes the originating flow only after a confirmed task is durably queued.
func (desktop *Desktop) confirmRemoteThen(profile domain.Profile, spec domain.Spec, queued func()) {
	spec.RegistryHost = profile.Host
	type checked struct{ target, previous string }
	desktop.run("Checking target tag conflicts...", func(ctx context.Context) (any, error) {
		target, previous, err := desktop.presenter.Target(ctx, profile, spec.Target)
		return checked{target, previous}, err
	}, func(result any) {
		value := result.(checked)
		spec.Target, spec.PreviousDigest = value.target, value.previous
		source := spec.Source.Reference + "\n" + spec.Source.Digest + desktop.tr("\nPlatform:") + spec.Source.Platform
		if spec.Operation == "import" {
			source = spec.Archive.Path + desktop.tr("\nEntry:") + spec.Entry.Name + desktop.tr("\nPlatforms:") + strings.Join(spec.Entry.Platforms, ", ")
		}
		if spec.Operation == "push" {
			source = spec.Source.Reference + "\n" + desktop.tr("Local image ID") + ": " + spec.Local.ImageID + "\n" + desktop.tr("Docker context: %s · %s", spec.Local.Engine.Context, spec.Local.Engine.Host)
		}
		message := desktop.tr("Source:") + source + desktop.tr("\n\nTarget:") + value.target
		if value.previous != "" {
			message += desktop.tr("\n\nWarning: the existing target tag will be replaced.\nCurrent digest:") + value.previous
		} else {
			message += desktop.tr("\nThe target tag does not currently exist")
		}
		message += desktop.tr("\n\nConflicts are checked again before execution. Other clients can still modify the target concurrently.")
		desktop.confirm("Confirm image operation", message, func(confirmed bool) {
			if confirmed {
				desktop.run("Queueing task...", func(context.Context) (any, error) { return desktop.presenter.SubmitRemote(spec) }, func(any) {
					if queued != nil {
						queued()
					}
					desktop.show(transfersPage)
				})
			}
		})
	})
}
