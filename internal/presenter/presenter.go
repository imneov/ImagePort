// Package presenter translates desktop intent into validated application commands.
package presenter

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	dockeradapter "github.com/imneov/ImagePort/internal/adapter/docker"
	adapter "github.com/imneov/ImagePort/internal/adapter/regclient"
	"github.com/imneov/ImagePort/internal/archive"
	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/local"
	"github.com/imneov/ImagePort/internal/registry"
	"github.com/imneov/ImagePort/internal/secret"
	"github.com/imneov/ImagePort/internal/store"
	"github.com/imneov/ImagePort/internal/task"
)

// Presenter is the desktop's application boundary; it owns background read work.
type Presenter struct {
	Store         *store.Store
	Vault         *secret.Vault
	Registry      registry.Port
	Local         local.Port
	Tasks         *task.Manager
	Limits        archive.Limits
	ctx           context.Context
	cancel        context.CancelFunc
	workers       sync.WaitGroup
	reads         sync.WaitGroup
	credentialsMu sync.Mutex
	targetsMu     sync.Mutex
}

// New assembles a presenter from initialized services without creating UI objects.
func New(database *store.Store, vault *secret.Vault, port registry.Port, tasks *task.Manager) *Presenter {
	ctx, cancel := context.WithCancel(context.Background())
	return &Presenter{Store: database, Vault: vault, Registry: port, Local: dockeradapter.New(), Tasks: tasks,
		Limits: archive.DefaultLimits(), ctx: ctx, cancel: cancel}
}

// Work executes read/form work off the UI thread; completion must dispatch UI updates.
func (presenter *Presenter) Work(work func(context.Context) (any, error), complete func(any, error)) {
	presenter.workers.Add(1)
	presenter.reads.Add(1)
	go func() {
		defer presenter.workers.Done()
		defer presenter.reads.Done()
		ctx, cancel := context.WithTimeout(presenter.ctx, 2*time.Minute)
		defer cancel()
		result, err := work(ctx)
		if presenter.ctx.Err() == nil {
			complete(result, err)
		}
	}()
}

// Observe publishes durable snapshots; no terminal event can be displaced by progress.
func (presenter *Presenter) Observe(update func([]domain.Task, error)) {
	presenter.workers.Add(1)
	go func() {
		defer presenter.workers.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-presenter.ctx.Done():
				return
			case <-ticker.C:
				items, err := presenter.Tasks.List()
				update(items, err)
			}
		}
	}()
}

// SaveProfile validates metadata and rolls back changed secrets if metadata fails.
func (presenter *Presenter) SaveProfile(profile domain.Profile, password string) (domain.Profile, error) {
	presenter.targetsMu.Lock()
	defer presenter.targetsMu.Unlock()
	presenter.credentialsMu.Lock()
	defer presenter.credentialsMu.Unlock()
	if profile.Username == "" {
		profile.SecretRef = ""
	}
	if err := presenter.validateAccountTarget(profile); err != nil {
		return profile, err
	}
	if err := domain.ValidateProfile(profile); err != nil {
		return profile, err
	}
	if profile.ID == "" {
		id, err := domain.NewID()
		if err != nil {
			return profile, err
		}
		profile.ID = id
	}
	previous, err := presenter.Store.Profile(profile.ID)
	if err != nil && !errors.Is(err, store.ErrProfileNotFound) {
		return profile, err
	}
	if profile.Username != "" && password == "" && (profile.SecretRef == "" || previous.Username != profile.Username || previous.SessionOnly != profile.SessionOnly) {
		return profile, errors.New("Enter a password or access token for the selected account and storage policy")
	}
	undo := func() error { return nil }
	if password != "" && profile.Username != "" {
		profile.SecretRef = profile.ID
		undo, err = presenter.replaceSecret(previous, profile, password)
		if err != nil {
			return profile, err
		}
	}
	if err := presenter.Store.SaveProfile(profile); err != nil {
		return profile, errors.Join(err, undo())
	}
	if previous.SecretRef != "" && (previous.SecretRef != profile.SecretRef || previous.SessionOnly != profile.SessionOnly) {
		if err := presenter.Vault.Delete(previous.SecretRef, previous.SessionOnly); err != nil {
			return profile, err
		}
	}
	return profile, nil
}

