# Backlog #9 — Round Out Coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add read/update coverage for deployments and repository pipeline variables — `deployment get`, `deployment list` filtering/sort, `pipeline-var get`, and `pipeline-var update`.

**Architecture:** Mirror the existing thin-client + thin-cmd pattern. New client methods on `DeploymentResource` and `PipelineVariableResource` (stdlib `net/http`, generic `decode[T]`); new Cobra leaves in `cmd/` that call the client and render via `cmd/render`. `pipeline-var update` keeps the client method a thin PUT of a full representation (`CreatePipelineVariableInput`); the cmd layer owns the "fetch current for defaults, require --value" UX so a secured variable's unreadable value is never accidentally blanked.

**Tech Stack:** Go 1.26, Cobra, stdlib `net/http`/`net/http/httptest`, `encoding/json`.

## Global Constraints

- Tests live only in `pkg/bitbucket/` via `newTestClient(t, handler)` + `mustEncodeJSON`. `cmd/` Cobra wiring is intentionally untested.
- Every new leaf command MUST get an entry in `commandRegistry` (`cmd/manifest_registry.go`) or `TestEveryLeafIsRegistered` fails. New output Go types (none expected here — all reuse existing types) would also need `typeRegistry`.
- After any leaf/flag/example change, regenerate the manifest golden: `go test ./cmd/ -update`. The snapshot strips schemas + examples.
- Documentation sync is REQUIRED in the same change: `README.md` (Commands block), `llms.txt`, and `CLAUDE.md` (Command hierarchy tree + Client pattern block if a client signature changed).
- `--format` default is `gcf`; `printOutput(v, textFn)` handles all three formats. Text renderers live in `cmd/render`.
- No em-dashes or non-ASCII typography anywhere (prose, comments, commit messages).
- Never push/tag without explicit user sign-off. Live write mutations need sign-off before any live test.

---

### Task 1: `deployment get --uuid` (client + render + cmd)

**Files:**
- Modify: `pkg/bitbucket/deployment.go`
- Modify: `pkg/bitbucket/deployment_test.go`
- Create render fn in: `cmd/render/infra.go`
- Modify: `cmd/deployment.go`
- Modify: `cmd/manifest_registry.go:92-93` (deployment section)
- Regen: `cmd/testdata/manifest.golden.json`
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Produces: `func (r *DeploymentResource) Get(ctx context.Context, uuid string) (Deployment, error)` — `GET repositories/{ws}/{repo}/deployments/{uuid}`, decodes a single `Deployment`.
- Produces: `func DeploymentDetailString(d bitbucket.Deployment) string` and `func DeploymentDetail(d bitbucket.Deployment)` in `cmd/render`.

- [ ] **Step 1: Write the failing client test**

Add to `pkg/bitbucket/deployment_test.go`:

```go
func TestDeployments_Get(t *testing.T) {
	dep := bitbucket.Deployment{
		UUID:        "{dep-1}",
		State:       bitbucket.DeploymentState{Name: "COMPLETED", Status: &bitbucket.DeploymentStatus{Name: "SUCCESSFUL"}},
		Environment: bitbucket.DeploymentEnvRef{UUID: "{env-prod}"},
		Deployable: bitbucket.Deployable{
			Commit:   &bitbucket.DeployableCommit{Hash: "abc123"},
			Pipeline: &bitbucket.DeployablePipeline{UUID: "{pipe-1}"},
		},
		LastUpdateTime: "2024-01-15T10:00:00+00:00",
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/deployments/{dep-1}" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, dep)
	}))
	got, err := client.Deployments("testws", "testrepo").Get(context.Background(), "{dep-1}")
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{dep-1}" || got.State.Status == nil || got.State.Status.Name != "SUCCESSFUL" {
		t.Errorf("unexpected deployment: %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestDeployments_Get`
Expected: FAIL — `client.Deployments(...).Get` undefined.

- [ ] **Step 3: Implement the client method**

Add to `pkg/bitbucket/deployment.go`:

```go
// Get returns a single deployment by UUID.
func (r *DeploymentResource) Get(ctx context.Context, uuid string) (Deployment, error) {
	data, err := r.client.do(ctx, "GET", r.basePath()+uuid, nil, nil)
	if err != nil {
		return Deployment{}, err
	}
	return decode[Deployment](data)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/bitbucket/ -run TestDeployments_Get`
Expected: PASS.

- [ ] **Step 5: Add the render detail function**

