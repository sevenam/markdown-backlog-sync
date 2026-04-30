# Testing strategy

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
Define test layers and supporting infrastructure.

## Layers
- **Unit:** parser/serializer (golden files), state store, merge engine.
- **Provider contract tests:** a shared suite each provider implementation
  must pass against a recorded HTTP fixture set (`go-vcr`).
- **Integration:** opt-in tests against real Azure DevOps + GitHub using
  scoped PATs from CI secrets, isolated to a sandbox project/repo.
- **End-to-end:** scripted scenarios in `testdata/e2e/` exercising
  `mbs init` → edit → `mbs sync` → assert remote and local state.

## Acceptance criteria
- `go test ./...` runs unit + contract tests offline in <30s.
- Integration tests gated behind `-tags=integration` and CI secrets.
- Coverage threshold enforced for `internal/markdown` and `internal/sync`.
