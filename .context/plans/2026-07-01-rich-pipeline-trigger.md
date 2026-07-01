# Rich `pipeline trigger` Implementation Plan (backlog #5)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Extend `bb pipeline trigger` beyond branch-only to run custom/named pipelines, target a tag or commit, and pass per-run variables.

**Architecture:** Replace the client `Trigger(ctx, branch)` with `Trigger(ctx, TriggerOptions)` that builds the richer `POST pipelines/` body (ref/commit target + optional custom selector + variables). The `cmd` layer parses/validates flags into `TriggerOptions`.

**Tech Stack:** Go 1.26, Cobra, stdlib `net/http/httptest`. No new deps.

## Global Constraints

- **Scope:** `--branch`/`--tag`/`--commit` (exactly one), `--custom NAME`, `--var K=V` (repeatable, unsecured). **No `--env-uuid`** (verified: not a real trigger-body field; env-targeted deploys use `--custom`). **No `--secure-var`** (avoid argv/shell-history secret exposure; unsecured only).
- **Confirmed API body shapes** (Atlassian blog + API ref + community): ref target `{type:"pipeline_ref_target", ref_type:"branch"|"tag", ref_name}`; commit target `{type:"pipeline_commit_target", commit:{type:"commit", hash}}`; custom pipeline `selector:{type:"custom", pattern}`; top-level `variables:[{key,value,secured?}]`.
- **Tests** live in `pkg/bitbucket/`; pure `cmd` helpers (`triggerRef`, `parseTriggerVars`) get table tests in `cmd/`; Cobra wiring untested.
- **Write operation:** no live trigger test without explicit user sign-off (it starts a real pipeline).
- **No em-dashes; ASCII only. End commits with:** `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`. **Do not push.**
- **Docs sync:** README, llms.txt (CLAUDE.md hierarchy unchanged - `trigger` is an existing leaf; the client-pattern block does not list `Trigger`).

---

## Task 1: Client - types + rich Trigger

**Files:** `pkg/bitbucket/types.go`, `pkg/bitbucket/pipeline.go`, `pkg/bitbucket/pipeline_test.go`.

- [ ] **Step 1:** Extend the wire types in `types.go` (replace the current `TriggerPipelineInput`/`TriggerTarget`):

```go
type TriggerPipelineInput struct {
	Target    TriggerTarget     `json:"target"`
	Variables []TriggerVariable `json:"variables,omitempty"`
}

type TriggerTarget struct {
	Type     string           `json:"type"`               // pipeline_ref_target | pipeline_commit_target
	RefType  string           `json:"ref_type,omitempty"` // branch | tag (ref targets)
	RefName  string           `json:"ref_name,omitempty"` // ref targets
	Commit   *TriggerCommit   `json:"commit,omitempty"`   // commit targets
	Selector *TriggerSelector `json:"selector,omitempty"` // custom pipeline
}

type TriggerCommit struct {
	Type string `json:"type"` // always "commit"
	Hash string `json:"hash"`
}

type TriggerSelector struct {
	Type    string `json:"type"` // always "custom"
	Pattern string `json:"pattern"`
}

type TriggerVariable struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured,omitempty"`
}

// TriggerRef selects what a pipeline runs against: exactly one of Branch/Tag/Commit.
type TriggerRef struct {
	Branch string
	Tag    string
	Commit string
}