In `cmd/render/infra.go`, add a `DeploymentDetailString`/`DeploymentDetail` pair modeled on `DeploymentListString` (read that function first to reuse its field access — `d.State.Name`, `d.State.Status`, `d.Environment.UUID`, `d.Deployable.Commit.Hash`, `d.Deployable.Pipeline.UUID`, `d.LastUpdateTime`). Nil-guard `State.Status`, `Deployable.Commit`, `Deployable.Pipeline`. Example shape:

```go
func DeploymentDetailString(d bitbucket.Deployment) string {
	var b strings.Builder
	fmt.Fprintf(&b, "UUID:        %s\n", d.UUID)
	state := d.State.Name
	if d.State.Status != nil {
		state = fmt.Sprintf("%s (%s)", d.State.Name, d.State.Status.Name)
	}
	fmt.Fprintf(&b, "State:       %s\n", state)
	fmt.Fprintf(&b, "Environment: %s\n", d.Environment.UUID)
	if d.Deployable.Commit != nil {
		fmt.Fprintf(&b, "Commit:      %s\n", d.Deployable.Commit.Hash)
	}
	if d.Deployable.Pipeline != nil {
		fmt.Fprintf(&b, "Pipeline:    %s\n", d.Deployable.Pipeline.UUID)
	}
	fmt.Fprintf(&b, "Updated:     %s\n", d.LastUpdateTime)
	return b.String()
}

func DeploymentDetail(d bitbucket.Deployment) { fmt.Print(DeploymentDetailString(d)) }
```

Match the actual imports/helpers already in `infra.go` (it may use a shared writer or label style — follow what `DeploymentListString` does).

- [ ] **Step 6: Add the cmd leaf**

In `cmd/deployment.go`, add a `deployment get` command with a required `--uuid` flag:

```go
var deploymentGetUUID string

var deploymentGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a deployment by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		dep, err := client.Deployments(ws, repo).Get(context.Background(), deploymentGetUUID)
		if err != nil {
			return err
		}
		return printOutput(dep, func() { render.DeploymentDetail(dep) })
	},
}
```

Wire it in `init()`:

```go
deploymentGetCmd.Flags().StringVar(&deploymentGetUUID, "uuid", "", "deployment UUID (required)")
deploymentGetCmd.MarkFlagRequired("uuid")
deploymentCmd.AddCommand(deploymentListCmd, deploymentGetCmd)
```

- [ ] **Step 7: Register in the manifest and regen golden**

Add to `cmd/manifest_registry.go` deployment section:

```go
"deployment get": {Action: actionRead, OutputType: "Deployment", Example: "bb deployment get --uuid '{uuid}'"},
```

Then: `go test ./cmd/ -update` and confirm `cmd/testdata/manifest.golden.json` now contains `deployment get`.

- [ ] **Step 8: Sync docs**

Add `bb deployment get --uuid UUID` to the README Commands block, `llms.txt`, and the `CLAUDE.md` command hierarchy (`deployment list` line becomes `deployment list / get`).

- [ ] **Step 9: Run full verification**

Run: `go build -o bb . && go test ./... && go vet ./... && gofmt -l cmd pkg internal`
Expected: build ok, all tests pass, vet clean, gofmt lists nothing (ignore pre-existing CRLF noise on `main.go`/`cmd/render/markdown.go` if it appears — do not reformat those).

- [ ] **Step 10: Commit**

```bash
git add pkg/bitbucket/deployment.go pkg/bitbucket/deployment_test.go cmd/render/infra.go cmd/deployment.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(deployment): add deployment get by uuid"
```

---

### Task 2: `deployment list --env-uuid --sort`

**Files:**
- Modify: `pkg/bitbucket/deployment.go`
- Modify: `pkg/bitbucket/deployment_test.go`
- Modify: `cmd/deployment.go`
- Modify: `cmd/manifest_registry.go` (deployment list example may gain flags — optional)
- Regen: `cmd/testdata/manifest.golden.json` (if example/flags change)
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Consumes: existing `basePath()`, `pagelenSmall`, `bbqlQuote` (unexported, same package), `decode[paged[Deployment]]`.
- Produces: `type DeploymentListOptions struct { EnvUUID string; Sort string }` and changes `List(ctx)` to `List(ctx context.Context, opts DeploymentListOptions) ([]Deployment, error)`.

**VERIFY FIRST (read-only, safe):** Before relying on the `q=environment.uuid=...` filter, confirm the deployments endpoint honors it against the live API (or Bitbucket REST docs). Build with a real `bb deployment list --env-uuid <uuid>` against a repo with environments and confirm it filters rather than 400s. If the endpoint does NOT support `q=`, fall back to unfiltered list + client-side filter on `d.Environment.UUID` and note it in the code comment.

- [ ] **Step 1: Update the existing List test + add filter/sort test**

