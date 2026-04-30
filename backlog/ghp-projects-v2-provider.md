# GitHub Projects v2 provider (phase 2)

## Properties
Type
:   Feature
Area
:   GitHub Projects v2
State
:   Proposed
Priority
:   2
Phase
:   5

## Summary
Add a provider that targets a GitHub Projects (v2) board via the GraphQL
API. Items in a Project may be either tracked Issues/PRs or
"draft" items that exist only on the project.

## Scope
- Auth via PAT with `project` scope (or fine-grained equivalent).
- Configurable per project: `owner`, `projectNumber`.
- Read items including their custom field values.
- Write: add existing issue to project, create draft items, update
  custom field values.

## Acceptance criteria
- Distinguishes draft items from linked issues; refuses to delete linked
  issues (only removes from the project).
- Provider contract tests pass with GraphQL fixtures.
