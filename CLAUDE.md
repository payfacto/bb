@.context/INDEX.md

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

`bb` is a Cobra-based CLI that wraps the Bitbucket Cloud REST API v2.0. Four packages with distinct responsibilities:

- **`cmd/`** — Cobra command definitions. Thin layer: flag parsing, calling `pkg/bitbucket`, printing output. No business logic.
- **`cmd/tui/`** — Bubble Tea TUI application. Launched when `bb` is run with no subcommand. Elm-style architecture: `app.go` (model/update/view), `menu.go` (home), `list.go` (list views), `detail.go` (PR/resource detail), `sections.go` (drill-down panels), `nav.go` (breadcrumb navigation), `styles.go` + `themes.go` (lipgloss styling), `keys.go` (key bindings), `cache.go` (in-process data cache).
- **`pkg/bitbucket/`** — Typed HTTP client. All API interaction lives here. Tests are here (using `httptest`).
- **`internal/config/`** — Loads `~/.bbcloud.yaml`, merges env vars and CLI flags. Consumed by `cmd/root.go`.

### Command wiring

`cmd/root.go` defines `PersistentPreRunE` which runs before every subcommand: it loads config, validates required fields, and constructs the `*bitbucket.Client`. The client is stored as a package-level var `bbc` used by all subcommands.

Command hierarchy:
```
bb                    (no args → launches TUI)
├── pr
│   ├── list / get / create / diff / approve / merge / decline
│   ├── activity / statuses / add-reviewer
│   ├── comment list / get / add / reply
│   └── task list / complete / reopen
├── pipeline list / get / trigger / stop / steps / log
├── pipeline-var list / create / delete
├── branch list / create / delete
├── tag list / create / delete
├── commit list / get
├── file get
├── repo list / get / create / update / delete / fork
├── issue list / get / create / close / reopen
├── deployment list
├── env list / get
├── member list
├── user me / get
├── webhook list / create / delete
├── deploy-key list / create / delete
├── restriction list / create / delete
├── download list / get / upload / delete
├── workspace list
└── setup   (interactive config wizard)
```

### Client pattern

Resources are scoped structs returned by the client:

```go
client.PRs(workspace, repo).List(ctx, state, sourceBranch, sort)
client.Repos(workspace).List(ctx, sort)
client.Branches(workspace, repo).List(ctx, sort)
client.Tags(workspace, repo).List(ctx, sort)
client.Commits(workspace, repo).List(ctx, branch, sort)
client.Issues(workspace, repo).List(ctx, sort)
client.Pipelines(workspace, repo).List(ctx, sort)
client.Comments(workspace, repo, prID).Add(ctx, input)
client.Tasks(workspace, repo, prID).Complete(ctx, taskID)
```

List methods whose Bitbucket endpoint accepts a `sort=` query parameter take
a trailing `sort string` argument. Pass `""` to preserve the endpoint's
default ordering. The CLI surfaces this via `--sort` on the corresponding
list command.

The generic `decode[T]()` function handles all JSON unmarshaling. HTTP errors
(4xx/5xx) are wrapped into `*bitbucket.APIError{Status, Message, Body}` so
callers (`cmd/errors.go`) can map them to stable CLI error codes.

### Configuration precedence (low → high)

1. `~/.bbcloud.yaml` (or `--config` path)
2. `BITBUCKET_USER` / `BITBUCKET_TOKEN` env vars
3. `--username` / `--token` / `--workspace` / `--repo` flags

### Output

`printOutput(v, textFn)` in `cmd/root.go` delegates to `renderValue` in
`cmd/output.go`. Supported formats: `json | gcf | text`. The built-in default
is `json` (pipe-friendly, works with `jq`). `--format text`
invokes a per-command `textFn`; `--format json` emits indented JSON; `--format gcf`
encodes via `gcf.EncodeGeneric` (Graph Compact Format - compact AI-native
encoding, ~71% fewer tokens than JSON; opt in via `--format gcf` / `BB_FORMAT=gcf`).
New commands should follow this pattern.

Format precedence (low to high): built-in default `json` < `~/.bbcloud.yaml`
`format:` field < `BB_FORMAT` env var < `--format`/`-f` flag. Resolution
happens in `resolveFormat` / `resolveFormatFrom` in `cmd/output.go`, called
once in `PersistentPreRunE` after config is loaded.

Non-TTY guard: when stdout is not a terminal and the resolved format is `text`,
it is coerced to `json` unless `--format` was set on this specific
invocation (an explicit per-command `--format text` is always honored).

Commands that override `PersistentPreRunE` but still emit output (e.g. `bb user
me`) MUST load config via `loadConfig(cmd)` (in `cmd/root.go`) rather than
`config.Load` directly — Cobra runs only the nearest `PersistentPreRunE` and does
not chain to the parent's, so `loadConfig` is what guarantees `resolveFormat`
runs (and thus that persisted `format:` / `BB_FORMAT` and the non-TTY guard apply).

