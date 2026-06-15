# Design: GCF output format for `bb`

Date: 2026-06-15
Status: Approved (pending implementation plan)

## Problem

`bb` emits JSON by default (machine-readable) or human `text` via `--format text`.
JSON is verbose. GCF (Graph Compact Format, https://github.com/blackwell-systems/gcf)
is an AI-native wire format that encodes the same structured data in ~71% fewer tokens,
which directly reduces cost and context pressure for the agents that consume `bb` output.

We want to:

1. Add `gcf` as a supported output format.
2. Make `gcf` the new built-in default (with a JSON escape hatch for existing automation).
3. Let users persist a preferred format once so they never have to pass `--format`.

## Goals

- Support three output formats: `gcf`, `json`, `text`.
- Ship `gcf` as the default for both interactive (TTY) and piped output.
- Provide a low-friction JSON escape hatch (`BB_FORMAT=json` or config) so existing
  scripts/agents keep working with a one-line change.
- Persist the preferred format in `~/.bbcloud.yaml`, settable via the `bb setup` wizard.
- Apply the format consistently to success output, error output, and the non-TTY guard.
- Keep the `--describe` manifest, README, llms.txt, and CLAUDE.md in sync (repo rule).

## Non-goals

- GCF's *graph profile* (for code/knowledge graphs). `bb` data is tabular records, so
  only the *generic profile* is used.
- A general `bb config` subcommand. Persistence is via the config file + setup wizard only.
- Changing the human `text` renderers in `cmd/render/`.
- Streaming/session-dedup GCF features (`NewStreamEncoder`, `DeltaPayload`). Out of scope
  for this iteration; one-shot `EncodeGeneric` per command is sufficient.

## Dependency

- `github.com/blackwell-systems/gcf-go` (MIT, zero runtime deps) — added to `go.mod`.
  Used directly via its public API (`EncodeGeneric(v any) string`); no reimplementation
  of the format. This is a hard requirement.

## Design

### Format model & precedence

Valid formats: `gcf`, `json`, `text`. Resolution precedence (low -> high):

1. Built-in default: `gcf`
2. `~/.bbcloud.yaml` `format:` field
3. `BB_FORMAT` environment variable
4. `--format` / `-f` flag

Unknown values produce a `validation_failed` CLI error listing the allowed set, rather
than silently falling through to JSON as the current `printOutput` does.

Format-to-backend mapping:

- `gcf` -> `gcf.EncodeGeneric(v)` (generic profile). Homogeneous record lists (PRs,
  repos, commits) map to GCF's tabular `## section [n]{fields}` form; single objects
  become section headers.
- `json` -> `encoding/json` (unchanged).
- `text` -> per-command `textFn` closures backed by `cmd/render/` (unchanged).

### Default flip & the non-TTY guard

Today `cmd/root.go` (PersistentPreRunE) forces `json` when stdout is not a TTY and
`--format` was not explicitly set. New behavior:

- Built-in default becomes `gcf` for both TTY and piped output.
- The guard's intent is preserved but narrowed to its real purpose: never pipe the human
  `text` format to a machine. New rule: if stdout is **not** a TTY and the resolved format
  is `text` but `text` was **not** explicitly requested (flag/env/config), coerce to `gcf`.
  An explicit `text` choice is honored even when piped.
- Escape hatch: existing automation pins JSON with `BB_FORMAT=json` or `format: json`.
- This is a documented breaking change: README/release notes state "bb now defaults to
  GCF; set `BB_FORMAT=json` to restore JSON."

### `cmd/output.go` (consolidated dispatcher)

New file in package `cmd`, the single source of truth for format concerns. Called by
both `root.go` and `errors.go`:

```go
// allowedFormats drives validation and the --describe manifest enum.
var allowedFormats = []string{"gcf", "json", "text"}

// resolveFormat applies precedence (default < config < BB_FORMAT < --format) and the
// non-TTY text-guard. Called once in PersistentPreRunE.
func resolveFormat(cmd *cobra.Command, cfg *config.Config) (string, error)

// renderValue switches format -> gcf.EncodeGeneric / json encoder / textFn.
func renderValue(v any, textFn func()) error

// renderError renders a *CLIError in the active format.
func renderError(e *CLIError)
```

`printOutput` keeps its name (~40 call sites) but its body re-points to `renderValue`,
adding the `gcf` branch to the existing `text`/json dispatch.

### Error rendering

`emitError` routes through `renderError`:

- `json` -> today's exact `{ "error": { "code", "message", "details" } }` output
  (unchanged contract for JSON users).
- `gcf` -> `gcf.EncodeGeneric(e)`, one GCF record.
- `text` -> a concise `error: <code>: <message>` line.

Errors still go to stderr and still exit non-zero; only serialization changes.

### Config & setup wizard

- `internal/config/config.go`: add `Format string` with `yaml:"format,omitempty"`, a
  `FormatDefault = "gcf"` constant, and `Validate()` rejection of unknown values.
- `bb setup`: add a format picker slot beside the existing theme selector (left/right
  cycles `gcf` / `json` / `text`), persisted to `~/.bbcloud.yaml`.

## Components and data flow

1. `PersistentPreRunE` loads config, then `resolveFormat` computes the effective format
   (precedence + non-TTY guard) and stores it in the existing package-level `format` var.
2. Subcommands call `printOutput(v, textFn)` -> `renderValue` switches on `format`.
3. On failure, `Execute` calls `emitError` -> `renderError` using the same `format`.

Each unit is independently testable: `resolveFormat` (pure, given cmd+cfg+TTY), `renderValue`
(given a value + format), `renderError` (given a CLIError + format).

## Error handling

- Unknown `--format`/`BB_FORMAT`/config value -> `validation_failed` error before any
  command runs.
- GCF encoding of an unexpected shape: `EncodeGeneric` is total over Go values; if it ever
  returns an empty/degenerate result we surface an `internal_error` rather than empty stdout.

## Testing

- `cmd/output_test.go`: precedence resolution (all four layers), the non-TTY text-guard
  (coerce vs honor-explicit), validation errors, and a GCF golden sample for a representative
  list payload.
- `internal/config`: `Format` default + round-trip tests (mirror existing `Theme` tests).
- Regenerate `cmd/testdata/manifest.golden.json` via `go test ./cmd/ -update`;
  `TestManifestSnapshot` enforces the manifest enum change.

## Docs sync (required by CLAUDE.md)

- `--format` flag help: "output format: gcf, json, or text", default `gcf`.
- README.md output section: new default, format list, `BB_FORMAT`, migration note.
- llms.txt: format reference + default.
- CLAUDE.md: Output section, default, `BB_FORMAT`, error-follows-format note.

## Alternatives considered

- **Default for TTY only, JSON when piped** (non-breaking): keeps the current guard
  forcing JSON for pipes. Rejected because agents - the consumers GCF is built for - are
  exactly the piped case, so they would never get the token savings by default.
- **GCF everywhere with no escape hatch**: simplest mental model but hard-breaks every
  existing JSON consumer with no migration path. Rejected.
- **Errors always JSON**: keeps the error contract stable across formats. Rejected in
  favor of errors-follow-format for a consistent single-format experience (user decision).
- **`internal/output` package with a `Renderer` interface** (Approach B): cleaner for many
  future formats, but heavier scaffolding for three formats and leaky for `text` (which
  needs the per-command `textFn`). Rejected for now.
- **Inline the `gcf` branch across root.go + errors.go** (Approach A): smallest diff but
  scatters format logic. Rejected in favor of one `cmd/output.go` home (Approach C).

## Open questions

None outstanding. Behavior, default, persistence, error rendering, and dependency are all
decided.

## References

- GCF format: https://github.com/blackwell-systems/gcf
- Go library: https://github.com/blackwell-systems/gcf-go (`EncodeGeneric`, `Encode`)
- Current output dispatch: `cmd/root.go` (`printOutput`, PersistentPreRunE guard)
- Error envelope: `cmd/errors.go` (`CLIError`, `emitError`, `mapError`)
- Text backend: `cmd/render/`
- Config + wizard pattern precedent: `internal/config/config.go` (`Theme`), `cmd/tui/setup.go`
