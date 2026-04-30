# Error handling and exit codes

## Properties
Type
:   Feature
Area
:   Core
State
:   Proposed
Priority
:   2
Phase
:   1

## Summary
Define a typed error hierarchy and a documented exit-code contract so
scripts can react predictably.

## Exit codes
- 0 success
- 1 generic error
- 2 usage / config error
- 3 authentication / authorization
- 4 conflict requiring user action
- 5 network / remote API error
- 6 workspace integrity error

## Acceptance criteria
- Errors wrap underlying causes (`errors.Join`/`%w`) and surface a
  single, actionable top-line message.
- `--verbose` prints the full chain.
- Documented in `docs/exit-codes.md`.