Errors are rendered in the active format via `renderError` in `cmd/output.go`:

- `format=json` - writes `{"error": {"code", "message", "details"}}` to stderr
  (byte-identical to the historical envelope; JSON consumers see no change).
- `format=gcf` - encodes a `{code, message, details?}` map view via
  `gcfErrorView` + `gcf.EncodeGeneric` (no `"error"` wrapper key, `details`
  omitted when empty; the unexported `cause` is never encoded).
- `format=text` - writes `error: <code>: <message>` to stderr.

Errors flow through `mapError` in `cmd/errors.go` first (maps raw errors to
`*CLIError`). Codes: `config_missing`, `auth_failed`, `not_found`,
`validation_failed`, `conflict`, `rate_limited`, `api_error`, `internal_error`.
Subcommands return `*CLIError` for typed errors or plain `error` for default
mapping. `rootCmd.SilenceErrors = true` keeps Cobra's prose printer from
interleaving with formatted output.

### Agent affordances

- `bb --describe` (root-level boolean flag) walks the Cobra tree and emits a
  JSON capability manifest covering every command, flag, action class
  (`read | write | destructive`), output Go type, and JSON Schema. Wiring:
  `cmd/describe.go` holds the walker, types, and reflection;
  `cmd/manifest_registry.go` holds the `commandRegistry` (data) and
  `typeRegistry` (Go zero values for schema reflection). The registry MUST
  contain every leaf — `TestEveryLeafIsRegistered` enforces this; the
  reverse `TestRegistryReferencesOnlyRealCommands` catches stale entries.
  When adding a new leaf, also add an entry to `commandRegistry` and (if it
  returns a new type) `typeRegistry`. Bump `manifestSchemaVersion` in
  `cmd/describe.go` if you change the manifest's wire shape.
- The manifest's structural shape is locked by `TestManifestSnapshot` against
  `cmd/testdata/manifest.golden.json`. Run `go test ./cmd/ -update` after
  intentional manifest changes.
- Create and update commands accept JSON on stdin via `stdinInputOr` in
  `cmd/stdin.go`. Stdin-capable commands drop `MarkFlagRequired` calls and
  validate required fields manually inside RunE via `requireFlag`.

### Testing

Tests live only in `pkg/bitbucket/`. They use stdlib `net/http/httptest` via `newTestClient()` (in `testhelpers_test.go`) — no mock frameworks. `cmd/` is intentionally untested (thin Cobra wiring).

### Documentation sync — REQUIRED on every command change

Whenever a command or flag is **added, removed, renamed, or its signature changes**, update **all three** of these in the same commit:

1. **`README.md`** — the "Commands" reference block (look for `bb pr list ...`, `bb pr create ...`, etc.) and any narrative examples that reference the changed shape.
2. **`llms.txt`** — the condensed command reference. Keep flag shapes in sync with README; this file is what agents read.
3. **`CLAUDE.md`** (this file) — the "Command hierarchy" tree near the top and any code example using the affected client method.

Also update the relevant example in the "Client pattern" code block if a `pkg/bitbucket` method signature changed (e.g. `client.PRs(ws, repo).List(ctx, state, sourceBranch, sort)`).

When adding a new leaf command, also add an entry to `commandRegistry` (and `typeRegistry` if a new type) in `cmd/describe.go`. The `TestEveryLeafIsRegistered` invariant fails the build if you forget.

PRs that touch `cmd/` or `pkg/bitbucket/*.go` public APIs without updating these three files should be considered incomplete.

## Key dependencies

Direct dependencies only (`go.mod` `require` block):

| Package | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI framework |
| `gopkg.in/yaml.v3` | Config file parsing |
| `golang.org/x/term` | Masked password input in `bb setup`; TTY detection for piped stdout/stdin |
| `github.com/invopop/jsonschema` | JSON Schema reflection for the `--describe` manifest |
| `github.com/zalando/go-keyring` | OS keyring storage for credentials (macOS Keychain, Windows Credential Manager, Linux libsecret) |
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm-style model/update/view) |
| `github.com/charmbracelet/bubbles` | TUI components — list, spinner, viewport, text input, table |
| `github.com/charmbracelet/lipgloss` | Terminal layout and styling (borders, colours, alignment) |
| `github.com/charmbracelet/glamour` | Markdown rendering in the terminal (PR descriptions, diffs) |

## Design reference

Design specs and TDD implementation plans live in `.context/specs/` and
`.context/plans/` (indexed by `.context/INDEX.md`) — e.g.
`.context/specs/2026-05-29-api-token-auth-design.md`. The original
`2026-04-04-bb-cli-design.md` is not currently present in the repo; see the
Outstanding backlog in `.context/HANDOFF.md`.
