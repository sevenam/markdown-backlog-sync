# Azure DevOps work item type ladder & process templates

## Properties
Type
:   Feature
Area
:   Azure DevOps
State
:   Proposed
Priority
:   2
Phase
:   2
Parent
:   core-hierarchy-model

## Summary
Azure DevOps projects are created from a process template (Basic, Agile,
Scrum, CMMI, or a customized inherited process) that defines the work
item type ladder and allowed parent/child relationships. The provider
must respect the project's actual process — not assume a fixed ladder.

## Reference ladders (built-in defaults)
| Process | Ladder (top → bottom)                                  |
|---------|--------------------------------------------------------|
| Basic   | Epic → Issue → Task                                    |
| Agile   | Epic → Feature → User Story / Bug → Task               |
| Scrum   | Epic → Feature → Product Backlog Item / Bug → Task     |
| CMMI    | Epic → Feature → Requirement / Bug → Task              |

## Behaviour
- On `mbs provider inspect <name>` the AzDo provider fetches the
  project's process via the WIT REST API
  (`_apis/wit/workitemtypes` + `workitemtypecategories`) and writes the
  discovered ladder into `.sync/providers/<name>.json`.
- The hierarchy linter (see `core-hierarchy-model`) consults that
  cached ladder to validate `Type` + `Parent.Type` combinations.
- `Type` values are case-sensitive AzDo type names; an alias map in
  config lets users keep friendly names locally:
  `[provider.type_aliases] story = "User Story"`.
- State transitions are validated against the type's state category
  map (`Proposed`/`InProgress`/`Resolved`/`Completed`/`Removed`) so
  Markdown `State: Done` works regardless of which process is used.

## Link types
Hierarchy uses `System.LinkTypes.Hierarchy-Forward` /
`System.LinkTypes.Hierarchy-Reverse`. The provider must:
- create exactly one Parent link per child (process rule), and
- on reparent, atomically remove the old parent link before adding the
  new one to avoid the "Only one link of type Parent allowed" error.

## Acceptance criteria
- Provider discovers and caches the project's ladder; cache is
  invalidated by `mbs provider refresh <name>`.
- Lint rejects `Type: Task` with `Parent.Type: Epic` on Agile (must go
  through Feature + Story) but accepts it on Basic.
- Reparent operation produces a single coherent change (remove + add)
  even if the underlying REST call requires two requests.
- Custom inherited processes work with no code changes (only cache
  refresh).
