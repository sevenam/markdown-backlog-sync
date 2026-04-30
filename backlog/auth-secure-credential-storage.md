# Secure credential storage

## Properties
Type
:   Feature
Area
:   Authentication
State
:   Proposed
Priority
:   1
Phase
:   1

## Summary
Use the OS-native secret store for PAT persistence via the
`zalando/go-keyring` library (or equivalent).

## Backends
- macOS: Keychain
- Windows: Credential Manager
- Linux: Secret Service (libsecret) with documented fallback to an
  encrypted file (`age`-based) when no DBus session is available
  (headless CI).

## Acceptance criteria
- Service name namespaced as `markdown-backlog-sync:<providerName>`.
- Graceful, actionable error if the keyring is unavailable, with the
  encrypted-file fallback opt-in via `--credentials-file`.
- Round-trip integration tests on each OS in CI.
