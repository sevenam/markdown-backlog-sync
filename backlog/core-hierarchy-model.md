# Hierarchy model (Epic → Feature → Story/Bug → Task)

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
Parent
:   core-markdown-format

## Summary
Define and enforce a cross-platform hierarchy model so a local Markdown
backlog can faithfully represent — and round-trip — the multi-level work
breakdown structures used by Azure DevOps Boards (Epic → Feature →
Story/Bug → Task) and GitHub (sub-issues, task lists, Projects v2 parent
field). Hierarchy is a first-class concern, not an afterthought of the
file format.

## Representation in Markdown
- `Parent` property in the item's `## Properties` block, with one of:
  - a local item reference: `Parent:   epic-billing` (resolves to the
    item whose filename stem or explicit `LocalId` matches), or
  - a fully-qualified remote reference:
    `Parent:   azdo:contoso/Phoenix#1234` /
    `Parent:   github:owner/repo#42`.
- Optional folder convention (purely a UX affordance, not authoritative):
  `backlog/<epic>/<feature>/<story>.md`. Folder placement only seeds the
  default `Parent` value when a new file is created via `mbs new`; the
  `Parent` property always wins on conflict.
- Children are *not* listed in the parent file. The relationship is
  stored once, on the child, to keep diffs local and avoid edit storms.

## Type ladder
- Per-provider config declares the allowed type ladder and which child
  types each parent may have. Defaults ship for AzDo Agile/Scrum/Basic
  and for GitHub (Epic/Feature via labels, Issue, Sub-issue, Task).
- `mbs lint` (and `mbs sync --check`) reject files whose `Type`/`Parent`
  pair violates the configured ladder, with a clear error.

## Validation & invariants
- Cycle detection: hierarchy must be a forest. `mbs lint` fails on
  cycles and reports the cycle path.
- Orphan handling: if a `Parent` reference does not resolve, the item is
  flagged as orphaned but still syncable; `mbs status` lists orphans.
- Reparenting is a single-property edit; the sync engine translates that
  into the appropriate add/remove link operations on the remote.

## CLI affordances
- `mbs tree [--root <id>]` prints the local hierarchy as an indented
  tree, with sync state markers.
- `mbs new <type> --parent <id>` creates a child file pre-populated with
  the right `Type` and `Parent` (and, if the folder convention is on,
  places it in the parent's folder).

## Acceptance criteria
- Parser exposes `Parent` as a structured reference (kind + value),
  not just a string.
- A reference resolver maps local ↔ remote IDs both ways, using
  `.sync/index.json`.
- Lint rules enforce: known type, allowed parent type, no cycles,
  resolvable parent (or explicit `--allow-orphans`).
- `mbs tree` round-trips with no provider calls (works fully offline).
- Reparenting an item produces exactly one link-update operation per
  affected remote on the next `mbs push`.

## Out of scope (this item)
- Multi-parent items (AzDo allows multiple `Parent` links in theory but
  the standard process templates forbid it; we mirror that constraint).
- Non-hierarchy link types (Related/Predecessor/Successor) — covered by
  `azdo-attachments-and-links` and a future `core-link-types` item.
