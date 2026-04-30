# GitHub sub-issues, task lists, and Projects v2 parent field

## Properties
Type
:   Feature
Area
:   GitHub
State
:   Proposed
Priority
:   2
Phase
:   2
Parent
:   core-hierarchy-model

## Summary
GitHub's hierarchy story is fragmented across three mechanisms. The
provider must map the single Markdown `Parent` property to whichever
mechanism the target repo/project actually uses.

## The three mechanisms
1. **Native sub-issues** (GA 2025) — REST/GraphQL endpoints
   `POST /repos/{o}/{r}/issues/{n}/sub_issues` and the
   `subIssues`/`parent` GraphQL fields. Up to 100 sub-issues per parent,
   8 levels deep. Preferred when available.
2. **Task lists in issue body** — `- [ ] owner/repo#123` checkboxes.
   Render as a hierarchy in the GitHub UI but are body-edits, not link
   operations. Used as a fallback when sub-issues are disabled or for
   cross-repo references that sub-issues don't allow.
3. **Projects v2 parent field** — the built-in `Parent issue` field on
   a Projects v2 board. Owned by `ghp-projects-v2-fields-mapping`; this
   item only ensures the `Parent` Markdown property is the single
   source of truth that flows into all three.

## Behaviour
- Provider config picks the strategy:
  `[provider.github.hierarchy] mode = "sub-issues" | "task-list" | "auto"`
  with `auto` preferring sub-issues and falling back to task lists.
- Cross-repo `Parent` references force `task-list` mode for that edge
  (sub-issues are intra-repo only) and surface a warning.
- Reparenting an issue via the local `Parent` property:
  - sub-issues mode: `DELETE` old sub-issue link, `POST` new one.
  - task-list mode: rewrite the parent's task list section in place
    (managed under a stable HTML comment marker so user edits outside
    the marker are preserved).

## Acceptance criteria
- A locally created `Parent: epic-billing` round-trips to a real
  sub-issue link on push and reads back as the same `Parent` value on
  pull.
- Task-list fallback uses a sentinel block
  (`<!-- mbs:children -->` ... `<!-- /mbs:children -->`) so unrelated
  body content survives.
- Capability flags on the provider truthfully report
  `SupportsSubIssues` / `SupportsTaskListParents` so the sync engine
  can short-circuit unsupported operations cleanly.
- Depth/width limits (8 levels, 100 children) are enforced client-side
  with a clear error before hitting the API.
