# Logging and (opt-in) telemetry

## Properties
Type
:   Feature
Area
:   Core
State
:   Proposed
Priority
:   3
Phase
:   4

## Summary
Structured logging via `log/slog` with `--verbose` / `--quiet` / `--json`
controls. No telemetry is collected by default; an opt-in anonymous
error-reporting hook (e.g. local file or user-provided webhook) may be
added later.

## Acceptance criteria
- All log lines include provider, item id, and operation when applicable.
- Sensitive values (PATs, tokens) are never logged; redaction unit-tested.
- `--json` emits one JSON object per line (machine-readable).
