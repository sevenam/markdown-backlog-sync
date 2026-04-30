# Conflict detection and resolution

## Properties
Type
:   Feature
Area
:   Core
State
:   Proposed
Priority
:   2
Phase
:   1

## Summary
Field-level 3-way merge between `local`, `remote`, and `base` (last
synced) representations of an item. Provide both automatic and
interactive resolution.

## Behavior
- Non-overlapping field changes auto-merge.
- Overlapping changes produce a conflict the user resolves via:
  - `--strategy local` / `--strategy remote` / `--strategy newest`
  - Interactive TUI picker (default when stdout is a TTY)
  - Markdown conflict markers inserted into the file (git-style) for
    body content; `mbs resolve <file>` finalizes.
- Unresolved items are skipped and reported with non-zero exit code 4.

## Acceptance criteria
- Deterministic merge given identical inputs.
- Property-level diffs in `mbs status --conflicts`.
- Clear documentation with worked examples.
- Test matrix covering every (localChanged × remoteChanged × fieldKind)
  combination.
