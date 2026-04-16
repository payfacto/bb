# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o bb .          # Build the CLI binary (version = "dev")
make build                # Build with git-derived version stamp
make test                 # go test ./...
go test ./...             # Run all tests
go test ./pkg/bitbucket/  # Run client-layer tests only
go test -run TestName ./pkg/bitbucket/  # Run a single test
```

Standard `go fmt` and `go vet` apply. No linter is configured.

## Versioning

Version is injected via `-ldflags -X 'github.com/payfacto/bb/cmd.Version=...'`.

- `cmd/root.go` defines `var Version = "dev"` and wires it into `rootCmd.Version`
  (enables `bb --version`).
- `cmd.Execute` forwards `Version` to `tui.Run(client, cfg, version)`; the TUI
  stashes it in the `tui.version` package-level var and renders it in the home
  header via `subtitleStyle` (`cmd/tui/menu.go`).
- `Makefile` derives the version from `git describe --tags --always --dirty`.
- `.goreleaser.yaml` injects `v{{.Version}}` at release time; the
  `.github/workflows/release.yml` workflow runs GoReleaser on any pushed tag
  matching `v*`.

When renaming/moving the `Version` variable, update both `Makefile` and
`.goreleaser.yaml` ldflags targets.

## Architecture

`bb` is a Cobra-based CLI that wraps the Bitbucket Cloud REST API v2.0. Three packages with distinct responsibilities:

- **`cmd/`** — Cobra command definitions. Thin layer: flag parsing, calling `pkg/bitbucket`, printing output. No business logic.
- **`pkg/bitbucket/`** — Typed HTTP client. All API interaction lives here. Tests are here (using `httptest`).
- **`internal/config/`** — Loads `~/.bbcloud.yaml`, merges env vars and CLI flags. Consumed by `cmd/root.go`.

### Command wiring

`cmd/root.go` defines `PersistentPreRunE` which runs before every subcommand: it loads config, validates required fields, and constructs the `*bitbucket.Client`. The client is stored as a package-level var `bbc` used by all subcommands.

Command hierarchy:
```
bb
└── pr
    ├── list / get / create / diff / approve / merge / decline
    ├── comment list / add / reply
    └── task list / complete / reopen
└── setup   (interactive config wizard)
```

### Client pattern

Resources are scoped structs returned by the client:

```go
client.PRs(workspace, repo).List(ctx, state)
client.Comments(workspace, repo, prID).Add(ctx, input)
client.Tasks(workspace, repo, prID).Complete(ctx, taskID)
```

The generic `decode[T]()` function handles all JSON unmarshaling. HTTP errors (4xx/5xx) are checked and wrapped before returning.

### Configuration precedence (low → high)

1. `~/.bbcloud.yaml` (or `--config` path)
2. `BITBUCKET_USER` / `BITBUCKET_TOKEN` env vars
3. `--username` / `--token` / `--workspace` / `--repo` flags

### Output

`printOutput(v, textFn)` in `cmd/root.go` handles dual-mode output. Default is JSON (machine-readable); `--format text` invokes a per-command `textFn`. New commands should follow this pattern.

### Testing

Tests live only in `pkg/bitbucket/`. They use stdlib `net/http/httptest` via `newTestClient()` (in `testhelpers_test.go`) — no mock frameworks. `cmd/` is intentionally untested (thin Cobra wiring).

## Key dependencies

| Package | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI framework |
| `gopkg.in/yaml.v3` | Config file parsing |
| `golang.org/x/term` | Masked password input in `bb setup` |

## Design reference

`docs/superpowers/specs/2026-04-04-bb-cli-design.md` — approved design doc with full command spec, auth strategy, and out-of-scope decisions.
