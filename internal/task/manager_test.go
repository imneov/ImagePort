package task_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/imneov/ImagePort/internal/domain"
	"github.com/imneov/ImagePort/internal/store"
	"github.com/imneov/ImagePort/internal/task"
)

type controlledRunner struct {
	started chan string
	release chan struct{}
}

// Run blocks until cancellation or release, letting tests observe target scheduling.
func (runner controlledRunner) Run(ctx context.Context, item domain.Task, checkpoint task.Checkpoint) (string, error) {
	if err := checkpoint("transfer", item.Spec.Source.Digest, ""); err != nil {
		return "", err
	}
	runner.started <- item.Spec.Target
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-runner.release:
		return "verified", nil
	}
}

// Reconcile simulates an absent expected result without writing anything.
func (runner controlledRunner) Reconcile(context.Context, domain.Task) (bool, error) {
	return false, nil
}

// database creates a real SQLite repository and registers reliable teardown.
func database(test *testing.T) *store.Store {
	test.Helper()
	repository, err := store.Open(filepath.Join(test.TempDir(), "tasks.db"))
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() {
		if err := repository.Close(); err != nil {
			test.Error(err)
		}
	})
	return repository
}

// manager creates a production scheduler with deterministic test execution boundaries.
func manager(test *testing.T, repository *store.Store, runner controlledRunner) *task.Manager {
	test.Helper()
	instance, err := task.New(repository, runner, 2, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() {
		if err := instance.Close(context.Background()); err != nil {
			test.Error(err)
		}
	})
	return instance
}

// started waits for an actual transfer start rather than relying on a scheduling sleep.
func started(test *testing.T, runner controlledRunner) string {
	test.Helper()
	select {
	case target := <-runner.started:
		return target
	case <-time.After(3 * time.Second):
		test.Fatal("scheduler did not start")
		return ""
	}
}

// terminal waits for the persisted terminal state while bounding failures.
func terminal(test *testing.T, manager *task.Manager, id string) domain.Task {
	test.Helper()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(3 * time.Second)
	for {
		select {
		case <-timeout:
			test.Fatal("task did not finish")
		case <-ticker.C:
			item, err := manager.Get(id)
			if err != nil {
				test.Fatal(err)
			}
			if item.State != domain.Queued && !item.State.Active() {
				return item
			}
		}
	}
}

// TestTargetSerialization runs unrelated targets concurrently while serializing matches.
func TestTargetSerialization(test *testing.T) {
	runner := controlledRunner{make(chan string, 8), make(chan struct{})}
	instance := manager(test, database(test), runner)
	for _, target := range []string{"registry/repo:one", "registry/repo:one", "registry/repo:two"} {
		if _, err := instance.Submit(domain.Spec{Target: target}); err != nil {
			test.Fatal(err)
		}
	}
	first, second := started(test, runner), started(test, runner)
	if first == second {
		test.Fatal("same target ran concurrently")
	}
	close(runner.release)
	if next := started(test, runner); next != "registry/repo:one" {
		test.Fatal("locked target not resumed")
	}
}

// TestCancelAndRetry records a new attempt and retains the originally pinned source.
func TestCancelAndRetry(test *testing.T) {
	runner := controlledRunner{make(chan string, 8), make(chan struct{})}
	instance := manager(test, database(test), runner)
	id, err := instance.Submit(domain.Spec{Target: "registry/repo:tag", Source: domain.Source{Digest: "sha256:pinned"}})
	if err != nil {
		test.Fatal(err)
	}
	started(test, runner)
	if err := instance.Cancel(id); err != nil {
		test.Fatal(err)
	}
	if item := terminal(test, instance, id); item.State != domain.Cancelled {
		test.Fatal("cancellation not finalized")
	}
	if err := instance.Retry(id, ""); err != nil {
		test.Fatal(err)
	}
	started(test, runner)
	close(runner.release)
	item := terminal(test, instance, id)
	if len(item.Attempts) != 2 || item.Current().Number != 2 || item.Spec.Source.Digest != "sha256:pinned" {
		test.Fatal("attempt/source identity changed")
	}
}

// TestRestartInterruptsAndHoldsQueue proves startup never blindly repeats old writes.
func TestRestartInterruptsAndHoldsQueue(test *testing.T) {
	repository := database(test)
	for _, state := range []domain.State{domain.Running, domain.CancelRequested, domain.Queued} {
		item := domain.Task{ID: string(state), State: state, CreatedAt: time.Now(), Attempts: []domain.Attempt{{Number: 1, State: state}}}
		if err := repository.SaveTask(item); err != nil {
			test.Fatal(err)
		}
	}
	runner := controlledRunner{make(chan string, 8), make(chan struct{})}
	instance := manager(test, repository, runner)
	if !instance.Paused() {
		test.Fatal("restored queue resumed automatically")
	}
	for _, id := range []string{string(domain.Running), string(domain.CancelRequested)} {
		item, err := instance.Get(id)
		if err != nil {
			test.Fatal(err)
		}
		if item.State != domain.Interrupted {
			test.Fatal("active attempt not interrupted")
		}
		if err := instance.Retry(id, ""); err == nil {
			test.Fatal("unknown outcome retried before verification")
		}
	}
	instance.Resume()
	started(test, runner)
	close(runner.release)
}

// TestDetachedSnapshots prevents mutation of scheduler state through returned slices.
func TestDetachedSnapshots(test *testing.T) {
	runner := controlledRunner{make(chan string, 8), make(chan struct{})}
	instance := manager(test, database(test), runner)
	id, err := instance.Submit(domain.Spec{Target: "registry/repo:tag"})
	if err != nil {
		test.Fatal(err)
	}
	started(test, runner)
	item, err := instance.Get(id)
	if err != nil {
		test.Fatal(err)
	}
	item.Attempts[0].Error = "injected"
	actual, err := instance.Get(id)
	if err != nil {
		test.Fatal(err)
	}
	if actual.Current().Error != "" {
		test.Fatal("snapshot mutated scheduler")
	}
	if err := instance.Cancel(id); err != nil && !errors.Is(err, context.Canceled) {
		test.Fatal(err)
	}
}
