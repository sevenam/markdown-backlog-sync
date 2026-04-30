# Cross-platform build and packaging

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
Make the binary trivially installable on Windows, macOS, and Linux
(amd64 + arm64).

## Acceptance criteria
- `goreleaser` config produces archives + checksums + SBOM for
  windows/darwin/linux × amd64/arm64.
- `go install github.com/sevenam/markdown-backlog-sync/cmd/mbs@latest`
  works without cgo.
- Reproducible builds (stable `-trimpath` + pinned toolchain).
- CI matrix runs `go test ./...` and `go vet` on all three OSes.
