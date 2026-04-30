# GitHub Issues field mapping

## Properties
Type
:   Feature
Area
:   GitHub Issues
State
:   Proposed
Priority
:   1
Phase
:   2

## Summary
Map Markdown item properties to/from GitHub Issue fields.

## Mapping (default)
| Markdown property        | GitHub Issue field           |
| ------------------------ | ---------------------------- |
| H1 title                 | `title`                      |
| Body (sections preserved)| `body` (Markdown, native)    |
| State                    | `state` (`open` / `closed`) + `state_reason` |
| Assignee / Assignees     | `assignees`                  |
| Labels                   | `labels`                     |
| Milestone                | `milestone.title`            |
| RemoteId                 | `number`                     |
| Url                      | `html_url`                   |
| Parent                   | Sub-issue / task-list ref (best-effort, capability-flagged) |

## Acceptance criteria
- Body is stored verbatim Markdown (no conversion needed).
- Closed-as-not-planned vs completed preserved via `state_reason`.
- Labels and milestones only push on remote that accepts them; otherwise
  warn (capability flag).
