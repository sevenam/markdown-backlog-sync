# markdown-backlog-sync

CLI tool for local sync of Azure DevOps Boards backlogs and GitHub
Issues, using **pure Markdown files** as the source of truth.

> Status: **Phase 1 foundation in place** (CLI skeleton, config,
> Markdown format, sync-state sidecar, provider interface, PAT auth,
> packaging, CI). Provider implementations land in subsequent phases.

## Install

Cross-platform single binary written in Go. From source:

```bash
go install github.com/sevenam/markdown-backlog-sync/cmd/mbs@latest
```

Pre-built binaries will be published via GitHub Releases (see
`.goreleaser.yaml`).

## Quickstart

Build:

```bash
go build -o bin/mbs.exe ./cmd/mbs
```

```bash
mbs init                 # create backlog.config.toml + backlog/ + .sync/
$EDITOR backlog.config.toml
mbs auth login --provider gh-main
mbs status
mbs sync                 # (Phase 2+: requires provider implementation)
```

## Documentation

- [Item file format](docs/item-format.md)
- [Exit codes](docs/exit-codes.md)
- [Implementation backlog](backlog/) — one Markdown file per planned feature

## Repository layout

```
cmd/mbs/                          CLI entry point
internal/cli/                     cobra command tree
internal/cli/exitcode/            typed exit-code error wrapper
internal/config/                  backlog.config.toml loader
internal/workspace/               workspace discovery + init
internal/markdown/                pure-Markdown item parser/serializer
internal/state/                   .sync/ sidecar (atomic JSON store)
internal/provider/                provider interface + registry
internal/provider/mock/           in-memory provider for tests
internal/auth/                    PAT resolver + memory/file backends
internal/auth/keyring/            OS keyring backend (Keychain/CredMgr/libsecret)
backlog/                          implementation backlog (planning items)
```

## License

MIT. See [LICENSE](LICENSE).
