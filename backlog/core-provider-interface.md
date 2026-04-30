# Provider interface and registry

## Properties
Type
:   Feature
Area
:   Core
State
:   Proposed
Priority
:   1
Phase
:   1

## Summary
Define the `Provider` Go interface that every backend (Azure DevOps,
GitHub Issues, GitHub Projects v2) must implement, plus a registry for
constructing providers from config.

## Interface (sketch)
```go
type Provider interface {
    Name() string
    Kind() string
    List(ctx, since *time.Time) iter.Seq2[RemoteItem, error]
    Get(ctx, remoteID string) (RemoteItem, error)
    Create(ctx, item Item) (RemoteItem, error)
    Update(ctx, remoteID string, patch ItemPatch, baseRev string) (RemoteItem, error)
    Delete(ctx, remoteID string) error      // may be no-op / state change
    Capabilities() Capabilities             // labels, milestones, parents, comments...
}
```

## Acceptance criteria
- Capability flags let the sync engine skip unsupported fields gracefully.
- Provider construction is data-driven from config + auth.
- Round-trip mapping helpers (`toRemote`, `fromRemote`) live in each
  provider package; core depends only on the interface.
- Mock provider used by sync-engine tests.
