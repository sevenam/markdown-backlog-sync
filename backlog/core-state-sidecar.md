# Sync state sidecar (.sync/)

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
Implement a per-workspace sidecar store under `.sync/` that records, for
each item, the data needed for safe bidirectional sync.

## Stored per item
- Local file path and stable local id (UUID minted on first sync).
- Provider name + remote id + remote url.
- Last-synced content hash (sha256 of canonical Markdown).
- Last-synced remote revision/etag/`rev` (Azure DevOps `rev`, GitHub
  `updated_at`+`node_id`).
- Last-synced timestamp.

## Layout
```
.sync/
  index.json                # provider -> { remoteId -> localId } map
  items/<localId>.json      # per-item state
  providers/<name>.json     # provider-level cursors (delta tokens)
```

## Acceptance criteria
- Atomic writes (write-temp + rename) to survive crashes mid-sync.
- Schema versioned with forward-compat migration hooks.
- `mbs status` reports drift between disk, sidecar, and remote.
- Documented as safe to commit OR gitignore; default `.gitignore` entry
  added by `mbs init` (user-confirmable).
