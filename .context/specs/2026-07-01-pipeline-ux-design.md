# Design: pipeline UX slice (build-number addressing, repo UUID, pipeline watch)

Status: approved (brainstorm) - ready for implementation plan
Date: 2026-07-01
Author: brainstormed with Claude
Related: [.context/reference/2026-07-01-pipeline-deploy-enhancement-audit.md](../reference/2026-07-01-pipeline-deploy-enhancement-audit.md) (items #1, #2, #3; #4 re-scoped)

## Summary

First slice of the bb pipeline/deploy enhancement roadmap. Removes the top
friction agents hit with pipelines: opaque UUID addressing, a repository UUID
that bb never surfaces, and the repeated `list -> steps -> log` polling loop.
Three deliverables:

1. Build-number addressing for `pipeline get/stop/steps/log` (`-n/--build-number`).
2. Expose the repository `uuid` in `bb repo get` output.
3. `bb pipeline watch` - a composite that resolves a target, polls to a terminal
   state, and emits a single result (agent-friendly), including detection of a
   paused manual gate with the Bitbucket web URL to resume it.

## Background and constraints

- The "no UUIDs / can't do pipeline-deploy commands" complaint traces to: every
  pipeline command after `list` requires a braced `{uuid}` even though agents see
  `build_number` in `list` output; and `bb repo get` drops the `uuid` field.
- Bitbucket's pipeline resource is addressable by build number as well as UUID
  (`GET pipelines/{identifier}`). The exact integer-path form must be confirmed
  against the live API during TDD; the unit tests assert the path bb constructs.
- Resuming a manual/paused pipeline step is NOT possible via the public API - it
  is open feature request BCLOUD-20050. So the originally-planned #4
  (`pipeline step trigger`) is dropped; its achievable value (surfacing the gate
  and where to resolve it) is folded into `pipeline watch`.
- bb coerces `text` output to `json` when stdout is not a TTY (agent context), so
  `watch` must have a clean non-TTY contract: a single final JSON object.

## Scope

In scope: items #1, #2, #3 above.

Out of scope (tracked in the roadmap backlog): rich `pipeline trigger`
(`--custom/--var/--tag/--commit/--env-uuid`), environment CRUD and env-var
management, `bb pr update`, `pr create` auto-detect, `deployment get`,
`pipeline-var update`, reliability investigation of `pr list` null returns, and
the lower-demand ops surface. The `env get` doc-drift bug in CLAUDE.md is left for
the backlog env work, not fixed here.

## Command surface

```
bb pipeline get    (-u UUID | -n BUILD)
bb pipeline stop   (-u UUID | -n BUILD)
bb pipeline steps  (-u UUID | -n BUILD)
bb pipeline log    (-u UUID | -n BUILD) --step-uuid UUID
bb pipeline watch  [-n BUILD | -u UUID | -b BRANCH] [--tail-log] [--interval 5] [--timeout 0]

bb repo get <slug>   # output now includes the repository uuid
```

Flag rules for get/stop/steps/log:
- Add `-n / --build-number INT`. Existing `-u / --pipeline-uuid` stays.
- Exactly one of `-u` / `-n` is required. If both are set, or neither, return a
  `validation_failed` CLIError with a clear message. Drop the current
  `MarkFlagRequired("pipeline-uuid")` and validate manually in RunE (mirrors the
  stdin-capable command pattern that validates required fields in RunE).

`watch` selection:
- `-n BUILD`, `-u UUID`, or `-b BRANCH` (latest pipeline on that branch).
- No selector: watch the most recent pipeline in the repo.
- `-n`/`-u`/`-b` are mutually exclusive; more than one is `validation_failed`.

## Client layer (`pkg/bitbucket/pipeline.go` + `types.go`)

New / changed methods on `PipelineResource`:

```go
// GetByBuildNumber fetches a pipeline by its integer build number.
func (r *PipelineResource) GetByBuildNumber(ctx context.Context, buildNumber int) (Pipeline, error)
// GET /repositories/{ws}/{repo}/pipelines/{build_number}

// Latest returns the most recent pipeline in the repo, or the most recent on
// branch when branch != "". Uses List(sort="-created_on") and, for a branch,
// filters on target.ref_name; returns the first result (or a not_found error
// when there are none).
func (r *PipelineResource) Latest(ctx context.Context, branch string) (Pipeline, error)
```

- `stop`/`steps`/`log` invoked with `-n` resolve the build number to a UUID via
  `GetByBuildNumber` in the cmd layer, then call the existing UUID-based methods.
  No new stop/steps/log endpoints (DRY).
- `Repo` struct gains `UUID string \`json:"uuid"\``. The Bitbucket API already
  returns it; the struct silently dropped it. Render it in `RepoDetailString`.
  Confirm `bb workspace list` already surfaces its UUID (the `Workspace` struct
  has the field); no change expected there.

Struct additions for watch state detection (confirm exact field names against
live data during TDD): the `Pipeline`/`PipelineState` model may need a
`PipelineStage` name and step-level fields sufficient to detect a paused manual
gate. Existing types: `PipelineState{Name, Result, Stage}` and
`PipelineStep{Name, State, ...}` already exist; extend minimally only if the poll
loop needs more to classify a gate.

## `pipeline watch` behavior

1. Resolve the target selector to a concrete pipeline (uuid + build number):
   `-n` -> GetByBuildNumber; `-u` -> Get; `-b` -> Latest(branch); none ->
   Latest("").
2. Poll loop every `--interval` seconds (default 5):
   - Fetch pipeline + steps.
   - Evaluate stop conditions (below).
   - If `--tail-log`, stream new log lines of the currently-running step to
     stderr (stdout is reserved for the final JSON).
3. Stop conditions and exit codes:
   - Pipeline COMPLETED, result SUCCESSFUL -> status `success`, exit 0.
   - Pipeline COMPLETED, result FAILED/ERROR/STOPPED -> status `failed` (carry
     the result name), exit 1.
   - Paused manual gate detected (pipeline halted waiting on a `trigger: manual`
     step) -> status `blocked`, exit 2. Output includes the gate step and the
     Bitbucket web URL to resume it (from the pipeline's HTML link, or
     constructed as `https://bitbucket.org/{ws}/{repo}/pipelines/results/{build_number}`).
   - `--timeout > 0` elapsed before a terminal state -> status `timeout`, exit 3.
     Default `--timeout 0` means no timeout (watch until terminal/blocked).
4. Output contract:
   - Non-TTY: no incremental output; on stop, emit one JSON object to stdout:
     `{pipeline, steps, status, manual_gate?: {step, url}}`. Exit code encodes
     outcome. (json/gcf both emit the typed struct.)
   - TTY: live-updating step view during polling; final summary on stop.
   - The exit-code contract makes `bb pipeline watch -n 5 && deploy` correct:
     proceeds only on success (0); a manual gate (2) or failure (1) stops the chain.

The precise Bitbucket state model for "paused manual gate" (stage/HALTED
semantics) will be validated against a live pipeline during TDD; the classifier
lives in one small, unit-tested function fed the fetched pipeline+steps, so the
poll loop and the classification are testable independently.

## Output / rendering

- `json` (default) and `gcf` emit the typed structs unchanged for get/steps.
- New `cmd/render` helpers: a `watch` summary renderer (text) and the TTY live
  view. `pipeline get`/`steps` text output unchanged aside from any new fields.

## Errors

- Flow through the existing `APIError -> CLIError` mapping.
- Ambiguous/missing pipeline selector (`-u`+`-n`, or neither where one is
  required; multiple watch selectors) -> `validation_failed` with guidance.
- `Latest` with no pipelines -> `not_found`.
- A build number or UUID that does not exist -> whatever Bitbucket returns,
  mapped (`not_found`).

## Agent manifest (`--describe`)

- Add `pipeline watch` as a `read`-class leaf in `commandRegistry` (output type:
  the watch result struct or a suitable existing type; add to `typeRegistry` if
  new). `get/stop/steps/log` gain the `-n` flag but remain the same leaves.
- Regenerate `cmd/testdata/manifest.golden.json` via `go test ./cmd/ -update`;
  `TestEveryLeafIsRegistered` / `TestManifestSnapshot` enforce it.

## Testing

TDD, stdlib `net/http/httptest`, in `pkg/bitbucket/pipeline_test.go`:

- `GetByBuildNumber`: asserts the constructed path and decodes the pipeline.
- `-n` -> uuid resolution for stop/steps/log (assert the resolved UUID is used).
- `Latest`: no-branch and branch modes (assert sort/filter query and first-result
  selection; not_found on empty).
- `watch`: a handler whose returned state advances across polls exercises
  poll-to-success, poll-to-failure, and manual-gate/blocked classification. Test
  the gate classifier as a pure function on representative pipeline+steps inputs.
- `Repo` uuid decode test.
- `cmd/` remains untested per repo convention (thin wiring); the watch classifier
  and client methods carry the coverage.

## Documentation sync (same change set)

Required by CLAUDE.md when commands/flags change:

1. README.md - Commands block: `-n/--build-number` on get/stop/steps/log, the new
   `pipeline watch` with its flags and exit-code contract, and the repo uuid note.
2. llms.txt - condensed reference matching README.
3. CLAUDE.md - command hierarchy (`pipeline ... watch`) and, if a client method
   signature is referenced, the Client-pattern block (GetByBuildNumber / Latest).

## Open items to resolve during implementation (not now)

1. Confirm `GET pipelines/{build_number}` accepts the plain integer path form
   (near-certain; unit tests assert bb's constructed path regardless).
2. Confirm the Bitbucket pipeline/step state fields that identify a paused manual
   gate, and add the minimal struct fields needed for the classifier.
