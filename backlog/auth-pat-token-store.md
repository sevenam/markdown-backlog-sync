# PAT token store and resolution

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
v1 authentication uses Personal Access Tokens for both Azure DevOps and
GitHub. Provide a uniform CLI for storing, listing, and removing them per
provider.

## Commands
- `mbs auth login --provider <name>` — prompts for a PAT (masked input)
  and stores it securely.
- `mbs auth status` — shows which providers have credentials and the
  token's reported scopes (validated against the API).
- `mbs auth logout --provider <name>` — deletes the stored credential.

## Resolution order at runtime
1. `--token` flag (discouraged; warns if used).
2. Env var: `MBS_TOKEN_<PROVIDER>` (e.g. `MBS_TOKEN_GH_MAIN`).
3. Generic env vars: `GITHUB_TOKEN`, `AZURE_DEVOPS_PAT`.
4. OS credential store (see secure-credential-storage item).

## Acceptance criteria
- Tokens never written to plain files by default.
- Token is validated on `auth login` (e.g. GET `/user` for GH, GET
  `_apis/connectionData` for Azure DevOps).
- `--token` and env-var paths documented for CI usage.
