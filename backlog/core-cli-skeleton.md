# CLI skeleton (cobra + viper)

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

## Summary
Bootstrap the Go module and CLI skeleton using `spf13/cobra` for command
parsing and `spf13/viper` for layered configuration.

## Acceptance criteria
- `go install` builds a single binary named `mbs`.
- Commands present (stubs OK): `init`, `pull`, `push`, `sync`, `status`,
  `auth login`, `auth logout`, `provider add`, `provider list`, `version`.
- `--workspace`, `--config`, `--verbose`, `--json`, `--dry-run` global flags.
- Exit codes follow a documented convention (0 ok, 1 generic, 2 usage,
  3 auth, 4 conflict, 5 network).
- `mbs --help` renders for every subcommand.

## Notes
Use `cobra-cli` to scaffold. Embed build metadata (version, commit, date)
via `-ldflags`.
