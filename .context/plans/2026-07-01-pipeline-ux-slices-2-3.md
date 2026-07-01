# Pipeline UX slices #2 (repo UUID) + #3 (pipeline watch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Surface the repository `uuid` in `bb repo get` (#2), and add `bb pipeline watch` (#3) - a poll-to-terminal command with an agent-friendly single-JSON contract, exit codes, and manual-gate detection.

**Architecture:** #2 is a struct field + one render line. #3 adds client methods (`Latest`, `Watch`) and a pure `classifyPipelineState` classifier in `pkg/bitbucket`, plus a `watch` Cobra command whose terminal outcome sets a package-level `exitCode` honored by `cmd.Execute`.

**Tech Stack:** Go 1.26, Cobra, stdlib `net/http/httptest`, `golang.org/x/term` for TTY detection. No new deps.

## Global Constraints

- **Slices #2 and #3 only.** Backlog items (#5+) are out of scope.
- **Tests live in `pkg/bitbucket/`** (external `bitbucket_test` + one internal `bitbucket` file for the unexported classifier). Pure `cmd` helpers (`watchExitCode`, selector validation) get table tests in `cmd/`; Cobra RunE wiring stays untested.
- **Manual-gate state model (confirmed via Atlassian docs):** pipeline `state.name == "IN_PROGRESS"` with `state.stage.name == "PAUSED"` (workflow/manual pause) or `"HALTED"` (system pause) => blocked. `COMPLETED` + `result.name == "SUCCESSFUL"` => success; any other result => failed. Classification uses only existing struct fields.
- **Non-TTY contract:** exactly one JSON object to stdout on stop; progress/tail-log go to stderr and only when stderr is a TTY. bb already coerces text->json/gcf when stdout is not a TTY.
- **Exit codes:** watch success=0, failed=1, blocked=2, timeout=3. Genuine errors (network, not-found) return an error -> envelope + exit 1.
- **No em-dashes; ASCII only. End commits with:** `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`. **Do not push.**
- **Docs sync (CLAUDE.md rule):** README, llms.txt, and CLAUDE.md (the command tree gains a new `watch` leaf this time).

---

## File Structure

- `pkg/bitbucket/types.go` - add `Repo.UUID`; add `PipelineWatchStatus`, `ManualGate`, `PipelineWatchResult`.
- `pkg/bitbucket/pipeline.go` - add `Latest`, `Watch`, `classifyPipelineState`, `firstIncompleteStep`, `pipelineWebURL`, `WatchOptions`.
- `pkg/bitbucket/pipeline_test.go` - `Latest`, `Watch` tests.
- `pkg/bitbucket/pipeline_internal_test.go` (new, package `bitbucket`) - `classifyPipelineState` table test.
- `pkg/bitbucket/repo_test.go` - `Repo.UUID` decode test.
- `cmd/render/repo.go` - render UUID line.
- `cmd/render/pipeline.go` - `PipelineWatch` renderer.
- `cmd/pipeline.go` - `watch` command; extend `pipelineSelector` with `branch`; `validateWatchSelector`, `resolveWatchPipeline`, `watchExitCode`, progress/tail-log helper.
- `cmd/root.go` - package-level `exitCode`; `Execute` honors it.
- `cmd/pipeline_test.go` - `watchExitCode` + `validateWatchSelector` table tests.
- `cmd/manifest_registry.go` + `cmd/testdata/manifest.golden.json` - new `pipeline watch` leaf + `PipelineWatchResult` type.
- `README.md`, `llms.txt`, `CLAUDE.md` - docs.

---

## Task 1: Slice #2 - repository UUID

**Files:** `pkg/bitbucket/types.go`, `pkg/bitbucket/repo_test.go`, `cmd/render/repo.go`.

- [ ] **Step 1: Failing decode test** in `repo_test.go`:

```go
func TestRepos_GetDecodesUUID(t *testing.T) {
	repo := bitbucket.Repo{Slug: "app", UUID: "{repo-uuid-1}"}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mustEncodeJSON(t, w, repo)
	}))
	got, err := client.Repos("testws").Get(context.Background(), "app")
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{repo-uuid-1}" {
		t.Errorf("expected uuid {repo-uuid-1}, got %q", got.UUID)
	}
}
```

- [ ] **Step 2: Run -> FAIL** (`Repo` has no `UUID` field): `go test ./pkg/bitbucket/ -run TestRepos_GetDecodesUUID`
- [ ] **Step 3:** Add `UUID string \`json:"uuid"\`` to `Repo` in `types.go` (after `Slug`):

```go
type Repo struct {
	Slug        string      `json:"slug"`
	UUID        string      `json:"uuid"`
	Name        string      `json:"name"`
	...
}
```

