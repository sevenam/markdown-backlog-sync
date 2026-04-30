# Distribution: Homebrew, Scoop, winget

## Properties
Type
:   Feature
Area
:   Distribution
State
:   Proposed
Priority
:   3
Phase
:   4

## Summary
Publish package manifests so users can install with one command on every
major platform.

## Acceptance criteria
- `goreleaser` updates a Homebrew tap (`sevenam/homebrew-tap`).
- `goreleaser` publishes a Scoop bucket manifest.
- A `winget` manifest PR is opened on each release (manual merge ok
  initially).
- README install instructions show all four paths (binary, brew, scoop,
  winget) plus `go install`.
