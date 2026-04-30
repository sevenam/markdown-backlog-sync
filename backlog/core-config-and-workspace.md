# Configuration and workspace discovery

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
Define a workspace concept (a folder containing a `backlog.config.toml` and
a `backlog/` directory) and the algorithm for discovering it from the
current working directory upward.

## Acceptance criteria
- `mbs init` creates `backlog.config.toml`, an empty `backlog/`, and a
  `.sync/` sidecar directory; adds `.sync/` to `.gitignore` if a git repo
  is detected (interactive prompt).
- Config schema supports multiple named providers, each pinned to one
  remote scope (an Azure DevOps project + team, or a GitHub `owner/repo`).
- Layered config: defaults < workspace file < env vars (`MBS_*`) < flags.
- Workspace discovery walks upward from cwd until a `backlog.config.toml`
  is found; clear error otherwise.
- Validation rejects duplicate provider names and unknown provider types.

## Example config
```toml
[workspace]
items_dir = "backlog"

[[provider]]
name = "gh-main"
type = "github-issues"
repo = "sevenam/markdown-backlog-sync"

[[provider]]
name = "azdo-platform"
type = "azure-devops-boards"
organization = "contoso"
project = "Platform"
team = "Platform Team"
```
