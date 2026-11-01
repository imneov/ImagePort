// Package task owns durable scheduling, attempts, cancellation and target locking.
package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/imneov/ImagePort/internal/domain"
)

// Repository persists detached snapshots and their key events.
type Repository interface {
	Tasks() ([]domain.Task, error)
	SaveTask(domain.Task) error
}

// Checkpoint durably records actual observable stages before execution continues.
type Checkpoint func(stage, expected, result string) error

// Runner executes operations and reconciles ambiguous outcomes without rewriting.
type Runner interface {
	Run(context.Context, domain.Task, Checkpoint) (string, error)
	Reconcile(context.Context, domain.Task) (bool, error)
}

// Unknown marks an outcome that must be reconciled before retrying.
type Unknown struct{ Reason string }

// Error reports the user-safe reason for an indeterminate result.
func (unknown *Unknown) Error() string { return unknown.Reason }

// Manager serializes target writes while limiting global transfer concurrency.
type Manager struct {
	mu             sync.Mutex
	repository     Repository
	runner         Runner
	logger         *slog.Logger
	tasks          map[string]domain.Task
	running        map[string]context.CancelFunc
	locks          map[string]bool
	wake           chan struct{}
	done           chan struct{}
	ctx            context.Context
	cancel         context.CancelFunc
	workers        sync.WaitGroup
	limit          int
	paused, closed bool
	fault          error
}

// New recovers active attempts as INTERRUPTED and holds any restored queue.
func New(repository Repository, runner Runner, concurrency int, logger *slog.Logger) (*Manager, error) {
	if concurrency < 1 || concurrency > 8 {
		return nil, errors.New("Concurrency must be between 1 and 8")
	}
	history, err := repository.Tasks()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	manager := &Manager{repository: repository, runner: runner, logger: logger, limit: concurrency,
		tasks: make(map[string]domain.Task), running: make(map[string]context.CancelFunc),
		locks: make(map[string]bool), wake: make(chan struct{}, 1), done: make(chan struct{}), ctx: ctx, cancel: cancel}
	for _, item := range history {
		if len(item.Attempts) == 0 {
			cancel()
			return nil, errors.New("Task history is missing an attempt record; preserve the database for recovery")
		}
		if item.State.Active() {
			item.State = domain.Interrupted
			attempt := &item.Attempts[len(item.Attempts)-1]
			attempt.State, attempt.Error = domain.Interrupted, "The previous attempt did not exit normally; verify the result first"
			item.UpdatedAt = time.Now().UTC()
			if err := repository.SaveTask(item); err != nil {
				cancel()
				return nil, err
			}
		}
		manager.paused = manager.paused || item.State == domain.Queued
		manager.tasks[item.ID] = item
	}
	go manager.schedule()
	return manager, nil
}

// Submit persists a user-confirmed operation before notifying the scheduler.
func (manager *Manager) Submit(spec domain.Spec) (string, error) {
	id, err := domain.NewID()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	item := domain.Task{ID: id, Spec: spec, State: domain.Queued, CreatedAt: now, UpdatedAt: now,
		Attempts: []domain.Attempt{{Number: 1, State: domain.Queued, Stage: "preflight"}}}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.closed || manager.fault != nil {
		return "", errors.New("The task queue is closed or the database is unavailable")
	}
	if err := manager.repository.SaveTask(item); err != nil {
		return "", err
	}
	manager.tasks[id] = clone(item)
	manager.signal()
	return id, nil
}

// signal coalesces scheduler wakeups; durable task state is the source of truth.
func (manager *Manager) signal() {
	select {
	case manager.wake <- struct{}{}:
	default:
	}
}

// targetKey uses the presenter's canonicalized output/ref rather than display text.
func targetKey(spec domain.Spec) string {
	if spec.DestinationEngine.Identity != "" {
		return "engine:" + spec.DestinationEngine.Identity + ":" + spec.Target
	}
	if spec.Operation == "delete-local" {
		return "engine:" + spec.Local.Engine.Identity + ":" + spec.Target
	}
	if spec.Operation == "delete-file" {
		return "file:" + strings.ToLower(spec.Archive.Path)
	}
	if spec.Output != "" {
		return "file:" + strings.ToLower(spec.Output)
	}
	return "registry:" + spec.Target
}

// schedule starts eligible tasks, skipping locked targets to avoid head-of-line stalls.
func (manager *Manager) schedule() {
	defer close(manager.done)
	for {
		select {
		case <-manager.ctx.Done():
			return
		case <-manager.wake:
			manager.mu.Lock()
			manager.dispatch()
			manager.mu.Unlock()
		}
	}
}