The current `TestDeployments_List` calls `List(context.Background())`. Update that call to `List(context.Background(), bitbucket.DeploymentListOptions{})` (empty opts → same behavior; still asserts `pagelen=25`, no `q`, no `sort`). Then add:

```go
func TestDeployments_List_EnvFilterAndSort(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("q"); got != `environment.uuid="{env-prod}"` {
			t.Errorf("unexpected q: %q", got)
		}
		if got := q.Get("sort"); got != "-last_update_time" {
			t.Errorf("unexpected sort: %q", got)
		}
		mustEncodeJSON(t, w, map[string]any{"values": []bitbucket.Deployment{}})
	}))
	_, err := client.Deployments("testws", "testrepo").List(context.Background(), bitbucket.DeploymentListOptions{
		EnvUUID: "{env-prod}",
		Sort:    "-last_update_time",
	})
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/bitbucket/ -run TestDeployments_List`
Expected: FAIL — `List` signature mismatch / `DeploymentListOptions` undefined.

- [ ] **Step 3: Implement options + updated List**

In `pkg/bitbucket/deployment.go`:

```go
// DeploymentListOptions filters and orders a deployment listing.
type DeploymentListOptions struct {
	EnvUUID string // filter to a single environment UUID (empty = all)
	Sort    string // Bitbucket sort field, "-" prefix for descending (empty = default)
}

// List returns the most recent deployments, optionally filtered by environment
// and ordered by Sort.
func (r *DeploymentResource) List(ctx context.Context, opts DeploymentListOptions) ([]Deployment, error) {
	q := url.Values{"pagelen": {pagelenSmall}}
	if opts.EnvUUID != "" {
		q.Set("q", fmt.Sprintf(`environment.uuid=%s`, bbqlQuote(opts.EnvUUID)))
	}
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Deployment]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
```

- [ ] **Step 4: Update the cmd call site + add flags**

In `cmd/deployment.go`, add package vars `deploymentListEnvUUID`, `deploymentListSort`; change the List call to pass `bitbucket.DeploymentListOptions{EnvUUID: deploymentListEnvUUID, Sort: deploymentListSort}` (add the `bitbucket` import); register flags in `init()`:

```go
deploymentListCmd.Flags().StringVar(&deploymentListEnvUUID, "env-uuid", "", "filter to a single environment UUID")
deploymentListCmd.Flags().StringVar(&deploymentListSort, "sort", "", "sort field (e.g. -last_update_time)")
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./pkg/bitbucket/ -run TestDeployments_List`
Expected: PASS (both tests).

- [ ] **Step 6: Live-verify the env filter (safe read)**

Run: `go build -o bb . && ./bb deployment list --env-uuid <a-real-env-uuid> --format json` against a repo with deployments (use `./bb env list` to get an env uuid). Confirm it returns filtered results (not a 400). If it 400s, switch to client-side filtering as noted in the VERIFY block and adjust the test to assert no `q` param + a post-decode filter.

- [ ] **Step 7: Regen golden (if example/flags surfaced) + docs**

If the `deployment list` manifest example is unchanged, golden may not diff — that is correct. Run `go test ./cmd/ -update` regardless and confirm. Update README/llms.txt/CLAUDE.md `deployment list` entry to document `--env-uuid` and `--sort`.

- [ ] **Step 8: Full verification**

Run: `go test ./... && go vet ./... && gofmt -l cmd pkg internal`
Expected: all pass/clean.

- [ ] **Step 9: Commit**

```bash
git add pkg/bitbucket/deployment.go pkg/bitbucket/deployment_test.go cmd/deployment.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(deployment): filter list by --env-uuid and add --sort"
```

---

### Task 3: `pipeline-var get --uuid`

**Files:**
- Modify: `pkg/bitbucket/pipeline_variable.go`
- Modify: `pkg/bitbucket/pipeline_variable_test.go`
- Modify: `cmd/pipeline_var.go`
- Modify: `cmd/manifest_registry.go:55-58` (pipeline-var section)
- Regen: `cmd/testdata/manifest.golden.json`
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Produces: `func (r *PipelineVariableResource) Get(ctx context.Context, uuid string) (PipelineVariable, error)` — `GET .../pipelines_config/variables/{uuid}`.
- Consumes: existing `render.PipelineVariableDetail`.

- [ ] **Step 1: Write the failing test**

Add to `pkg/bitbucket/pipeline_variable_test.go`:

```go
func TestPipelineVariables_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/pipelines_config/variables/uuid-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, bitbucket.PipelineVariable{UUID: "uuid-1", Key: "ENV", Value: "prod", Secured: false})
	}))
	got, err := client.PipelineVariables("testws", "testrepo").Get(context.Background(), "uuid-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "ENV" || got.Value != "prod" {
		t.Errorf("unexpected variable: %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestPipelineVariables_Get`
