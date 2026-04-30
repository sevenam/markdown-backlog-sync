# Azure DevOps work item field mapping

## Properties
Type
:   Feature
Area
:   Azure DevOps
State
:   Proposed
Priority
:   1
Phase
:   3

## Summary
Map Markdown item properties to/from Azure DevOps work-item fields.

## Mapping (default)
| Markdown property | Azure DevOps field             |
| ----------------- | ------------------------------ |
| H1 title          | `System.Title`                 |
| Body (`## Summary`)| `System.Description` (HTML)   |
| `## Acceptance criteria` | `Microsoft.VSTS.Common.AcceptanceCriteria` |
| Type              | `System.WorkItemType`          |
| State             | `System.State`                 |
| Assignee          | `System.AssignedTo`            |
| Labels            | `System.Tags` (semicolon list) |
| Iteration         | `System.IterationPath`         |
| AreaPath          | `System.AreaPath`              |
| Parent            | Parent link (`System.LinkTypes.Hierarchy-Reverse`); see `azdo-work-item-types-ladder` for the ladder rules |
| Estimate          | `Microsoft.VSTS.Scheduling.StoryPoints` |

## Acceptance criteria
- Markdown ↔ HTML conversion is round-trip-stable for the supported
  subset (headings, lists, code, links, images).
- Custom field overrides allowed via config:
  `[provider.field_map] "Risk" = "Custom.Risk"`.
- Unmapped properties survive on the Markdown side and emit a warning.