- [ ] **Step 4:** Render it in `RepoDetailString` (after the Slug line):

```go
	sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("Slug:       "), IDStyle.Render(r.Slug)))
	if r.UUID != "" {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render("UUID:       "), DimStyle.Render(r.UUID)))
	}
```

- [ ] **Step 5: Run** `go test ./... ` -> PASS. Commit:

```
feat(repo): surface repository uuid in repo get

The Bitbucket API already returns the repository uuid; the Repo struct
silently dropped it. Decode it and render it in repo get detail output.

Co-Authored-By: ...
```

---

## Task 2: Slice #3 client - types + classifier

**Files:** `pkg/bitbucket/types.go`, `pkg/bitbucket/pipeline.go`, `pkg/bitbucket/pipeline_internal_test.go`.

- [ ] **Step 1:** Add types to `types.go` (near the Pipeline types):

```go
// PipelineWatchStatus is the terminal outcome of `pipeline watch`.
type PipelineWatchStatus string

const (
	WatchSuccess PipelineWatchStatus = "success"
	WatchFailed  PipelineWatchStatus = "failed"
	WatchBlocked PipelineWatchStatus = "blocked"
	WatchTimeout PipelineWatchStatus = "timeout"
)

// ManualGate identifies the step a pipeline is paused on and the web URL where
// it can be resumed (the public API cannot resume it - see BCLOUD-20050).
type ManualGate struct {
	Step string `json:"step"`
	URL  string `json:"url"`
}

// PipelineWatchResult is the single object `pipeline watch` emits on stop.
type PipelineWatchResult struct {
	Pipeline   Pipeline       `json:"pipeline"`
	Steps      []PipelineStep `json:"steps"`
	Status     PipelineWatchStatus `json:"status"`
	ManualGate *ManualGate    `json:"manual_gate,omitempty"`
}
```

- [ ] **Step 2:** Internal classifier table test `pipeline_internal_test.go` (package `bitbucket`) covering: COMPLETED+SUCCESSFUL=success/terminal; COMPLETED+FAILED=failed/terminal; COMPLETED with nil result=failed/terminal; IN_PROGRESS+PAUSED=blocked/terminal with gate step; IN_PROGRESS+HALTED=blocked; IN_PROGRESS+RUNNING=not terminal; PENDING=not terminal. Assert `(status, gateStep, terminal)`.

- [ ] **Step 3: Run -> FAIL** (undefined `classifyPipelineState`).

- [ ] **Step 4:** Implement in `pipeline.go`:

```go
// classifyPipelineState returns the terminal watch status for a pipeline and
// whether it has reached a terminal state. gateStep is the step the pipeline is
// blocked on when the status is blocked (best-effort; may be "").
func classifyPipelineState(p Pipeline, steps []PipelineStep) (status PipelineWatchStatus, gateStep string, terminal bool) {
	switch p.State.Name {
	case "COMPLETED":
		if p.State.Result != nil && p.State.Result.Name == "SUCCESSFUL" {
			return WatchSuccess, "", true
		}
		return WatchFailed, "", true
	case "IN_PROGRESS":
		if p.State.Stage != nil {
			switch p.State.Stage.Name {
			case "PAUSED", "HALTED":
				return WatchBlocked, firstIncompleteStep(steps), true
			}
		}
	}
	return "", "", false
}

// firstIncompleteStep returns the name of the first step not yet COMPLETED, or
// "" if every step is complete (or there are none).
func firstIncompleteStep(steps []PipelineStep) string {
	for _, s := range steps {
		if s.State.Name != "COMPLETED" {
			return s.Name
		}
	}
	return ""
}
```

- [ ] **Step 5: Run -> PASS.** Commit types + classifier.

---

## Task 3: Slice #3 client - Latest + Watch

**Files:** `pkg/bitbucket/pipeline.go`, `pkg/bitbucket/pipeline_test.go`. Add `net/http` + `time` imports.

- [ ] **Step 1:** `Latest` tests (external): no-branch returns first of `-created_on` list; branch mode filters `target.ref_name` and returns first match; empty result returns an `*APIError` with `Status == 404`.

- [ ] **Step 2:** `Watch` tests (external, httptest with advancing state, `Interval: time.Millisecond`): poll-to-success (state flips IN_PROGRESS->COMPLETED/SUCCESSFUL across polls => `WatchSuccess`); poll-to-failure (=> `WatchFailed`); blocked (handler returns IN_PROGRESS/PAUSED => `WatchBlocked` with non-nil `ManualGate` whose URL ends `/pipelines/results/<n>`); timeout (handler stays IN_PROGRESS/RUNNING, `Timeout: 15ms` => `WatchTimeout`).

- [ ] **Step 3: Run -> FAIL.**

