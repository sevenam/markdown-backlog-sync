# Azure DevOps iterations and area paths

## Properties
Type
:   Feature
Area
:   Azure DevOps
State
:   Proposed
Priority
:   2
Phase
:   3

## Summary
Resolve and validate `IterationPath` / `AreaPath` values, since these are
hierarchical strings that must exist in the project's classification
nodes.

## Acceptance criteria
- On push, validate that the path exists; if not, emit a clear error or
  optionally auto-create when `--create-classification-nodes` is set.
- On pull, normalize to the canonical backslash form.
- `mbs azdo iterations list --provider <name>` helper command.
