// Package transfer coordinates validation, SDK operations and verified results.
package transfer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/local"
	"github.com/imneov/ImagePort/internal/registry"
	"github.com/imneov/ImagePort/internal/task"
)

// Profiles supplies saved metadata at execution time rather than caching passwords.
type Profiles interface {
	Profile(string) (domain.Profile, error)
}

// Runner orchestrates actual transfer phases without depending on Fyne.
type Runner struct {
	Registry registry.Port
	Profiles Profiles
	Limits   archive.Limits
	Logger   *slog.Logger
	Local    local.Port
}

// Run executes the confirmed operation, checkpointing expected digests before writes.
func (runner *Runner) Run(ctx context.Context, item domain.Task, checkpoint task.Checkpoint) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if item.Spec.Operation == "load" || item.Spec.Operation == "copy-local" || item.Spec.Operation == "export-local" || item.Spec.Operation == "delete-local" || item.Spec.Operation == "delete-file" {
		return runner.runLocal(ctx, item.Spec, checkpoint)
	}
	profile, err := runner.Profiles.Profile(item.Spec.ProfileID)
	if err != nil {
		return "", err
	}
	if err := checkProfileIdentity(profile, item.Spec, false); err != nil {
		return "", err
	}
	if profile.Host != item.Spec.RegistryHost {
		return "", errors.New("The registry address changed after confirmation; create a new task")
	}
	if err := checkpoint("preflight", "", ""); err != nil {
		return "", err
	}
	switch item.Spec.Operation {
	case "download":
		return runner.download(ctx, profile, item.Spec, checkpoint)
	case "copy":
		return runner.copy(ctx, profile, item.Spec, checkpoint)
	case "import":
		return runner.upload(ctx, profile, item.Spec, checkpoint)
	case "push":
		return runner.push(ctx, profile, item.Spec, checkpoint)
	case "pull":
		return runner.pull(ctx, profile, item.Spec, checkpoint)
	case "delete-remote":
		port, valid := runner.Registry.(registry.TagRemover)
		if !valid {
			return "", errors.New("This registry does not support deleting a single tag")
		}
		err := port.DeleteTag(ctx, profile, item.Spec.Target, item.Spec.PreviousDigest)
		if registry.IsCode(err, "UNKNOWN_RESULT") {
			return "", &task.Unknown{Reason: err.Error()}
		}
		return item.Spec.Target, err
	default:
		return "", errors.New("Unsupported image management operation")
	}
}

// targetUnchanged verifies the previously observed tag before an explicit write.
func (runner *Runner) targetUnchanged(ctx context.Context, profile domain.Profile, spec domain.Spec) error {
	digest, err := runner.Registry.Digest(ctx, profile, spec.Target)
	if registry.IsCode(err, "NOT_FOUND") {
		digest, err = "", nil
	}
	if err != nil {
		return err
	}
	if digest != spec.PreviousDigest {
		return errors.New("The target tag changed after confirmation; confirm replacement again. No target write was made")
	}
	return nil
}

// remoteResult independently reads back after success, error or cancellation.
func (runner *Runner) remoteResult(profile domain.Profile, target, expected string, writeErr error) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	actual, err := runner.Registry.Digest(ctx, profile, target)
	if err == nil && actual == expected {
		return target + "\n" + actual, nil
	}
	if err != nil && !registry.IsCode(err, "NOT_FOUND") {
		return "", &task.Unknown{Reason: "Cannot read back the remote result; check permissions or network access and verify before retrying"}
	}
	if errors.Is(writeErr, context.Canceled) {
		return "Remote blobs may remain; ImagePort will not delete them automatically", context.Canceled
	}
	if writeErr != nil {
		return "", writeErr
	}
	return "", &task.Unknown{Reason: "The target digest differs from the expected result; check whether another client changed the tag"}
}

// copy pins the selected source digest and retains the original tag.
func (runner *Runner) copy(ctx context.Context, profile domain.Profile, spec domain.Spec, checkpoint task.Checkpoint) (string, error) {
	if spec.Source.Digest == "" || spec.Source.Platform == "" {
		return "", errors.New("Resolve the image and explicitly select a platform first")
	}
	if err := runner.targetUnchanged(ctx, profile, spec); err != nil {
		return "", err
	}
	if err := checkpoint("transfer", spec.Source.Digest, ""); err != nil {
		return "", err
	}
	writeErr := runner.copySource(ctx, profile, spec)
	if err := checkpoint("verify", "", ""); err != nil {
		return "", &task.Unknown{Reason: err.Error()}
	}
	return runner.remoteResult(profile, spec.Target, spec.Source.Digest, writeErr)
}

// copySource resolves the captured source connection, rejecting changed addresses before writes.
func (runner *Runner) copySource(ctx context.Context, target domain.Profile, spec domain.Spec) error {
	if spec.SourceProfileID == "" {
		return runner.Registry.Copy(ctx, target, spec.Source, spec.Target)
	}
	source, err := runner.Profiles.Profile(spec.SourceProfileID)
	if err != nil {
		return err
	}
	if err := checkProfileIdentity(source, spec, true); err != nil {
		return err
	}
	if source.Host != spec.SourceRegistryHost {
		return errors.New("The source registry address changed; select the source again")
	}
	port, supported := runner.Registry.(registry.CrossRegistryCopier)
	if !supported {
		return errors.New("The registry adapter does not support copying between connections")
	}
	return port.CopyBetween(ctx, source, target, spec.Source, spec.Target)
}

// Reconcile reads persisted expected results, never blindly repeats a write.
func (runner *Runner) Reconcile(ctx context.Context, item domain.Task) (bool, error) {
	if item.Spec.Operation == "load" || item.Spec.Operation == "copy-local" || item.Spec.Operation == "pull" || item.Spec.Operation == "export-local" || strings.HasPrefix(item.Spec.Operation, "delete-") {
		return runner.reconcileLocal(ctx, item)
	}
	if item.Spec.Operation == "download" {
		return runner.checkDownload(ctx, item.Spec)
	}
	expected := item.Current().ExpectedDigest
	if expected == "" {
		return false, nil
	}
	profile, err := runner.Profiles.Profile(item.Spec.ProfileID)
	if err != nil {
		return false, err
	}
	if profile.Host != item.Spec.RegistryHost {
		return false, errors.New("The registry address changed; restore the original connection before verification")
	}
	actual, err := runner.Registry.Digest(ctx, profile, item.Spec.Target)
	if registry.IsCode(err, "NOT_FOUND") {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if actual != expected && actual != item.Spec.PreviousDigest {
		return false, fmt.Errorf("The target contains different content; check its current digest first: %s", actual)
	}
	return actual == expected, nil
}

// checkProfileIdentity prevents saved metadata edits from silently changing a task's account.
func checkProfileIdentity(profile domain.Profile, spec domain.Spec, source bool) error {
	if !spec.AccessPinned {
		return nil
	}
	expected := spec.ProfileUsername
	if source {
		expected = spec.SourceUsername
	}
	if profile.Username != expected {
		return errors.New("The task account changed; restore the original account or create a new task")
	}
	return nil
}