- [ ] **Step 4:** Implement:

```go
// Latest returns the most recent pipeline in the repo, or the most recent on
// branch when branch != "". Filters the -created_on list client-side; returns a
// 404 *APIError when no pipeline matches.
func (r *PipelineResource) Latest(ctx context.Context, branch string) (Pipeline, error) {
	pipelines, err := r.List(ctx, "-created_on")
	if err != nil {
		return Pipeline{}, err
	}
	for _, p := range pipelines {
		if branch == "" || p.Target.RefName == branch {
			return p, nil
		}
	}
	msg := "no pipelines found"
	if branch != "" {
		msg = fmt.Sprintf("no pipelines found for branch %q", branch)
	}
	return Pipeline{}, &APIError{Status: http.StatusNotFound, Message: msg}
}

// WatchOptions configures Watch. Interval defaults to 5s when <= 0; Timeout <= 0
// means no timeout. OnPoll, when set, is called with each poll's pipeline+steps.
type WatchOptions struct {
	Interval time.Duration
	Timeout  time.Duration
	OnPoll   func(Pipeline, []PipelineStep)
}

// Watch polls a pipeline until it reaches a terminal state (completed, blocked
// on a manual gate) or the timeout elapses, returning the classified result.
func (r *PipelineResource) Watch(ctx context.Context, pipelineUUID string, opts WatchOptions) (PipelineWatchResult, error) {
	interval := opts.Interval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	start := time.Now()
	for {
		p, err := r.Get(ctx, pipelineUUID)
		if err != nil {
			return PipelineWatchResult{}, err
		}
		steps, err := r.Steps(ctx, pipelineUUID)
		if err != nil {
			return PipelineWatchResult{}, err
		}
		if opts.OnPoll != nil {
			opts.OnPoll(p, steps)
		}
		if status, gateStep, terminal := classifyPipelineState(p, steps); terminal {
			result := PipelineWatchResult{Pipeline: p, Steps: steps, Status: status}
			if status == WatchBlocked {
				result.ManualGate = &ManualGate{Step: gateStep, URL: r.pipelineWebURL(p.BuildNumber)}
			}
			return result, nil
		}
		if opts.Timeout > 0 && time.Since(start) >= opts.Timeout {
			return PipelineWatchResult{Pipeline: p, Steps: steps, Status: WatchTimeout}, nil
		}
		select {
		case <-ctx.Done():
			return PipelineWatchResult{}, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// pipelineWebURL builds the Bitbucket web URL for a pipeline result page.
func (r *PipelineResource) pipelineWebURL(buildNumber int) string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s/pipelines/results/%d", r.workspace, r.repo, buildNumber)
}
```

- [ ] **Step 5: Run -> PASS.** Commit Latest + Watch.

---

## Task 4: Slice #3 cmd - exit code + watch command

**Files:** `cmd/root.go`, `cmd/pipeline.go`, `cmd/render/pipeline.go`, `cmd/pipeline_test.go`.

- [ ] **Step 1:** In `cmd/root.go` add `var exitCode int` (package level) and make `Execute` honor it after a successful run:

```go
func Execute() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	if err := rootCmd.Execute(); err != nil {
		emitError(mapError(err))
		os.Exit(1)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
```

- [ ] **Step 2:** Extend `pipelineSelector` with `branch string`; add pure helpers + watch resolver in `cmd/pipeline.go`:

```go
// watchExitCode maps a terminal watch status to the process exit code that lets
// `bb pipeline watch ... && next` proceed only on success.
func watchExitCode(status bitbucket.PipelineWatchStatus) int {
	switch status {
	case bitbucket.WatchSuccess:
		return 0
	case bitbucket.WatchFailed:
		return 1
	case bitbucket.WatchBlocked:
		return 2
	case bitbucket.WatchTimeout:
		return 3
	}
	return 0
}

// validateWatchSelector allows at most one of uuid/build/branch (none => latest).
func (s pipelineSelector) validateWatchSelector() error {
	set := 0
	if s.uuid != "" {
		set++
	}
	if s.build > 0 {
		set++
	}
	if s.branch != "" {
		set++
	}
	if set > 1 {
		return newCLIError(ErrCodeValidationFailed,
			"--pipeline-uuid, --build-number, and --branch are mutually exclusive", nil)
	}
	return nil
}

// resolveWatchPipeline picks the pipeline to watch: build/uuid address one
// directly; otherwise the latest pipeline (on branch, if set).
func (s pipelineSelector) resolveWatchPipeline(ctx context.Context, res *bitbucket.PipelineResource) (bitbucket.Pipeline, error) {
	if err := s.validateWatchSelector(); err != nil {
		return bitbucket.Pipeline{}, err
	}
	switch {
	case s.build > 0:
		return res.GetByBuildNumber(ctx, s.build)
	case s.uuid != "":
		return res.Get(ctx, s.uuid)
	default:
		return res.Latest(ctx, s.branch)
	}
}
```