// TriggerOptions is the ergonomic input to Trigger.
type TriggerOptions struct {
	Ref       TriggerRef
	Custom    string
	Variables []TriggerVariable
}
```

Adding `omitempty` to `RefType`/`RefName` is required so a commit target does not send empty ref fields.

- [ ] **Step 2:** Failing tests in `pipeline_test.go` - rewrite `TestPipelines_Trigger` to the options form and add cases:
  - branch (existing assertions, via `TriggerOptions{Ref: TriggerRef{Branch:"main"}}`): body `target.type==pipeline_ref_target`, `ref_type==branch`, `ref_name==main`, no `selector`, no `variables`.
  - tag: `ref_type==tag`, `ref_name==v1.0`.
  - commit: `target.type==pipeline_commit_target`, `commit.type==commit`, `commit.hash==<sha>`, no `ref_type`.
  - custom + vars: `selector.type==custom`, `selector.pattern==deploy`, `variables[0]=={key:FOO,value:bar}` (no `secured` key when false).

- [ ] **Step 3: Run -> FAIL** (Trigger signature changed / new fields).

- [ ] **Step 4:** Implement in `pipeline.go` (replace existing `Trigger`):

```go
// Trigger starts a new pipeline. opts.Ref selects the target (exactly one of
// Branch/Tag/Commit; the cmd layer validates this). opts.Custom runs a named
// custom pipeline; opts.Variables are per-run variables.
func (r *PipelineResource) Trigger(ctx context.Context, opts TriggerOptions) (Pipeline, error) {
	target, err := buildTriggerTarget(opts)
	if err != nil {
		return Pipeline{}, err
	}
	input := TriggerPipelineInput{Target: target, Variables: opts.Variables}
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Pipeline{}, err
	}
	return decode[Pipeline](data)
}

// buildTriggerTarget maps a TriggerOptions to the API target body. It expects
// exactly one ref to be set (guaranteed by the cmd layer) and returns an error
// otherwise so a misuse is not silently sent as an empty target.
func buildTriggerTarget(opts TriggerOptions) (TriggerTarget, error) {
	var t TriggerTarget
	switch {
	case opts.Ref.Commit != "":
		t.Type = "pipeline_commit_target"
		t.Commit = &TriggerCommit{Type: "commit", Hash: opts.Ref.Commit}
	case opts.Ref.Branch != "":
		t.Type = "pipeline_ref_target"
		t.RefType = "branch"
		t.RefName = opts.Ref.Branch
	case opts.Ref.Tag != "":
		t.Type = "pipeline_ref_target"
		t.RefType = "tag"
		t.RefName = opts.Ref.Tag
	default:
		return TriggerTarget{}, fmt.Errorf("trigger: no ref selected (branch, tag, or commit required)")
	}
	if opts.Custom != "" {
		t.Selector = &TriggerSelector{Type: "custom", Pattern: opts.Custom}
	}
	return t, nil
}
```

- [ ] **Step 5: Run -> PASS.** Commit client.

---

## Task 2: cmd - flags, validation, wiring

**Files:** `cmd/pipeline.go`, `cmd/pipeline_test.go`.

- [ ] **Step 1:** Failing pure-helper tests in `cmd/pipeline_test.go`:
  - `triggerRef`: none -> validation_failed; each single -> ok with the right field; any two -> validation_failed.
  - `parseTriggerVars`: `["A=1","B=x=y"]` -> `[{A,1},{B,x=y}]` (split on first `=`); `["bad"]` and `["=v"]` -> validation_failed; empty slice -> nil.

- [ ] **Step 2: Run -> FAIL** (undefined helpers).

- [ ] **Step 3:** Implement helpers + rewrite the trigger command in `cmd/pipeline.go` (needs `strings` import):

```go
// triggerRef validates that exactly one ref selector is set and returns it.
func triggerRef(branch, tag, commit string) (bitbucket.TriggerRef, error) {
	set := 0
	for _, v := range []string{branch, tag, commit} {
		if v != "" {
			set++
		}
	}
	switch {
	case set == 0:
		return bitbucket.TriggerRef{}, newCLIError(ErrCodeValidationFailed,
			"one of --branch, --tag, or --commit is required", nil)
	case set > 1:
		return bitbucket.TriggerRef{}, newCLIError(ErrCodeValidationFailed,
			"--branch, --tag, and --commit are mutually exclusive", nil)
	}
	return bitbucket.TriggerRef{Branch: branch, Tag: tag, Commit: commit}, nil
}

// parseTriggerVars parses repeated KEY=VALUE flags into unsecured per-run
// pipeline variables. VALUE may contain '='; KEY may not be empty.
func parseTriggerVars(pairs []string) ([]bitbucket.TriggerVariable, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	vars := make([]bitbucket.TriggerVariable, 0, len(pairs))
	for _, p := range pairs {
		key, value, ok := strings.Cut(p, "=")
		if !ok || key == "" {
			return nil, newCLIError(ErrCodeValidationFailed,
				fmt.Sprintf("invalid --var %q: expected KEY=VALUE", p), nil)
		}
		vars = append(vars, bitbucket.TriggerVariable{Key: key, Value: value})
	}
	return vars, nil
}
```

Trigger command RunE:

```go
var (
	pipelineTriggerBranch string
	pipelineTriggerTag    string
	pipelineTriggerCommit string
	pipelineTriggerCustom string
	pipelineTriggerVars   []string
)

var pipelineTriggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Trigger a new pipeline (branch/tag/commit, optional custom pipeline + variables)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		ref, err := triggerRef(pipelineTriggerBranch, pipelineTriggerTag, pipelineTriggerCommit)
		if err != nil {
			return err
		}
		vars, err := parseTriggerVars(pipelineTriggerVars)
		if err != nil {
			return err
		}
		p, err := client.Pipelines(ws, repo).Trigger(context.Background(), bitbucket.TriggerOptions{
			Ref:       ref,
			Custom:    pipelineTriggerCustom,
			Variables: vars,
		})
		if err != nil {
			return err
		}
		return printOutput(p, func() {
			fmt.Printf("Pipeline #%d triggered.\nUUID: %s\n", p.BuildNumber, p.UUID)
		})
	},
}
```

- [ ] **Step 4:** Update `init()` flag registration - drop `MarkFlagRequired("branch")`, add the new flags:

```go
	pipelineTriggerCmd.Flags().StringVarP(&pipelineTriggerBranch, "branch", "b", "", "branch to run the pipeline on")
	pipelineTriggerCmd.Flags().StringVar(&pipelineTriggerTag, "tag", "", "tag to run the pipeline on")
	pipelineTriggerCmd.Flags().StringVar(&pipelineTriggerCommit, "commit", "", "commit hash to run the pipeline on")
	pipelineTriggerCmd.Flags().StringVar(&pipelineTriggerCustom, "custom", "", "name of a custom pipeline to run")
	pipelineTriggerCmd.Flags().StringArrayVar(&pipelineTriggerVars, "var", nil, "per-run variable KEY=VALUE (repeatable)")
```

(Exactly one of branch/tag/commit is validated in RunE, replacing the required-flag marker.)

- [ ] **Step 5:** `go build ./... && go vet ./... && go test ./...` -> green. Commit cmd.

---

## Task 3: Manifest + docs

- [ ] **Step 1:** `trigger` is an existing leaf; only its flags changed (auto-collected). Regenerate golden: `go test ./cmd/ -run TestManifestSnapshot` (FAIL) -> `go test ./cmd/ -update` -> `go test ./cmd/` (PASS). Keep the registry Example `bb pipeline trigger --branch main` (still valid); optionally not changed.
- [ ] **Step 2:** README Pipelines block: replace `bb pipeline trigger -b BRANCH` with `bb pipeline trigger (-b BRANCH | --tag TAG | --commit SHA) [--custom NAME] [--var K=V ...]`; add a one-line note that env-targeted deploys use `--custom`. Flag table: `--tag`/`--commit`/`--custom`/`--var` are long-only (no new short rows needed; `-b` already listed).
- [ ] **Step 3:** llms.txt: mirror the trigger line.
- [ ] **Step 4:** `go test ./...` green; commit docs + manifest.

---

## Self-Review

- Rich trigger (branch/tag/commit + custom + vars): Tasks 1-2. Covered.
- Exactly-one ref, KEY=VALUE parsing: `triggerRef`/`parseTriggerVars`, unit-tested. Covered.
- API body shapes asserted at the client layer (each target variant + selector + variables): Task 1 Step 2. Covered.
- No `--env-uuid`, no `--secure-var` (per decisions). Covered.
- Manifest golden + docs: Task 3. Covered.
- Types consistent: `TriggerOptions{Ref TriggerRef, Custom string, Variables []TriggerVariable}` used identically in client and cmd.

## Judgement calls / notes

- **`--commit` without `--custom`** is allowed (the API supports running the default pipeline for a specific commit). Not pre-restricted; an unsupported combination surfaces as the API's own error.
- **Live write test deferred to user sign-off** - triggering starts a real pipeline. Unit tests assert the exact POST body bb constructs.
- **Backward compatible:** `bb pipeline trigger -b main` still works (now validated in RunE instead of via MarkFlagRequired).
