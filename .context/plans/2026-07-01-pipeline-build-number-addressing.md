# Pipeline build-number addressing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let `bb pipeline get/stop/steps/log` address a pipeline by its integer build number (`-n/--build-number`) as an alternative to the opaque braced `-u/--pipeline-uuid`.

**Architecture:** Add one client method, `PipelineResource.GetByBuildNumber`, that fetches a pipeline by the integer path form Bitbucket already supports. In the `cmd` layer, each of the four commands gains an `-n` flag; a shared validator enforces "exactly one of `-u`/`-n`", and a shared resolver turns a build number into a UUID (via `GetByBuildNumber`) so the existing UUID-based stop/steps/log methods are reused unchanged (DRY). `get` with `-n` returns `GetByBuildNumber` directly (no redundant fetch).

**Tech Stack:** Go 1.26, Cobra, stdlib `net/http/httptest` for client tests. No new dependencies.

## Global Constraints

- **Slice scope is item #1 only.** Repo UUID (#2) and `pipeline watch` (#3) are out of scope; do not implement them here.
- **`cmd/` is intentionally untested** (thin Cobra wiring). All new unit-test coverage lives in `pkg/bitbucket/pipeline_test.go`. The `cmd` layer is verified via `go build`, `go vet`, and the manifest snapshot test.
- **Exactly one of `-u`/`-n`** required for get/stop/steps/log. Both set, or neither, is a `validation_failed` `*CLIError`. A build number is "set" when `> 0` (Bitbucket build numbers start at 1; `0` is the flag zero value).
- **Bitbucket UUIDs keep their braces.** Do not strip `{}` from UUID paths (existing invariant; see `pipeline.go` Log doc comment). Build numbers are plain integers with no braces.
- **Documentation sync is required in the same change set** whenever a flag is added: update `README.md`, `llms.txt`, and `CLAUDE.md` (per project CLAUDE.md rule).
- **No em-dashes** anywhere (prose, comments, commit messages). Use `-`. ASCII punctuation only.
- **End every commit message** with: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **Do not push** to git/Bitbucket without explicit user confirmation.

---

## File Structure

- `pkg/bitbucket/pipeline.go` (modify) - add `GetByBuildNumber`.
- `pkg/bitbucket/pipeline_test.go` (modify) - add `TestPipelines_GetByBuildNumber`.
- `cmd/pipeline.go` (modify) - add `-n` flags + `validatePipelineSelector` + `pipelineTargetUUID`; rewrite the four RunE bodies; drop `MarkFlagRequired("pipeline-uuid")` (keep it for `--step-uuid` on `log`).
- `cmd/manifest_registry.go` (modify) - refresh the four `pipeline` example strings (they currently reference a nonexistent `--uuid` flag; switch to the new `-n` form).
- `cmd/testdata/manifest.golden.json` (regenerate) - via `go test ./cmd/ -update`.
- `README.md`, `llms.txt` (modify) - command reference + flag table.
- `CLAUDE.md` (verify; likely no change - its command tree lists no flags).

---

## Task 1: Client `GetByBuildNumber`

**Files:**
- Modify: `pkg/bitbucket/pipeline.go`
- Test: `pkg/bitbucket/pipeline_test.go`

**Interfaces:**
- Consumes: existing `PipelineResource.basePath()` (returns `.../pipelines/`), `Client.do`, generic `decode[T]`.
- Produces: `func (r *PipelineResource) GetByBuildNumber(ctx context.Context, buildNumber int) (Pipeline, error)` - GETs `/repositories/{ws}/{repo}/pipelines/{build_number}` and decodes a `Pipeline`. Consumed by `cmd/pipeline.go` in Task 2.

- [ ] **Step 1: Write the failing test**

Add to `pkg/bitbucket/pipeline_test.go` (imports `context`, `net/http`, `strings`, `testing`, and the `bitbucket` package are already present):

```go
func TestPipelines_GetByBuildNumber(t *testing.T) {
	pipeline := bitbucket.Pipeline{UUID: "{abc-123}", BuildNumber: 42}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		// A build number is addressed as a plain integer path segment, with no
		// braces - unlike the UUID form, which requires them.
		if !strings.HasSuffix(r.URL.Path, "/pipelines/42") {
			t.Errorf("expected path to end with /pipelines/42, got %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, pipeline)
	}))
	got, err := client.Pipelines("testws", "testrepo").GetByBuildNumber(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{abc-123}" {
		t.Errorf("expected uuid {abc-123}, got %s", got.UUID)
	}
	if got.BuildNumber != 42 {
		t.Errorf("expected build 42, got %d", got.BuildNumber)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestPipelines_GetByBuildNumber`
