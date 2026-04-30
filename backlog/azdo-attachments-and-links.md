# Azure DevOps attachments and links

## Properties
Type
:   Feature
Area
:   Azure DevOps
State
:   Proposed
Priority
:   3
Phase
:   3

## Summary
Round-trip work-item links (Related, Parent/Child, Predecessor, Hyperlink)
and attachments referenced from the Markdown body.

## Acceptance criteria
- Links section in Markdown lists related items by `RemoteId` or URL and
  syncs to/from `relations` on the work item.
- Image references in the body upload as attachments on first push and
  download into a sibling `attachments/` folder on first pull.
- Attachment binary content is content-addressed (sha256) to avoid
  re-uploading unchanged files.
