package task

import (
	"context"
	"errors"
	"time"

	"github.com/imneov/ImagePort/internal/domain"
)

// Cancel persists a request before propagating it; queued tasks stop immediately.
func (manager *Manager) Cancel(id string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	item, found := manager.tasks[id]
	if !found {
		return errors.New("Task not found")
	}
	if item.State != domain.Queued && !item.State.Active() {
		return errors.New("This task has already finished")
	}
	if item.State == domain.Queued {
		item.State = domain.Cancelled
	} else {
		item.State = domain.CancelRequested
	}
	attempt := &item.Attempts[len(item.Attempts)-1]
	attempt.State = item.State
	if item.State == domain.Cancelled {
		attempt.EndedAt = time.Now().UTC()
	}
	if err := manager.save(item); err != nil {
		return err
	}
	if cancel, running := manager.running[id]; running {
		cancel()
	}
	return nil
}

// Retry adds an attempt with the same immutable source after renewed target consent.
func (manager *Manager) Retry(id, previousDigest string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	item, found := manager.tasks[id]
	if !found {
		return errors.New("Task not found")
	}
	if manager.closed || manager.fault != nil {
		return errors.New("The task queue is unavailable")
	}
	if item.State != domain.Failed && item.State != domain.Cancelled {
		return errors.New("Only failed or cancelled tasks can be retried; verify unknown outcomes first")
	}
	item.Spec.PreviousDigest = previousDigest
	item.State = domain.Queued
	item.Attempts = append(item.Attempts, domain.Attempt{Number: len(item.Attempts) + 1, State: domain.Queued, Stage: "preflight"})
	if err := manager.save(item); err != nil {
		return err
	}
	manager.signal()
	return nil
}

// Reconcile only reads results; no import, export or remote write is performed.
func (manager *Manager) Reconcile(ctx context.Context, id string) error {
	item, err := manager.Get(id)
	if err != nil {
		return err
	}
	if item.State != domain.Interrupted {
		return errors.New("This task does not need interrupted-result verification")
	}
	succeeded, err := manager.runner.Reconcile(ctx, item)
	if err != nil {
		return err
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	current := manager.tasks[id]
	if current.State != domain.Interrupted {
		return errors.New("The task state changed; refresh and try again")
	}
	current.State = domain.Failed
	attempt := &current.Attempts[len(current.Attempts)-1]
	attempt.Error = "Verification did not find the expected result; confirm the target again before a manual retry"
	if succeeded {
		current.State, attempt.Error = domain.Succeeded, ""
	}
	attempt.State, attempt.EndedAt = current.State, time.Now().UTC()
	return manager.save(current)
}

// Resume releases a restored queue only after an explicit user action.
func (manager *Manager) Resume() {
	manager.mu.Lock()
	manager.paused = false
	manager.signal()
	manager.mu.Unlock()
}

// Paused reports whether restored queued tasks are waiting for user resumption.
func (manager *Manager) Paused() bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.paused
}

// SetConcurrency updates the limit for future scheduling without interrupting workers.
func (manager *Manager) SetConcurrency(limit int) error {
	if limit < 1 || limit > 8 {
		return errors.New("Concurrency must be between 1 and 8")
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.limit = limit
	manager.signal()
	return nil
}

// Close stops scheduling and requests cancellation; deadline failures retain DB access.
func (manager *Manager) Close(ctx context.Context) error {
	manager.mu.Lock()
	manager.closed = true
	manager.cancel()
	manager.mu.Unlock()
	select {
	case <-manager.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	finished := make(chan struct{})
	go func() { manager.workers.Wait(); close(finished) }()
	select {
	case <-finished:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