Expected: FAIL - compile error `got.GetByBuildNumber undefined` (method not yet defined).

- [ ] **Step 3: Write minimal implementation**

Add to `pkg/bitbucket/pipeline.go`, immediately after the `Get` method (around line 47):

```go
// GetByBuildNumber returns a single pipeline addressed by its integer build
// number - the "#42" agents already see in `pipeline list` output - rather
// than its UUID. Bitbucket Cloud's pipeline resource accepts either form:
//
//	GET /repositories/{ws}/{repo}/pipelines/{build_number}
//
// The build number is a plain integer path segment; unlike the UUID form it
// carries no surrounding braces.
func (r *PipelineResource) GetByBuildNumber(ctx context.Context, buildNumber int) (Pipeline, error) {
	path := fmt.Sprintf("%s%d", r.basePath(), buildNumber)
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Pipeline{}, err
	}
	return decode[Pipeline](data)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/bitbucket/ -run TestPipelines_GetByBuildNumber`
Expected: PASS.

- [ ] **Step 5: Run the full client package tests**

Run: `go test ./pkg/bitbucket/`
Expected: PASS (no regressions).

- [ ] **Step 6: Commit**

```bash
git add pkg/bitbucket/pipeline.go pkg/bitbucket/pipeline_test.go
git commit -m "feat(pipeline): add GetByBuildNumber client method

Fetch a pipeline by its integer build number via the plain-integer path
form (GET pipelines/{build_number}). Groundwork for -n/--build-number
addressing on the pipeline get/stop/steps/log commands.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: `cmd` wiring - `-n` flags, selector validation, UUID resolution

**Files:**
- Modify: `cmd/pipeline.go`
- Modify: `cmd/manifest_registry.go` (example strings for the four pipeline leaves)
- Regenerate: `cmd/testdata/manifest.golden.json`

**Interfaces:**
- Consumes: `bitbucket.PipelineResource.GetByBuildNumber` (Task 1); existing `client.Pipelines(ws, repo) *bitbucket.PipelineResource`, `workspaceAndRepo()`, `printOutput`, `newCLIError`, `ErrCodeValidationFailed`, `render.PipelineDetail/PipelineSteps`.
- Produces: two `cmd`-package helpers reused by the four commands:
  - `func validatePipelineSelector(uuid string, buildNumber int) error`
  - `func pipelineTargetUUID(ctx context.Context, res *bitbucket.PipelineResource, uuid string, buildNumber int) (string, error)`

- [ ] **Step 1: Add the `bitbucket` import to `cmd/pipeline.go`**

The current import block is:

```go
import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)
```

Replace it with:

```go
import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)
```

- [ ] **Step 2: Add the two helpers at the top of `cmd/pipeline.go`**

Insert immediately after the `pipelineCmd` var block (after line 15, before `pipelineListSort`):

```go
// validatePipelineSelector enforces that exactly one of the UUID / build-number
// selectors is set for the get/stop/steps/log commands. A build number counts
// as "set" when it is > 0: Bitbucket build numbers start at 1, so 0 is the
// flag's unset zero value.
func validatePipelineSelector(uuid string, buildNumber int) error {
	hasUUID := uuid != ""
	hasBuild := buildNumber > 0
	switch {
	case hasUUID && hasBuild:
		return newCLIError(ErrCodeValidationFailed,
			"--pipeline-uuid and --build-number are mutually exclusive; set exactly one", nil)
	case !hasUUID && !hasBuild:
		return newCLIError(ErrCodeValidationFailed,
			"one of --pipeline-uuid or --build-number is required", nil)
	}
	return nil
}

// pipelineTargetUUID validates the selector and resolves it to a pipeline UUID.
// When a build number is supplied it fetches the pipeline to obtain its UUID so
// the existing UUID-based stop/steps/log methods can be reused unchanged.
func pipelineTargetUUID(ctx context.Context, res *bitbucket.PipelineResource, uuid string, buildNumber int) (string, error) {
	if err := validatePipelineSelector(uuid, buildNumber); err != nil {
		return "", err
	}
	if uuid != "" {
		return uuid, nil
	}
	p, err := res.GetByBuildNumber(ctx, buildNumber)
	if err != nil {
		return "", err
	}
	return p.UUID, nil
}
```

- [ ] **Step 3: Rewrite the `get` command var + RunE**

Replace the `pipelineGetUUID` var and `pipelineGetCmd` (current lines 35-51) with:

```go
var (
	pipelineGetUUID  string
	pipelineGetBuild int
)

var pipelineGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get pipeline details",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := validatePipelineSelector(pipelineGetUUID, pipelineGetBuild); err != nil {
			return err
		}
		ctx := context.Background()
		res := client.Pipelines(ws, repo)
		var p bitbucket.Pipeline
		if pipelineGetBuild > 0 {
			p, err = res.GetByBuildNumber(ctx, pipelineGetBuild)
		} else {
			p, err = res.Get(ctx, pipelineGetUUID)
		}
		if err != nil {
			return err
		}
		return printOutput(p, func() { render.PipelineDetail(p) })
	},
}
```

- [ ] **Step 4: Rewrite the `stop` command var + RunE**

Replace the `pipelineStopUUID` var and `pipelineStopCmd` (current lines 74-91) with:

```go
var (
	pipelineStopUUID  string
	pipelineStopBuild int
)

var pipelineStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		ctx := context.Background()
		res := client.Pipelines(ws, repo)
		uuid, err := pipelineTargetUUID(ctx, res, pipelineStopUUID, pipelineStopBuild)
		if err != nil {
			return err
		}
		if err := res.Stop(ctx, uuid); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "stopped", "uuid": uuid}, func() {
			fmt.Printf("Pipeline %s stopped.\n", uuid)
		})
	},
}
```

- [ ] **Step 5: Rewrite the `steps` command var + RunE**

Replace the `pipelineStepsUUID` var and `pipelineStepsCmd` (current lines 93-109) with:

```go
var (
	pipelineStepsUUID  string
	pipelineStepsBuild int
)

var pipelineStepsCmd = &cobra.Command{
	Use:   "steps",
	Short: "List steps of a pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		ctx := context.Background()
		res := client.Pipelines(ws, repo)
		uuid, err := pipelineTargetUUID(ctx, res, pipelineStepsUUID, pipelineStepsBuild)
		if err != nil {
			return err
		}
		steps, err := res.Steps(ctx, uuid)
		if err != nil {
			return err
		}
		return printOutput(steps, func() { render.PipelineSteps(steps) })
	},
}
```

- [ ] **Step 6: Rewrite the `log` command vars + RunE**

Replace the `pipelineLog*` var block and `pipelineLogCmd` (current lines 111-131) with:

```go
var (
	pipelineLogPipelineUUID string
	pipelineLogBuild        int
	pipelineLogStepUUID     string
)

var pipelineLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Get log output for a pipeline step (always plain text)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		ctx := context.Background()
		res := client.Pipelines(ws, repo)
		uuid, err := pipelineTargetUUID(ctx, res, pipelineLogPipelineUUID, pipelineLogBuild)
		if err != nil {
			return err
		}
		log, err := res.Log(ctx, uuid, pipelineLogStepUUID)
		if err != nil {
			return err
		}
		fmt.Print(log)
		return nil
	},
}
```

- [ ] **Step 7: Rewrite `init()` flag registration**

Replace the body of `init()` (current lines 133-157) with:

```go
func init() {
	pipelineListCmd.Flags().StringVar(&pipelineListSort, "sort", "",
		"sort by Bitbucket field, prefix with - for descending (default -created_on)")

	pipelineGetCmd.Flags().StringVarP(&pipelineGetUUID, "pipeline-uuid", "u", "", "pipeline UUID (alternative to --build-number)")
	pipelineGetCmd.Flags().IntVarP(&pipelineGetBuild, "build-number", "n", 0, "pipeline build number (alternative to --pipeline-uuid)")

	pipelineTriggerCmd.Flags().StringVarP(&pipelineTriggerBranch, "branch", "b", "", "branch to trigger pipeline on (required)")
	pipelineTriggerCmd.MarkFlagRequired("branch")

	pipelineStopCmd.Flags().StringVarP(&pipelineStopUUID, "pipeline-uuid", "u", "", "pipeline UUID (alternative to --build-number)")
	pipelineStopCmd.Flags().IntVarP(&pipelineStopBuild, "build-number", "n", 0, "pipeline build number (alternative to --pipeline-uuid)")

	pipelineStepsCmd.Flags().StringVarP(&pipelineStepsUUID, "pipeline-uuid", "u", "", "pipeline UUID (alternative to --build-number)")
	pipelineStepsCmd.Flags().IntVarP(&pipelineStepsBuild, "build-number", "n", 0, "pipeline build number (alternative to --pipeline-uuid)")

	pipelineLogCmd.Flags().StringVarP(&pipelineLogPipelineUUID, "pipeline-uuid", "u", "", "pipeline UUID (alternative to --build-number)")
	pipelineLogCmd.Flags().IntVarP(&pipelineLogBuild, "build-number", "n", 0, "pipeline build number (alternative to --pipeline-uuid)")
	pipelineLogCmd.Flags().StringVar(&pipelineLogStepUUID, "step-uuid", "", "step UUID (required)")
	pipelineLogCmd.MarkFlagRequired("step-uuid")

	pipelineCmd.AddCommand(pipelineListCmd, pipelineGetCmd, pipelineTriggerCmd,
		pipelineStopCmd, pipelineStepsCmd, pipelineLogCmd)
	rootCmd.AddCommand(pipelineCmd)
}
```

Note: the four `MarkFlagRequired("pipeline-uuid")` calls are gone (selection is validated in RunE now); `trigger`'s `--branch` and `log`'s `--step-uuid` stay required via `MarkFlagRequired`.

- [ ] **Step 8: Build and vet the cmd layer**

Run: `go build ./... && go vet ./cmd/...`
Expected: no output, exit 0.

- [ ] **Step 9: Refresh the manifest example strings**

In `cmd/manifest_registry.go`, replace the four pipeline example lines (currently referencing a nonexistent `--uuid` flag) with the new `-n` form:

```go
	"pipeline list":    {Action: actionRead, OutputType: "[]Pipeline", Example: "bb pipeline list"},
	"pipeline get":     {Action: actionRead, OutputType: "Pipeline", Example: "bb pipeline get -n 42"},
	"pipeline trigger": {Action: actionWrite, OutputType: "Pipeline", Example: "bb pipeline trigger --branch main"},
	"pipeline stop":    {Action: actionDestructive, OutputType: "ResultMap", Example: "bb pipeline stop -n 42"},
	"pipeline steps":   {Action: actionRead, OutputType: "[]PipelineStep", Example: "bb pipeline steps -n 42"},
	"pipeline log":     {Action: actionRead, OutputType: "string", Example: "bb pipeline log -n 42 --step-uuid '{step}'"},
```

- [ ] **Step 10: Confirm the manifest snapshot currently fails (proves the change is captured)**

Run: `go test ./cmd/ -run TestManifestSnapshot`
Expected: FAIL - the golden lacks the new `build-number` flags and shows the old examples.

- [ ] **Step 11: Regenerate the golden manifest**

Run: `go test ./cmd/ -update`
Expected: PASS (regenerates `cmd/testdata/manifest.golden.json`).

- [ ] **Step 12: Run the full cmd test suite**

Run: `go test ./cmd/`
Expected: PASS - `TestManifestSnapshot`, `TestEveryLeafIsRegistered`, `TestRegistryReferencesOnlyRealCommands` all green.

- [ ] **Step 13: Run the entire test suite + fmt**

Run: `go test ./... && gofmt -l cmd/pipeline.go cmd/manifest_registry.go`
Expected: all tests PASS; `gofmt -l` prints nothing (files already formatted).

- [ ] **Step 14: Commit**

```bash
git add cmd/pipeline.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json
git commit -m "feat(pipeline): address get/stop/steps/log by build number (-n)

Add -n/--build-number as an alternative to -u/--pipeline-uuid on the
pipeline get/stop/steps/log commands. Exactly one selector is required,
validated in RunE (replacing MarkFlagRequired on pipeline-uuid). A build
number resolves to a UUID via GetByBuildNumber so the existing UUID-based
methods are reused. Manifest examples corrected (they referenced a
nonexistent --uuid flag) and the golden snapshot regenerated.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Documentation sync

**Files:**
- Modify: `README.md`
- Modify: `llms.txt`
- Verify: `CLAUDE.md`

**Interfaces:** None (docs only). Must match the flag shapes shipped in Task 2.

- [ ] **Step 1: Update the README command block**

In `README.md`, replace the Pipelines command block (lines 195-202) so get/stop/steps/log show the selector choice:

