# (Deferred) OAuth Device Flow / GitHub App auth

## Properties
Type
:   Spike
Area
:   Authentication
State
:   Deferred
Priority
:   4
Phase
:   Later

## Summary
v1 ships PAT-only auth per product decision. This item captures the
design for a future, friendlier auth path so it can be added without
breaking changes.

## Options to evaluate later
- **GitHub App + OAuth Device Flow** with a public client id baked into
  the binary; user enters a code at `https://github.com/login/device`.
  No client secret needed. Refresh tokens supported.
- **Reuse `gh auth token`** when the GitHub CLI is installed on the host
  — zero extra setup for the user.
- **Azure DevOps via Microsoft Entra ID device code** (MSAL Go) for
  organizations where PATs are disabled.
- **Reuse `az account get-access-token`** when Azure CLI is installed.

## Acceptance criteria (for the future feature)
- `mbs auth login --provider <name> --method device` works without
  pre-creating a PAT.
- Tokens (and refresh tokens, if any) stored via the same secure
  credential layer.
- Falls back to PAT seamlessly.
