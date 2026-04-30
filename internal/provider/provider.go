// Package provider defines the interface every backlog backend must
// implement plus a registry so they can be constructed from config.
package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Capabilities describes which optional features a provider supports.
// The sync engine checks these flags before attempting unsupported ops.
type Capabilities struct {
	Labels      bool
	Milestones  bool
	Iterations  bool
	AreaPaths   bool
	Comments    bool
	Attachments bool
	ParentLinks bool
	HardDelete  bool
	DeltaSince  bool // true if List supports an incremental "since" cursor
}

// RemoteItem is a provider-side view of an item in a backend-neutral form.
type RemoteItem struct {
	ID         string // stable remote ID (e.g. issue number, work item id)
	URL        string // canonical web URL
	Title      string
	Body       string // markdown
	Type       string // e.g. "issue", "Bug", "User Story"
	State      string // provider-native state name
	Assignees  []string
	Labels     []string
	Milestone  string
	Iteration  string
	AreaPath   string
	Parent     string            // remote ID of parent item, if any
	Properties map[string]string // overflow / unmapped fields
	Rev        string            // opaque revision token (etag, rev number, updated_at)
	UpdatedAt  time.Time
}

// ItemPatch is a sparse update intended for Update calls. Nil fields mean
// "unchanged"; empty strings mean "clear".
//
// Note (Phase 2 follow-up): the Properties map cannot currently
// distinguish "clear" from "set to empty string" or carry typed values.
// Once a real provider needs that (notably Azure DevOps custom fields),
// expand this into a richer FieldPatch representation.
type ItemPatch struct {
	Title      *string
	Body       *string
	Type       *string
	State      *string
	Assignees  *[]string
	Labels     *[]string
	Milestone  *string
	Iteration  *string
	AreaPath   *string
	Parent     *string
	Properties map[string]string
}

// Provider is the interface every backend implements.
type Provider interface {
	Name() string
	Kind() string
	Capabilities() Capabilities

	// List enumerates remote items, optionally only those updated after
	// the given cursor. Implementations may ignore the cursor if
	// Capabilities().DeltaSince is false.
	List(ctx context.Context, since *time.Time) ([]RemoteItem, error)

	Get(ctx context.Context, remoteID string) (RemoteItem, error)
	Create(ctx context.Context, item RemoteItem) (RemoteItem, error)
	Update(ctx context.Context, remoteID string, patch ItemPatch, baseRev string) (RemoteItem, error)
	// Delete removes (or soft-removes) the remote item. baseRev is the
	// revision the caller observed when it decided to delete; providers
	// SHOULD reject the delete with ErrConflict if the remote has moved
	// on. Pass an empty baseRev to force-delete unconditionally.
	Delete(ctx context.Context, remoteID string, baseRev string) error
}

// Factory constructs a Provider from a config options map. credResolver is
// invoked to obtain the PAT or other secret for the named provider.
type Factory func(name string, options map[string]any, credResolver CredentialResolver) (Provider, error)

// CredentialResolver returns the secret value (typically a PAT) associated
// with the given provider name. Implementations should never log the value.
type CredentialResolver interface {
	Resolve(providerName string) (string, error)
}

// Registry maps provider type strings to their Factory.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{factories: map[string]Factory{}} }

// Register adds a factory for the given provider type. Panics on duplicate
// registration to surface programmer errors at startup.
func (r *Registry) Register(kind string, f Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.factories[kind]; ok {
		panic(fmt.Sprintf("provider type %q already registered", kind))
	}
	r.factories[kind] = f
}

// Build constructs a provider by type. Returns ErrUnknownType for unknown
// kinds.
func (r *Registry) Build(kind, name string, options map[string]any, creds CredentialResolver) (Provider, error) {
	r.mu.RLock()
	f, ok := r.factories[kind]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownType, kind)
	}
	return f(name, options, creds)
}

// Kinds returns the set of registered provider type strings.
func (r *Registry) Kinds() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.factories))
	for k := range r.factories {
		out = append(out, k)
	}
	return out
}

// ErrUnknownType is returned by Registry.Build for unregistered kinds.
var ErrUnknownType = errors.New("unknown provider type")

// Sentinel errors that providers SHOULD wrap when a real implementation
// can detect the corresponding condition. The sync engine inspects these
// to decide retry, conflict, and exit-code behavior.
var (
	// ErrItemNotFound indicates the remote item does not exist.
	ErrItemNotFound = errors.New("provider: item not found")
	// ErrConflict indicates an optimistic-concurrency / revision mismatch.
	ErrConflict = errors.New("provider: revision conflict")
	// ErrAuth indicates an authentication or authorization failure.
	ErrAuth = errors.New("provider: authentication failed")
	// ErrRateLimited indicates the remote rejected the call due to rate
	// limiting; callers should consult RetryAfter on a *RateLimitError.
	ErrRateLimited = errors.New("provider: rate limited")
	// ErrTransient is a generic retryable error (5xx, network blips).
	ErrTransient = errors.New("provider: transient error")
	// ErrValidation is a non-retryable user-input/data error.
	ErrValidation = errors.New("provider: validation error")
)

// RateLimitError optionally carries a duration to wait before retrying.
// Providers can return &RateLimitError{...} which still satisfies
// errors.Is(err, ErrRateLimited).
type RateLimitError struct {
	RetryAfter time.Duration
	Err        error
}

func (e *RateLimitError) Error() string {
	if e == nil || e.Err == nil {
		return ErrRateLimited.Error()
	}
	return e.Err.Error()
}

func (e *RateLimitError) Unwrap() error { return e.Err }
func (e *RateLimitError) Is(target error) bool {
	return target == ErrRateLimited
}