Expected: FAIL — `Get` undefined.

- [ ] **Step 3: Implement the client method**

Add to `pkg/bitbucket/pipeline_variable.go`:

```go
// Get returns a single pipeline variable by UUID. Secured variables are
// returned without their value (the API hides it).
func (r *PipelineVariableResource) Get(ctx context.Context, uuid string) (PipelineVariable, error) {
	data, err := r.client.do(ctx, "GET", r.basePath()+uuid, nil, nil)
	if err != nil {
		return PipelineVariable{}, err
	}
	return decode[PipelineVariable](data)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/bitbucket/ -run TestPipelineVariables_Get`
Expected: PASS.

- [ ] **Step 5: Add the cmd leaf**

In `cmd/pipeline_var.go`, add a `pipeline-var get --uuid` command reusing `render.PipelineVariableDetail`:

```go
var pipelineVarGetUUID string

var pipelineVarGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a pipeline variable by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		v, err := client.PipelineVariables(ws, repo).Get(context.Background(), pipelineVarGetUUID)
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}
```

Register in `init()`: add flag + `MarkFlagRequired("uuid")` and add `pipelineVarGetCmd` to the `pipelineVarCmd.AddCommand(...)` call.

- [ ] **Step 6: Register in manifest + regen golden**

Add to `cmd/manifest_registry.go`:

```go
"pipeline-var get": {Action: actionRead, OutputType: "PipelineVariable", Example: "bb pipeline-var get --uuid '{uuid}'"},
```

Run `go test ./cmd/ -update`; confirm golden contains `pipeline-var get`.

- [ ] **Step 7: Sync docs + verify + commit**

Update README/llms.txt/CLAUDE.md (`pipeline-var list / create / delete` becomes `... / get / ...`). Then:

```bash
go test ./... && go vet ./... && gofmt -l cmd pkg internal
git add pkg/bitbucket/pipeline_variable.go pkg/bitbucket/pipeline_variable_test.go cmd/pipeline_var.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(pipeline-var): add get by uuid"
```

---

### Task 4: `pipeline-var update --uuid --value [--key] [--secured]` (stdin-capable)

**Files:**
- Modify: `pkg/bitbucket/pipeline_variable.go`
- Modify: `pkg/bitbucket/pipeline_variable_test.go`
- Modify: `cmd/pipeline_var.go`
- Modify: `cmd/manifest_registry.go`
- Regen: `cmd/testdata/manifest.golden.json`
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Consumes: `Get` from Task 3 (cmd flag path uses it to default key/secured), existing `CreatePipelineVariableInput` (reused as the PUT body), `stdinInputOr`, `requireFlag`.
- Produces: `func (r *PipelineVariableResource) Update(ctx context.Context, uuid string, input CreatePipelineVariableInput) (PipelineVariable, error)` — `PUT .../pipelines_config/variables/{uuid}` with the full `{key,value,secured}` representation.

**Design note (why the client stays thin):** Bitbucket's PUT replaces the whole variable and requires `key`. A secured variable's value is NOT readable via GET, so a fetch-then-merge that reused the fetched value would blank it. Therefore: the client just PUTs whatever body it is given; the cmd flag path REQUIRES `--value` and only fetches current to fill `key`/`secured` defaults. The stdin path supplies a complete body and PUTs it directly (no fetch).

- [ ] **Step 1: Write the failing client test**

Add to `pkg/bitbucket/pipeline_variable_test.go`:

```go
func TestPipelineVariables_Update(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/pipelines_config/variables/uuid-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["key"] != "ENV" || body["value"] != "staging" {
			t.Errorf("unexpected body: %+v", body)
		}
		mustEncodeJSON(t, w, bitbucket.PipelineVariable{UUID: "uuid-1", Key: "ENV", Value: "staging"})
	}))
	got, err := client.PipelineVariables("testws", "testrepo").Update(context.Background(), "uuid-1", bitbucket.CreatePipelineVariableInput{
		Key:   "ENV",
		Value: "staging",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != "staging" {
		t.Errorf("expected value staging, got %s", got.Value)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestPipelineVariables_Update`
Expected: FAIL — `Update` undefined.

- [ ] **Step 3: Implement the client method**

Add to `pkg/bitbucket/pipeline_variable.go`:

