# TECHSTACK.md

Core technology stack for `bb` — a Bitbucket Cloud CLI and TUI built in Go. All entries are derived from `go.mod`, `Makefile`, `.goreleaser.yaml`, `.github/workflows/release.yml`, and the source tree.

## Language and Runtime

- Go `1.26.1` (per `go.mod`)
- Single-binary CLI; entrypoint `main.go` → `cmd.Execute()`
- Cross-compiled for `linux`, `darwin`, `windows` × `amd64`, `arm64` with `CGO_ENABLED=0`

## Core Frameworks and Libraries

- CLI framework: `github.com/spf13/cobra` v1.10.2 (with `github.com/spf13/pflag` v1.0.9)
- TUI framework: `github.com/charmbracelet/bubbletea` v1.3.10 (Elm-style model/update/view)
- TUI components: `github.com/charmbracelet/bubbles` v1.0.0 (list, spinner, viewport, text input, table)
- Terminal styling/layout: `github.com/charmbracelet/lipgloss` v1.1.1-pre
- Markdown rendering: `github.com/charmbracelet/glamour` v1.0.0 (with `yuin/goldmark` v1.7.13)
- HTML sanitization in rendered markdown: `microcosm-cc/bluemonday` v1.0.27
- Browser launcher (OAuth flow): `github.com/pkg/browser`
- GCF output encoding: `github.com/blackwell-systems/gcf-go` v1.2.0 (GCF - Graph Compact Format, compact AI-native output)

## Data and Persistence

- No application database. State is in-process (`cmd/tui/cache.go`) and on-disk config only.
- Config file: `~/.bbcloud.yaml` parsed via `gopkg.in/yaml.v3` v3.0.1 (`internal/config/config.go`).
- History/cache files under user home (see `internal/history/history.go`).
- Remote data source: Bitbucket Cloud REST API v2.0 (via stdlib `net/http`).

## API and Contract Tooling

- Consumer of Bitbucket Cloud REST API v2.0 (no contract files in repo).
- Typed client lives in `pkg/bitbucket/`; JSON decoded via stdlib `encoding/json` with a generic `decode[T]()` helper.
- No OpenAPI/GraphQL/protobuf artifacts present.

## Security and Secrets

- OS keyring storage for credentials: `github.com/zalando/go-keyring` v0.2.6 (macOS Keychain, Windows Credential Manager via `danieljoos/wincred`, Linux libsecret via `godbus/dbus`).
- OAuth 2.0 flow against `bitbucket.org/site/oauth2` with CSRF state from `crypto/rand` (`internal/auth/oauth.go`).
- Masked password/token input via `golang.org/x/term` v0.41.0.
- Tokens are never written to the YAML config (`Token` field is `yaml:"-"`).
- App Password and OAuth supported; precedence: YAML config → `BITBUCKET_USER`/`BITBUCKET_TOKEN` env vars → CLI flags.

## Build and Dependency Management

- Go modules (`go.mod` / `go.sum`).
- `Makefile` targets: `build`, `install`, `test`, `clean`.
- Version stamped via `-ldflags -X 'github.com/payfacto/bb/cmd.Version=...'`, derived from `git describe --tags --always --dirty`.
- Release builds: GoReleaser v2 (`.goreleaser.yaml`) producing tar.gz (Linux/macOS) and zip (Windows) archives with a `checksums.txt`.
- Distribution: Homebrew tap `payfacto/homebrew-tap` (formula output to `Formula/`).

## Testing Stack

- Go stdlib `testing` only — no third-party test framework, no assertion library.
- HTTP client tests use stdlib `net/http/httptest` (`pkg/bitbucket/testhelpers_test.go`).
- Tests live in `pkg/bitbucket/`, `cmd/render/`, `cmd/tui/`, `internal/auth/`, `internal/config/`, `internal/history/`. `cmd/` Cobra wiring is intentionally untested.
- No linter configured; `go fmt` and `go vet` are the standard checks.

## CI/CD and Delivery

- GitHub Actions workflow: `.github/workflows/release.yml`
  - Trigger: pushed tags matching `v*`
  - `actions/checkout@v4`, `actions/setup-go@v5` (Go version from `go.mod`, with module cache)
  - Runs `go test ./...` before release
  - `goreleaser/goreleaser-action@v7.0.0` with `~> v2` builds and publishes
  - Uses `HOMEBREW_TAP_TOKEN` secret for Homebrew tap PR
- No separate CI workflow for PRs detected in the repository.

## Infrastructure and Deployment

- No server-side infrastructure. `bb` is distributed as standalone binaries.
- No Dockerfile, docker-compose, Kubernetes manifests, Terraform, or CloudFormation in the repo.
- Deployment surface: GitHub Releases + Homebrew formula.

## Developer Experience Tooling

- Interactive TUI launched on `bb` with no subcommand (`cmd/tui/`).
- Interactive setup wizard: `bb setup`.
- Shell completion generation: `cmd/completion.go` (Cobra-provided).
- Clipboard support in TUI: `github.com/atotto/clipboard` (transitive via Charm libs).
- Project-level `CLAUDE.md` and `llms.txt` keep agent-readable command reference in sync with `README.md`.
