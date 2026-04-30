# GitHub Projects v2 fields mapping

## Properties
Type
:   Feature
Area
:   GitHub Projects v2
State
:   Proposed
Priority
:   2
Phase
:   5

## Summary
Map project custom fields (single-select, number, date, iteration, text)
to Markdown properties.

## Acceptance criteria
- Field discovery: `mbs ghp fields list --provider <name>` enumerates
  available fields and their option ids.
- Config maps Markdown property → field id; unknown fields tolerated.
- Iteration fields validated against the project's iteration
  configuration before push.
- Single-select changes that reference a missing option emit a clear
  error (no silent drops).
