# GitHub labels and milestones

## Properties
Type
:   Feature
Area
:   GitHub Issues
State
:   Proposed
Priority
:   2
Phase
:   2

## Summary
Manage label and milestone lifecycle so push operations don't fail on
references to remote objects that don't exist yet.

## Acceptance criteria
- On push, missing labels can be auto-created when
  `--create-missing-labels` is set; otherwise the sync fails for that
  item with a clear message.
- Milestone create/update requires explicit opt-in
  (`--create-missing-milestones`).
- `mbs gh labels list/create/delete` helper commands.
- Color of auto-created labels configurable; defaults to deterministic
  hash of label name.