```
bb pipeline list [--sort FIELD]
bb pipeline get (-u UUID | -n BUILD)
bb pipeline trigger -b BRANCH
bb pipeline stop (-u UUID | -n BUILD)
bb pipeline steps (-u UUID | -n BUILD)
bb pipeline log (-u UUID | -n BUILD) --step-uuid UUID
```

- [ ] **Step 2: Update the README global flag table**

In `README.md`, find the flag-table row (line 147):

```
| `-u` | `--pipeline-uuid` | `pipeline get/stop/steps/log` |
```

Add a row directly beneath it:

```
| `-n` | `--build-number` | `pipeline get/stop/steps/log` |
```

- [ ] **Step 3: Update `llms.txt`**

In `llms.txt`, replace the pipeline lines (154-158) to match README:

```
bb pipeline get (-u UUID | -n BUILD)
bb pipeline trigger -b BRANCH
bb pipeline stop (-u UUID | -n BUILD)
bb pipeline steps (-u UUID | -n BUILD)
bb pipeline log (-u UUID | -n BUILD) --step-uuid UUID
```

(Preserve the surrounding lines, including `bb pipeline list` above and any text below.)

- [ ] **Step 4: Verify `CLAUDE.md` needs no change**

Run: `grep -n "pipeline" CLAUDE.md`
Expected: the command-hierarchy tree shows `pipeline list / get / trigger / stop / steps / log` with no per-flag detail, and the Client-pattern block does not enumerate `Get`/`GetByBuildNumber`. No structural command change occurred (same leaves), so CLAUDE.md requires no edit. If a `pipeline get -u` example is found in narrative text, update it to `(-u UUID | -n BUILD)`; otherwise leave CLAUDE.md untouched.

- [ ] **Step 5: Commit**

```bash
git add README.md llms.txt CLAUDE.md
git commit -m "docs(pipeline): document -n/--build-number on get/stop/steps/log

Sync README command block + flag table and llms.txt with the new
build-number selector. CLAUDE.md unchanged (its command tree carries no
per-flag detail).

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

(If Step 4 produced no `CLAUDE.md` change, drop `CLAUDE.md` from the `git add`.)

---

## Self-Review

**1. Spec coverage (item #1 only):**
- `-n/--build-number` on get/stop/steps/log: Task 2 Steps 3-7.
- Exactly one of `-u`/`-n`, else `validation_failed`; drop `MarkFlagRequired("pipeline-uuid")`, validate in RunE: Task 2 Steps 2, 7.
- Build number resolves to UUID for stop/steps/log via `GetByBuildNumber`; `get -n` returns it directly: Task 1 + Task 2 Steps 2-6.
- Manifest: get/stop/steps/log gain the `-n` flag but remain the same leaves; golden regenerated: Task 2 Steps 9-11. (No `commandRegistry`/`typeRegistry` structural change needed since no new leaf and no new output type.)
- Client test asserts constructed path (`/pipelines/42`) and decode: Task 1 Step 1.
- Docs sync: Task 3.

**2. Placeholder scan:** No TBD/TODO/"add validation"/"handle edge cases". Every code step shows complete code. OK.

**3. Type consistency:** `GetByBuildNumber(ctx, int) (Pipeline, error)` is defined identically in Task 1 and consumed with that signature in Task 2 (`validatePipelineSelector`, `pipelineTargetUUID`, and `get`'s RunE). Flag var names (`pipelineGetBuild`, `pipelineStopBuild`, `pipelineStepsBuild`, `pipelineLogBuild`) are consistent between their `var` blocks and `init()`. `res.Get`/`res.Stop`/`res.Steps`/`res.Log` signatures unchanged. OK.

## Deferred / to confirm during implementation

- **Live-API confirmation of the integer path form** (`GET pipelines/{build_number}`) is a spec "open item". The unit test asserts bb's constructed path regardless; a live smoke test (`bb pipeline get -n <n>` against a real repo) is a nice-to-have but is not a gate for this slice and needs real credentials + an existing pipeline.
  - **CONFIRMED LIVE 2026-07-01:** `bb pipeline get -n 59` against the live Bitbucket API returned the full pipeline (`build_number: 59`, COMPLETED/SUCCESSFUL), proving `GET pipelines/59` (plain integer path, no braces) is accepted. This also de-risks slice #3 (`pipeline watch` reuses `GetByBuildNumber` for its `-n` selector).
