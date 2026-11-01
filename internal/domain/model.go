// Package domain defines UI-independent image management models.
package domain

import "time"

// Profile contains registry metadata only; SecretRef points to a separate vault.
type Profile struct {
	ID, Name, Host, Username, SecretRef, CAPEM string
	TargetID                                   string
	SessionOnly                                bool
	// HTTP is restricted to loopback development registries; production uses TLS.
	HTTP bool
}

// Source is an explicitly selected, immutable image manifest.
type Source struct {
	Reference, IndexDigest, Digest, Platform string
	Size                                     int64
}

// ArchiveEntry describes one selectable root, preserving all its platforms.
type ArchiveEntry struct {
	Name, Digest string
	Platforms    []string
}

// Archive identifies a validated file and its selectable image roots.
type Archive struct {
	Path, Format, SHA256 string
	Size                 int64
	Entries              []ArchiveEntry
}

// Spec stores user-confirmed transfer inputs, never credentials.
type Spec struct {
	Operation, ProfileID, RegistryHost  string
	SourceProfileID, SourceRegistryHost string
	Source                              Source
	Target, Output                      string
	Archive                             Archive
	Entry                               ArchiveEntry
	Local                               LocalSource
	DestinationEngine                   Engine
	SourceTargetID, DestinationTargetID string
	// AccessPinned distinguishes new tasks whose account identities are immutable.
	AccessPinned                    bool
	ProfileUsername, SourceUsername string
	// PreviousDigest is empty only when the target was confirmed absent.
	PreviousDigest string
}

// State distinguishes a cancellation request from a confirmed stopped attempt.
type State string

const (
	Queued          State = "QUEUED"
	Running         State = "RUNNING"
	CancelRequested State = "CANCEL_REQUESTED"
	Succeeded       State = "SUCCEEDED"
	Failed          State = "FAILED"
	Cancelled       State = "CANCELLED"
	Interrupted     State = "INTERRUPTED"
)

// Active reports whether state represents work that has not exited.
func (state State) Active() bool {
	return state == Running || state == CancelRequested
}

// Attempt records one execution, including its pinned result and safe error.
type Attempt struct {
	Number                               int
	State                                State
	Stage, ExpectedDigest, Result, Error string
	StartedAt, EndedAt                   time.Time
}

// Task is a durable logical operation with an append-only attempt history.
type Task struct {
	ID                   string
	Spec                 Spec
	State                State
	CreatedAt, UpdatedAt time.Time
	Attempts             []Attempt
}

// Current returns the latest attempt value, or a zero value for an invalid task.
func (task Task) Current() Attempt {
	if len(task.Attempts) == 0 {
		return Attempt{}
	}
	return task.Attempts[len(task.Attempts)-1]
}

// Page retains the server cursor; its items must not be sliced a second time.
type Page struct {
	Items    []string
	Next     string
	Total    int
	HasTotal bool
}

// Engine identifies a Docker context and endpoint without persisting TLS keys.
type Engine struct {
	Context, Identity, Host string
	FromEnv                 bool
}

// LocalSource pins a daemon image by its content ID and engine identity.
type LocalSource struct {
	Engine  Engine
	ImageID string
}

// LocalImage describes actual daemon metadata; ID is distinct from manifest digests.
type LocalImage struct {
	ID       string
	Tags     []string
	Size     int64
	Created  time.Time
	Platform string
}

// LocalImages contains one engine's complete image listing for local filtering.
type LocalImages struct {
	Engine Engine
	Images []LocalImage
}