```go
// Update replaces a pipeline variable by UUID with the given representation.
// The Bitbucket PUT requires the full {key,value,secured} body.
func (r *PipelineVariableResource) Update(ctx context.Context, uuid string, input CreatePipelineVariableInput) (PipelineVariable, error) {
	data, err := r.client.do(ctx, "PUT", r.basePath()+uuid, input, nil)
	if err != nil {
		return PipelineVariable{}, err
	}
	return decode[PipelineVariable](data)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/bitbucket/ -run TestPipelineVariables_Update`
Expected: PASS.

- [ ] **Step 5: Add the cmd leaf**

In `cmd/pipeline_var.go`, add the update command. Flag path requires `--value` and fetches current to default `--key`/`--secured`; stdin path PUTs a full body directly:

```go
var (
	pipelineVarUpdateUUID    string
	pipelineVarUpdateKey     string
	pipelineVarUpdateValue   string
	pipelineVarUpdateSecured bool
)

var pipelineVarUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a pipeline variable by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := requireFlag("uuid", pipelineVarUpdateUUID); err != nil {
			return err
		}
		res := client.PipelineVariables(ws, repo)

		var input bitbucket.CreatePipelineVariableInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreatePipelineVariableInput {
			return bitbucket.CreatePipelineVariableInput{
				Key:     pipelineVarUpdateKey,
				Value:   pipelineVarUpdateValue,
				Secured: pipelineVarUpdateSecured,
			}
		})
		if err != nil {
			return err
		}

		if !consumed {
			// Flag path: --value is required; default key/secured from the
			// current variable unless overridden. (A secured variable's value
			// is not readable, so we never reuse the fetched value.)
			if err := requireFlag("value", pipelineVarUpdateValue); err != nil {
				return err
			}
			current, err := res.Get(context.Background(), pipelineVarUpdateUUID)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("key") {
				input.Key = current.Key
			}
			if !cmd.Flags().Changed("secured") {
				input.Secured = current.Secured
			}
		}

		v, err := res.Update(context.Background(), pipelineVarUpdateUUID, input)
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}
```

Register in `init()`:

```go
pipelineVarUpdateCmd.Flags().StringVar(&pipelineVarUpdateUUID, "uuid", "", "variable UUID (required)")
pipelineVarUpdateCmd.Flags().StringVarP(&pipelineVarUpdateKey, "key", "k", "", "new key (defaults to current)")
pipelineVarUpdateCmd.Flags().StringVarP(&pipelineVarUpdateValue, "value", "v", "", "new value (required unless piping JSON)")
pipelineVarUpdateCmd.Flags().BoolVar(&pipelineVarUpdateSecured, "secured", false, "mark variable as secured (defaults to current)")
// no MarkFlagRequired — uuid/value are validated in RunE so stdin JSON works.
```

Add `pipelineVarUpdateCmd` to the `pipelineVarCmd.AddCommand(...)` call.

- [ ] **Step 6: Register in manifest + regen golden**

Add:

```go
"pipeline-var update": {Action: actionWrite, OutputType: "PipelineVariable", StdinType: "CreatePipelineVariableInput", Example: "bb pipeline-var update --uuid '{uuid}' --value newval"},
```

Run `go test ./cmd/ -update`; confirm golden contains `pipeline-var update`.

- [ ] **Step 7: Sync docs + verify + commit**

Update README/llms.txt/CLAUDE.md (`pipeline-var` line becomes `list / get / create / update / delete`). Then:

```bash
go test ./... && go vet ./... && gofmt -l cmd pkg internal
git add pkg/bitbucket/pipeline_variable.go pkg/bitbucket/pipeline_variable_test.go cmd/pipeline_var.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(pipeline-var): add update by uuid (stdin-capable)"
```

---

## Self-Review

**Spec coverage** (audit #9 sub-items):
- `bb deployment get --uuid` → Task 1. ✓
- `bb deployment list --env-uuid` + sort → Task 2. ✓
- `bb pipeline-var update --uuid --value` (PUT) → Task 4. ✓
- `pipeline-var get` → Task 3. ✓
- Resolve env UUID → name in output → **explicitly deferred** (user decision); not in this plan. ✓

**Placeholder scan:** none — all steps carry real code/commands.

**Type consistency:** `DeploymentResource.Get`, `DeploymentListOptions{EnvUUID,Sort}`, `PipelineVariableResource.Get`/`Update`, reuse of `CreatePipelineVariableInput` and `render.PipelineVariableDetail`/new `render.DeploymentDetail` are consistent across tasks. Task 4's cmd flag path consumes Task 3's `Get`.

**Ordering:** Task 3 (Get) precedes Task 4 (Update uses Get for defaults). Task 1 before Task 2 (both touch deployment.go/test; Task 2 edits the List signature Task 1 leaves alone). Correct.