// dispatch is called with mu held and persists RUNNING before launching a worker.
func (manager *Manager) dispatch() {
	if manager.paused || manager.closed || manager.fault != nil {
		return
	}
	for _, item := range manager.sorted() {
		key := targetKey(item.Spec)
		if len(manager.running) >= manager.limit {
			return
		}
		if item.State != domain.Queued || manager.locks[key] {
			continue
		}
		item.State = domain.Running
		attempt := &item.Attempts[len(item.Attempts)-1]
		attempt.State, attempt.StartedAt = domain.Running, time.Now().UTC()
		if err := manager.save(item); err != nil {
			return
		}
		ctx, cancel := context.WithCancel(manager.ctx)
		manager.running[item.ID], manager.locks[key] = cancel, true
		manager.workers.Add(1)
		go manager.execute(ctx, clone(item))
	}
}

// execute retains locks until the runner exits and the terminal state is persisted.
func (manager *Manager) execute(ctx context.Context, item domain.Task) {
	defer manager.workers.Done()
	result, runErr := manager.runner.Run(ctx, item, func(stage, expected, result string) error {
		return manager.checkpoint(item.ID, stage, expected, result)
	})
	manager.mu.Lock()
	defer manager.mu.Unlock()
	current := manager.tasks[item.ID]
	state := domain.Succeeded
	var unknown *Unknown
	if errors.As(runErr, &unknown) {
		state = domain.Interrupted
	} else if errors.Is(runErr, context.Canceled) {
		state = domain.Cancelled
	} else if runErr != nil {
		state = domain.Failed
	}
	current.State = state
	attempt := &current.Attempts[len(current.Attempts)-1]
	attempt.State, attempt.Result, attempt.EndedAt = state, result, time.Now().UTC()
	if runErr != nil {
		attempt.Error = runErr.Error()
	}
	if err := manager.save(current); err != nil {
		manager.logger.Error("task persistence failed", "task", item.ID)
	}
	manager.running[item.ID]()
	delete(manager.running, item.ID)
	delete(manager.locks, targetKey(item.Spec))
	manager.signal()
}

// save updates durable state; failures stop scheduling and remain visible in memory.
func (manager *Manager) save(item domain.Task) error {
	item.UpdatedAt = time.Now().UTC()
	if err := manager.repository.SaveTask(item); err != nil {
		manager.fault = errors.New("Database write failed; the queue has stopped. Keep the files and verify tasks after restarting")
		item.State = domain.Interrupted
		attempt := &item.Attempts[len(item.Attempts)-1]
		attempt.State, attempt.Error = domain.Interrupted, manager.fault.Error()
		manager.tasks[item.ID] = clone(item)
		return manager.fault
	}
	manager.tasks[item.ID] = clone(item)
	manager.logger.Info("task state", "task", item.ID, "attempt", item.Current().Number, "state", item.State, "stage", item.Current().Stage)
	return nil
}

// checkpoint persists a stage and expected digest; it never reports guessed progress.
func (manager *Manager) checkpoint(id, stage, expected, result string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	item := manager.tasks[id]
	attempt := &item.Attempts[len(item.Attempts)-1]
	attempt.Stage = stage
	if expected != "" {
		attempt.ExpectedDigest = expected
	}
	if result != "" {
		attempt.Result = result
	}
	return manager.save(item)
}

// clone prevents callers from mutating manager-owned slice storage.
func clone(item domain.Task) domain.Task {
	item.Attempts = append([]domain.Attempt(nil), item.Attempts...)
	item.Spec.Archive.Entries = append([]domain.ArchiveEntry(nil), item.Spec.Archive.Entries...)
	for index := range item.Spec.Archive.Entries {
		entry := &item.Spec.Archive.Entries[index]
		entry.Platforms = append([]string(nil), entry.Platforms...)
	}
	item.Spec.Entry.Platforms = append([]string(nil), item.Spec.Entry.Platforms...)
	return item
}

// sorted returns oldest-first internal records; caller must hold mu.
func (manager *Manager) sorted() []domain.Task {
	items := make([]domain.Task, 0, len(manager.tasks))
	for _, item := range manager.tasks {
		items = append(items, item)
	}
	sort.Slice(items, func(left, right int) bool { return items[left].CreatedAt.Before(items[right].CreatedAt) })
	return items
}

// List returns immutable task snapshots and any queue-wide persistence failure.
func (manager *Manager) List() ([]domain.Task, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	items := manager.sorted()
	for index := range items {
		items[index] = clone(items[index])
	}
	return items, manager.fault
}

// Get returns an immutable snapshot by id.
func (manager *Manager) Get(id string) (domain.Task, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	item, found := manager.tasks[id]
	if !found {
		return domain.Task{}, fmt.Errorf("Task not found")
	}
	return clone(item), nil
}
