# Distribution: GitHub Releases

## Properties
Type
:   Feature
Area
:   Distribution
State
:   Proposed
Priority
:   2
Phase
:   4

## Summary
Automate signed releases using `goreleaser` triggered by version tags.

## Acceptance criteria
- Tagging `vX.Y.Z` builds and publishes archives, checksums, SBOM, and
  release notes (generated from conventional commits).
- Artifacts cosign-signed; checksums file signed too.
- `mbs version` reports build metadata embedded at release time.