- [ ] **Step 3:** Add the `watch` command + flag vars in `cmd/pipeline.go`. Progress + tail-log via an `OnPoll` closure that writes to stderr only when stderr is a TTY (`term.IsTerminal(int(os.Stderr.Fd()))`); tail-log fetches the running step's log and prints only the newly-appended tail (best-effort, ignore fetch errors). RunE resolves the target, calls `res.Watch`, sets `exitCode = watchExitCode(result.Status)`, and prints via `printOutput`.

- [ ] **Step 4:** `render.PipelineWatch(result)` in `cmd/render/pipeline.go`: one summary line (`#<n>  <STATUS>`), and when blocked, the gate step + resume URL.

- [ ] **Step 5:** Register flags + command in `init()`: `-n/--build-number`, `-u/--pipeline-uuid`, `-b/--branch`, `--tail-log`, `--interval` (int, default 5), `--timeout` (int, default 0); `pipelineCmd.AddCommand(pipelineWatchCmd)`.

- [ ] **Step 6:** Pure-logic tests in `cmd/pipeline_test.go`: `watchExitCode` (all four statuses + default) and `validateWatchSelector` (none ok, each-single ok, any-two errors validation_failed).

- [ ] **Step 7:** `go build ./... && go vet ./... && go test ./...` -> green. Commit cmd watch.

---

## Task 5: Manifest + docs

- [ ] **Step 1:** `cmd/manifest_registry.go`: add `"pipeline watch": {Action: actionRead, OutputType: "PipelineWatchResult", Example: "bb pipeline watch -n 42"}` and `"PipelineWatchResult": bitbucket.PipelineWatchResult{}` to `typeRegistry`.
- [ ] **Step 2:** `go test ./cmd/ -run TestManifestSnapshot` -> FAIL; `go test ./cmd/ -update`; `go test ./cmd/` -> PASS (`TestEveryLeafIsRegistered` green).
- [ ] **Step 3:** README: add to Pipelines block `bb pipeline watch [-n BUILD | -u UUID | -b BRANCH] [--tail-log] [--interval 5] [--timeout 0]` and a short note on exit codes (0/1/2/3) and the repo uuid in `repo get`. Add `-b`/`--branch` already listed; ensure `--tail-log/--interval/--timeout` mentioned.
- [ ] **Step 4:** llms.txt: mirror the README watch line + exit-code note.
- [ ] **Step 5:** CLAUDE.md: command hierarchy `pipeline list / get / trigger / stop / steps / log` -> append `/ watch`.
- [ ] **Step 6:** `go test ./...` green; commit docs + manifest.

---

## Self-Review

- #2 repo uuid: Task 1 (decode + render + test). Covered.
- #3 client `Latest` (branch + no-branch + not_found), `Watch` (success/failure/blocked/timeout), classifier (pure test): Tasks 2-3. Covered.
- Manual-gate detection + deep-link URL: `classifyPipelineState` + `pipelineWebURL` + `ManualGate`. Covered.
- Exit codes 0/1/2/3 via `exitCode`+`Execute`: Task 4. `watchExitCode` unit-tested.
- Selector (at-most-one, none=latest): `validateWatchSelector`/`resolveWatchPipeline`, unit-tested. Covered.
- Non-TTY single-JSON contract: `printOutput(result, ...)`; progress/tail-log to stderr gated on TTY. Covered.
- Manifest new leaf + type; docs (README/llms/CLAUDE): Task 5. Covered.
- Types consistent: `PipelineWatchResult`, `PipelineWatchStatus`, `WatchOptions`, `classifyPipelineState(p, steps) (status, gateStep, terminal)` used identically across tasks.

## Judgement calls / deferred

- **TTY "live view" simplified** to a per-poll status line on stderr (not a repainting TUI). Keeps stdout clean for the final JSON and avoids a fragile, hard-to-test render loop. Documented.
- **`--tail-log` is best-effort:** streams the running step's newly-appended log to stderr, ignoring transient fetch errors (logs 404 before a step starts). cmd-layer, untested per convention.
- **Manual-gate detection validated against Atlassian docs, not a live paused pipeline.** The classifier is pure and unit-tested; the exact PAUSED/HALTED strings are documented values. Flag for live confirmation against a real paused pipeline before relying on the `blocked` exit code in automation.
- **`Latest` branch filter is client-side** over the most-recent page (per spec). A branch whose latest pipeline is older than a full page of newer pipelines on other branches could be missed; acceptable for "latest", noted.
