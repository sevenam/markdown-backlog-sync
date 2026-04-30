# Markdown item file format (pure Markdown)

## Properties
Type
:   Feature
Area
:   Core
State
:   Proposed
Priority
:   1
Phase
:   1

## Summary
Specify and implement a lossless parser/serializer for backlog item files.
Files are pure Markdown — no YAML/TOML front-matter. Metadata lives in a
canonical `## Properties` section using a Markdown definition list.

## File contract
- First non-empty line MUST be an H1 with the item title.
- A single `## Properties` section MUST appear before any other H2.
- Properties are a definition list (`Term\n:   Value`).
- Optional H2 sections in this order: `## Summary`, `## Acceptance
  criteria`, `## Notes`, `## Comments`, `## Links`, `## Attachments`,
  `## History` (sync-managed).
- Unknown sections are preserved verbatim during round-trip.

## Required properties
`Type`, `State`, `Provider`, `RemoteId` (empty for unsynced items).

## Optional well-known properties
`Priority`, `Assignee`, `Labels`, `Milestone`, `Iteration`, `AreaPath`,
`Parent`, `Estimate`, `CreatedAt`, `UpdatedAt`, `Url`.

## Acceptance criteria
- Parser → in-memory model → serializer is byte-stable for files that
  already follow the conventions (golden-file tests).
- Unknown properties and sections survive round-trip.
- Clear, line-numbered error messages for malformed files.
- A `mbs fmt` command rewrites files into canonical form.
