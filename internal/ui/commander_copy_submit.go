package ui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"github.com/distribution/reference"
	"github.com/imneov/ImagePort/internal/domain"
)

// specification validates review inputs and captures the supported route without reading widgets later.
func (review *copyReview) specification() (domain.Spec, error) {
	spec := domain.Spec{Local: review.source.entry.Local, Archive: review.source.archive, DestinationEngine: review.engine,
		SourceTargetID: review.source.target.ID, DestinationTargetID: review.target.ID, AccessPinned: true, SourceUsername: review.source.profile.Username}
	if review.source.target.Kind == "filesystem" {
		index := review.entry.SelectedIndex()
		if index < 0 || index >= len(spec.Archive.Entries) {
			return spec, errors.New("Inspect the archive and select an entry first")
		}
		spec.Entry = spec.Archive.Entries[index]
	}
	if review.source.target.Kind == "registry" {
		source, err := review.selectedRemoteSource()
		if err != nil {
			return spec, err
		}
		spec.Source = source
		spec.SourceProfileID, spec.SourceRegistryHost = review.source.profile.ID, review.source.profile.Host
	} else {
		spec.Source.Platform = review.platform.Selected
	}
	if review.target.Kind == "filesystem" {
		return review.archiveSpecification(spec)
	}

	name, tag := strings.TrimSpace(review.name.Text), strings.TrimSpace(review.tag.Text)
	if err := validateReviewName(name, tag); err != nil {
		return spec, err
	}
	spec.Target = name + ":" + tag
	if review.target.Kind == "registry" {
		project := strings.TrimSpace(review.project.Text)
		if project == "" || strings.ContainsAny(project, "/\\ \r\n\t") {
			return spec, errors.New("Choose a destination project before copying")
		}
		spec.Target = project + "/" + spec.Target
		spec.Operation = map[string]string{"registry": "copy", "engine": "push", "filesystem": "import"}[review.source.target.Kind]
	} else {
		spec.Operation = map[string]string{"registry": "pull", "engine": "copy-local", "filesystem": "load"}[review.source.target.Kind]
		if spec.Source.Platform == "" || spec.Source.Platform == "all" {
			return spec, errors.New("Choose one platform to load into the local image engine")
		}
		if spec.Operation == "pull" {
			spec.ProfileID, spec.RegistryHost = review.source.profile.ID, review.source.profile.Host
			spec.ProfileUsername = review.source.profile.Username
		}
	}
	return spec, nil
}

// archiveSpecification captures an explicit export filename without allowing a directory override.
func (review *copyReview) archiveSpecification(spec domain.Spec) (domain.Spec, error) {

	name := strings.TrimSpace(review.filename.Text)
	if strings.ContainsAny(name, "/\\") || !strings.HasSuffix(strings.ToLower(name), ".tar") || name == ".tar" {
		return spec, errors.New("Enter an archive filename ending in .tar, without a directory path")
	}
	spec.Output = filepath.Join(review.outputDirectory, name)
	spec.Operation = "export-local"
	if review.source.target.Kind == "registry" {
		spec.Operation, spec.ProfileID, spec.RegistryHost = "download", review.source.profile.ID, review.source.profile.Host
		spec.ProfileUsername = review.source.profile.Username
	}
	return spec, nil
}

// validateReviewName requires a valid lower-case image name and an explicit non-digest tag.
func validateReviewName(name, tag string) error {
	parsed, err := reference.ParseNormalizedNamed(name + ":" + tag)
	if err != nil || name == "" || tag == "" || strings.ContainsAny(name, "@: ") {
		return errors.New("Enter a valid image name and explicit version tag")
	}
	if _, valid := parsed.(reference.Tagged); !valid {
		return errors.New("Enter a valid image name and explicit version tag")
	}
	return nil
}

// selectedRemoteSource preserves the index when all platforms were chosen explicitly in review.
func (review *copyReview) selectedRemoteSource() (domain.Source, error) {
	platform := review.platform.Selected
	if platform == review.workspace.desktop.tr("All source platforms") && len(review.source.sources) > 0 {
		source := review.source.sources[0]
		if source.IndexDigest != "" {
			source.Digest, source.Platform = source.IndexDigest, "all"
			return source, nil
		}
	}
	for _, source := range review.source.sources {
		if source.Platform == platform {
			return source, nil
		}
	}
	return domain.Source{}, errors.New("Select a platform first")
}

// checkAndSubmit observes the exact target before queuing, asking only for real conflicts.
func (review *copyReview) checkAndSubmit() {
	if review.checking {
		return
	}
	spec, err := review.specification()
	if err != nil {
		review.showFeedback(err.Error())
		return
	}
	review.checking = true
	review.submit.Disable()
	account, reference, revision := review.accountID, review.preview.Text, review.revision
	service := review.workspace.desktop.presenter
	review.work(func(ctx context.Context) (any, error) {
		if review.target.Kind == "registry" {
			profile, err := service.TaskProfile(review.target, account)
			if err != nil {
				return nil, err
			}
			spec.ProfileID, spec.RegistryHost, spec.ProfileUsername = profile.ID, profile.Host, profile.Username
			spec.Target, spec.PreviousDigest, err = service.Target(ctx, profile, spec.Target)
			return spec, err
		}
		if review.target.Kind == "engine" {
			var err error
			spec.PreviousDigest, err = service.Local.TagID(ctx, review.engine, spec.Target)
			return spec, err
		}
		return spec, nil
	}, func(value any) {
		review.checking = false
		review.submit.Enable()
		if revision != review.revision {
			review.invalidate()
			return
		}
		review.confirmObserved(value.(domain.Spec), reference)
	})
}

// confirmObserved requires renewed replacement consent for a changed reference or observed digest.
func (review *copyReview) confirmObserved(spec domain.Spec, reference string) {
	if spec.PreviousDigest != "" && (!review.replace.Checked || review.observedReference != reference || review.previous != spec.PreviousDigest) {
		review.observedReference, review.previous = reference, spec.PreviousDigest
		review.replace.SetChecked(false)
		review.replace.Show()
		review.showFeedback("The destination already has this version. Confirm replacement or choose another name.")
		return
	}
	review.submit.Disable()
	review.work(func(context.Context) (any, error) {
		return review.workspace.desktop.presenter.SubmitWorkspace(spec)
	}, func(value any) {
		review.workspace.desktop.selectedTask = value.(string)
		review.popup.Hide()
		review.workspace.refreshQueue()
	})
}

// work restores review controls after errors and discards callbacks from a dismissed review.
func (review *copyReview) work(work func(context.Context) (any, error), done func(any)) {
	review.workspace.desktop.presenter.Work(work, func(value any, err error) {
		fyne.Do(func() {
			if review.closed || review.workspace.desktop.closing {
				return
			}
			review.checking = false
			review.submit.Enable()
			if err != nil {
				review.showFeedback(err.Error())
				return
			}
			done(value)
		})
	})
}

// showFeedback displays a translated task-specific validation/conflict message in review.
func (review *copyReview) showFeedback(message string) {
	review.feedback.SetText(review.workspace.desktop.translateDetails(message))
	review.feedback.Show()
}
