# Azure DevOps Boards provider implementation

## Properties
Type
:   Feature
Area
:   Azure DevOps
State
:   Proposed
Priority
:   1
Phase
:   3

## Summary
Implement the `Provider` interface against Azure DevOps Boards REST API
(`7.1`). Use the official Go SDK
(`github.com/microsoft/azure-devops-go-api/azuredevops`) where it covers
the surface; fall back to direct REST calls otherwise.

## Scope
- Authenticated via PAT (Basic auth, empty username, PAT as password).
- Org/project/team scoped per provider config entry.
- Read: list work items via WIQL with `System.ChangedDate >= cursor`.
- Write: create + update work items via JSON Patch documents; use `rev`
  for optimistic concurrency.
- Soft-delete: set state to `Removed`/`Closed` (configurable); no hard
  delete by default.

## Acceptance criteria
- Provider contract tests pass with recorded fixtures.
- Honors HTTP 429 + `Retry-After`.
- Surfaces 401/403 as exit code 3 with a clear remediation message.
