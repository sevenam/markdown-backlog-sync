# GitHub Issues comments sync

## Properties
Type
:   Feature
Area
:   GitHub Issues
State
:   Proposed
Priority
:   3
Phase
:   2

## Summary
Optionally sync issue comments under a `## Comments` section in the
Markdown file.

## Behavior
- Disabled by default; enable per-provider via
  `[provider.options] sync_comments = true`.
- Each comment is rendered as `### <author> — <ISO timestamp>` followed
  by the comment body.
- Local edits to existing comment blocks update the corresponding remote
  comment; new blocks at the bottom create new comments.
- Comment IDs tracked in `.sync/items/<localId>.json`.

## Acceptance criteria
- Round-trip preserves author + timestamp metadata.
- Deletions on either side are mirrored only with explicit
  `--allow-comment-delete` (safety).