// replaceSecret returns a rollback capability without putting the old value in metadata.
func (presenter *Presenter) replaceSecret(previous, next domain.Profile, password string) (func() error, error) {
	sameLocation := previous.SecretRef == next.SecretRef && previous.SessionOnly == next.SessionOnly
	undo := func() error { return presenter.Vault.Delete(next.SecretRef, next.SessionOnly) }
	if sameLocation {
		original, err := presenter.Vault.Get(previous.SecretRef, previous.SessionOnly)
		if err != nil && !errors.Is(err, secret.ErrNotFound) {
			return nil, err
		}
		if err == nil {
			undo = func() error { return presenter.Vault.Set(previous.SecretRef, original, previous.SessionOnly) }
		}
	}
	if err := presenter.Vault.Set(next.SecretRef, password, next.SessionOnly); err != nil {
		return nil, err
	}
	return undo, nil
}

// Logout clears credentials and the profile reference, leaving anonymous access.
func (presenter *Presenter) Logout(profile domain.Profile) error {
	presenter.credentialsMu.Lock()
	defer presenter.credentialsMu.Unlock()
	if err := presenter.Vault.Delete(profile.SecretRef, profile.SessionOnly); err != nil {
		return err
	}
	profile.SecretRef, profile.Username = "", ""
	return presenter.Store.SaveProfile(profile)
}

// Target reads current tag state before presenting explicit overwrite consent.
func (presenter *Presenter) Target(ctx context.Context, profile domain.Profile, input string) (string, string, error) {
	target, err := adapter.Canonical(profile, input, true)
	if err != nil {
		return "", "", err
	}
	digest, err := presenter.Registry.Digest(ctx, profile, target)
	if registry.IsCode(err, "NOT_FOUND") {
		return target, "", nil
	}
	return target, digest, err
}

// Download validates the final path and immutable platform selection before enqueueing.
func (presenter *Presenter) Download(profile domain.Profile, source domain.Source, output string) (string, error) {
	if source.Digest == "" || source.Platform == "" {
		return "", errors.New("Select a platform explicitly")
	}
	path, err := archive.Destination(output)
	if err != nil {
		return "", err
	}
	if _, err := os.Lstat(path); err == nil {
		return "", errors.New("The file already exists; choose another filename")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return presenter.Tasks.Submit(domain.Spec{Operation: "download", ProfileID: profile.ID, RegistryHost: profile.Host, Source: source, Output: path})
}

// SubmitRemote validates canonical inputs previously checked by Target and confirmed.
func (presenter *Presenter) SubmitRemote(spec domain.Spec) (string, error) {
	if spec.Operation != "copy" && spec.Operation != "import" && spec.Operation != "push" {
		return "", errors.New("Invalid operation type")
	}
	profile, err := presenter.Store.Profile(spec.ProfileID)
	if err != nil {
		return "", err
	}
	if spec.RegistryHost != profile.Host {
		return "", errors.New("The connection address changed; check the target again")
	}
	target, err := adapter.Canonical(profile, spec.Target, true)
	if err != nil {
		return "", err
	}
	spec.Target = target
	if spec.Operation == "copy" {
		if err := presenter.validateCopy(spec, profile); err != nil {
			return "", err
		}
	}
	if spec.Operation == "copy" && (spec.Source.Digest == "" || spec.Source.Platform == "") {
		return "", errors.New("Select a platform first")
	}
	if spec.Operation == "copy" && strings.Split(spec.Source.Reference, "@")[0] == target {
		return "", errors.New("The new tag must differ from the source tag")
	}
	if spec.Operation == "import" && spec.Archive.SHA256 == "" {
		return "", errors.New("Inspect the archive and select an entry first")
	}
	if spec.Operation == "push" {
		if err := domain.ValidateLocalSource(spec.Local); err != nil {
			return "", err
		}
	}
	return presenter.Tasks.Submit(spec)
}

// Close cancels background reads and waits before credential/database teardown.
func (presenter *Presenter) Close() {
	presenter.cancel()
	presenter.workers.Wait()
	presenter.Vault.Clear()
}

// WaitReads waits for outstanding read callbacks, for the headless renderer whose dispatcher runs inline.
// Callers must not use it on a native UI thread while dispatch is pending.
func (presenter *Presenter) WaitReads() { presenter.reads.Wait() }
