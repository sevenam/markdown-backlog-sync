# Sync engine (pull / push / bidirectional)

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
Core engine that reconciles local Markdown files with one or more remote
providers. Implements pull, push, and full bidirectional sync as a single
algorithm parameterized by direction.

## Algorithm
For each known item (union of local files and remote list since last
cursor):
1. Load `local`, `remote`, and `base` (last-synced snapshot from `.sync/`).
2. Classify: unchanged / local-only / remote-only / both-changed.
3. Apply action: noop / push / pull / **3-way merge** (see conflict-
   resolution item).
4. Persist new sidecar state atomically.

## Acceptance criteria
- `mbs pull`, `mbs push`, `mbs sync` all share one engine.
- `--dry-run` prints the planned actions without mutating anything.
- `--provider <name>` and `--filter <expr>` scope a sync.
- Concurrent provider sync with bounded parallelism (default 4).
- Resumable: a crash mid-sync leaves a consistent sidecar.
- Rate-limit aware backoff for both providers.
