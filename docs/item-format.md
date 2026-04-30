# Item file format

Backlog items are **pure Markdown**, one file per item, stored under the
workspace items directory (default `./backlog/`). There is no YAML or
TOML front-matter.

## Required structure

1. The first non-empty line MUST be an H1 — that is the item title.
2. The first H2 MUST be `## Properties` and use a Markdown definition
   list to encode item metadata.
3. Any further H2 sections (`## Summary`, `## Acceptance criteria`,
   `## Notes`, etc.) are preserved verbatim during round-trip.

```markdown
# Add OAuth login

## Properties
Type
:   Feature
State
:   In Progress
Labels
:   auth
:   priority/high

## Summary
Some prose…

## Acceptance criteria
- one
- two
```

## Property values

- A property term is a non-empty line.
- The next line(s) start with `:   ` (colon + three spaces). Multiple
  consecutive `:   ` lines combine into a multi-value (used for things
  like `Labels`).
- A blank line ends the current property.

## Sync state

A hidden `.sync/` directory at the workspace root keeps per-item state
(remote ids, etags, last-synced content hash). It can be safely
gitignored, or committed to enable team-wide conflict detection. `mbs
init` adds it to `.gitignore` by default in git repositories.

## Canonicalization

Run `mbs fmt` to rewrite item files into canonical form (consistent
whitespace, deterministic property serialization). `mbs fmt --check`
exits non-zero if any file would be rewritten — useful in CI.
