# Documentation site

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
Publish a docs site (mkdocs-material or Hugo + GitHub Pages) covering
install, quickstart, configuration, the Markdown file format, the auth
model, and per-provider mapping reference.

## Acceptance criteria
- `docs/` source in-repo; built and deployed via GitHub Actions on
  pushes to `main`.
- Auto-generated CLI reference from cobra command tree.
- Versioned docs aligned with releases.
