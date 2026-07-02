# Backlog #8 — Environment CRUD + Env-Var Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `bb env get/create/delete` and a new `bb env-var list/create/update/delete --env-uuid UUID` command group, rounding out deployment-environment management and fixing the CLAUDE.md `env get` doc-drift.

**Architecture:** Mirror the existing thin-client + thin-cmd pattern (see the just-shipped `pipeline-var` and `deployment` commands). Environments extend the existing `EnvironmentResource`. Environment variables get a new `EnvironmentVariableResource` scoped by `(workspace, repo, envUUID)` that REUSES the existing `PipelineVariable` type and `CreatePipelineVariableInput` body (the API returns identical `{uuid,key,value,secured}` shapes — literally `type:"pipeline_variable"`), plus the existing `render.PipelineVariableList/Detail`.

**Tech Stack:** Go 1.26, Cobra, stdlib `net/http`/`net/http/httptest`, `encoding/json`.

**Scope decision (user-confirmed):** `env update` is DEFERRED — it uses an undocumented `POST .../environments/{uuid}/changes/` endpoint with no authoritative body schema. Not in this plan. API reference: `.superpowers/sdd/env-api-research.md`.

## Global Constraints

- Tests live only in `pkg/bitbucket/` via `newTestClient(t, handler)` + `mustEncodeJSON`. `cmd/` Cobra wiring is intentionally untested.
- Every new leaf command MUST get an entry in `commandRegistry` (`cmd/manifest_registry.go`) or `TestEveryLeafIsRegistered` fails. A new StdinType Go type MUST be added to `typeRegistry` in the same file (`CreateEnvironmentInput` is new; `PipelineVariable`/`CreatePipelineVariableInput`/`Environment` already exist).
- After any leaf/flag/example change, regenerate the manifest golden: `go test ./cmd/ -update`.
- Documentation sync REQUIRED in the same change: `README.md` (Commands block), `llms.txt`, and `CLAUDE.md` (Command hierarchy tree; add `env-var` as a new top-level group).
- Braced UUIDs (`{uuid}`) embedded in URL PATH segments MUST be wrapped in `url.PathEscape(...)` — matches the `pipeline.go` idiom (lines 60/293/300/319). Query params via `url.Values` are auto-encoded and need no manual escaping.
- `bb env-var` is a NEW top-level command (sibling of `bb env`), in a new file `cmd/env_var.go`. The client resource lives in a new file `pkg/bitbucket/environment_variable.go`.
- Env-var write ops and env create/delete are WRITE mutations: unit-test the request body/path only. Do NOT live-test writes without explicit user sign-off. Live-verifying READS (env get, env-var list) during TDD is expected and safe (creds configured: workspace `payfactopay`, repo `payment-platform-portal`; real env uuids: Test `{3682b0a5-8b98-4a6e-bef7-b330b02ec34c}`, prod `{a00e1c09-e3a3-43ba-9a85-8ac984f333dc}`, Staging `{863caa02-663e-40bb-a99c-b6a26f7f7c97}`).
- No em-dashes; plain ASCII punctuation only (prose, comments, commit messages).
- Never push/tag without explicit user sign-off.

---

### Task 1: `env get --uuid` + `env delete --uuid`

**Files:**
- Modify: `pkg/bitbucket/environment.go`
- Modify: `pkg/bitbucket/environment_test.go`
- Modify: `cmd/render/infra.go` (add `EnvDetailString`/`EnvDetail`)
- Modify: `cmd/env.go`
- Modify: `cmd/manifest_registry.go` (env section)
- Regen: `cmd/testdata/manifest.golden.json`
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Produces: `func (r *EnvironmentResource) Get(ctx context.Context, uuid string) (Environment, error)` — `GET .../environments/{uuid}` (PathEscape the uuid).
- Produces: `func (r *EnvironmentResource) Delete(ctx context.Context, uuid string) error` — `DELETE .../environments/{uuid}`.
- Produces: `func EnvDetailString(e bitbucket.Environment) string` + `func EnvDetail(e bitbucket.Environment)`.

- [ ] **Step 1: Write failing client tests**

Add to `pkg/bitbucket/environment_test.go` (read the existing `TestEnvironments_List` first to match style):

```go
func TestEnvironments_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/environments/%7Benv-1%7D" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		mustEncodeJSON(t, w, bitbucket.Environment{UUID: "{env-1}", Name: "Staging", EnvironmentType: bitbucket.EnvironmentType{Name: "Staging"}})
	}))
	got, err := client.Environments("testws", "testrepo").Get(context.Background(), "{env-1}")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Staging" {
		t.Errorf("unexpected env: %+v", got)
	}
}

func TestEnvironments_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/environments/%7Benv-1%7D" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := client.Environments("testws", "testrepo").Delete(context.Background(), "{env-1}"); err != nil {
		t.Fatal(err)
	}
}
```

NOTE on path assertions: assert against `r.URL.EscapedPath()` (NOT `r.URL.Path`). Go's httptest server DECODES `r.URL.Path` back to literal braces (`{env-1}`) regardless of client escaping, so it cannot verify `url.PathEscape`; `EscapedPath()` returns the wire form `%7Benv-1%7D` and thus confirms the client applied `PathEscape`. This mirrors the existing `TestPipelines_Log` (`pkg/bitbucket/pipeline_test.go:510-513`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/bitbucket/ -run TestEnvironments_`
Expected: FAIL — `Get`/`Delete` undefined.

- [ ] **Step 3: Implement client methods**

Add to `pkg/bitbucket/environment.go` (add `"net/url"` to imports):

```go
// Get returns a single deployment environment by UUID.
func (r *EnvironmentResource) Get(ctx context.Context, uuid string) (Environment, error) {
	data, err := r.client.do(ctx, "GET", r.basePath()+url.PathEscape(uuid), nil, nil)
	if err != nil {
		return Environment{}, err
	}
	return decode[Environment](data)
}

// Delete removes a deployment environment by UUID.
func (r *EnvironmentResource) Delete(ctx context.Context, uuid string) error {
	_, err := r.client.do(ctx, "DELETE", r.basePath()+url.PathEscape(uuid), nil, nil)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/bitbucket/ -run TestEnvironments_`
Expected: PASS (adjust the path assertion per the Step 1 NOTE if needed).

- [ ] **Step 5: Add render.EnvDetail**

In `cmd/render/infra.go`, add (model on the existing `EnvListString` field access — `e.UUID`, `e.Name`, `e.EnvironmentType.Name`, `e.Lock.Name`):

```go
// EnvDetailString returns formatted text for a single environment.
func EnvDetailString(e bitbucket.Environment) string {
	var b strings.Builder
	fmt.Fprintf(&b, "UUID: %s\n", e.UUID)
	fmt.Fprintf(&b, "Name: %s\n", e.Name)
	fmt.Fprintf(&b, "Type: %s\n", e.EnvironmentType.Name)
	fmt.Fprintf(&b, "Lock: %s\n", e.Lock.Name)
	return b.String()
}

func EnvDetail(e bitbucket.Environment) { fmt.Print(EnvDetailString(e)) }
```

- [ ] **Step 6: Add cmd leaves**

In `cmd/env.go`, add `env get --uuid` and `env delete --uuid` (model on `deployment get` and `pipeline-var delete`), each with a required `--uuid` flag via `MarkFlagRequired("uuid")`. `get` uses `printOutput(env, func(){ render.EnvDetail(env) })`; `delete` uses `printOutput(map[string]any{"deleted": true, "uuid": deploymentGetUUID}, func(){ fmt.Printf("Environment %s deleted\n", uuid) })` (add `"fmt"` import). Wire both into `envCmd.AddCommand(...)`.

- [ ] **Step 7: Manifest + golden**

Add to `cmd/manifest_registry.go` env section:

```go
"env get":    {Action: actionRead, OutputType: "Environment", Example: "bb env get --uuid '{uuid}'"},
"env delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb env delete --uuid '{uuid}'"},
```

Run `go test ./cmd/ -update`; confirm both appear in the golden.

- [ ] **Step 8: Docs + live-verify get + full verify + commit**

Docs: `CLAUDE.md` `env list` -> `env list / get / delete`; README + llms.txt add the two lines.
Live-verify (safe read): `go build -o bb . && ./bb env get --uuid '{a00e1c09-e3a3-43ba-9a85-8ac984f333dc}' --format json` returns the prod environment. Record the result.
Full verify: `go test ./... && go vet ./... && gofmt -l cmd pkg internal`.

```bash
git add pkg/bitbucket/environment.go pkg/bitbucket/environment_test.go cmd/render/infra.go cmd/env.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(env): add env get and env delete by uuid"
```

---

### Task 2: `env create --name --type` (stdin-capable)

**Files:**
- Modify: `pkg/bitbucket/environment.go`
- Modify: `pkg/bitbucket/environment_test.go`
- Modify: `pkg/bitbucket/types.go` (add `CreateEnvironmentInput`)
- Modify: `cmd/env.go`
- Modify: `cmd/manifest_registry.go` (env section + typeRegistry)
- Regen: `cmd/testdata/manifest.golden.json`
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Consumes: `render.EnvDetail` (Task 1), `stdinInputOr`, `requireFlag`.
- Produces: `type CreateEnvironmentInput struct { Name string \`json:"name"\`; EnvironmentType EnvironmentType \`json:"environment_type"\` }` (reuses existing `EnvironmentType{Name string}`).
- Produces: `func (r *EnvironmentResource) Create(ctx context.Context, input CreateEnvironmentInput) (Environment, error)` — `POST .../environments/`.

- [ ] **Step 1: Add the input type**

In `pkg/bitbucket/types.go`, near the `Environment` types (around line 484), add:

```go
// CreateEnvironmentInput is the request body for creating a deployment
// environment. EnvironmentType.Name must be one of "Test", "Staging",
// "Production" (case-sensitive).
type CreateEnvironmentInput struct {
	Name            string          `json:"name"`
	EnvironmentType EnvironmentType `json:"environment_type"`
}
```

- [ ] **Step 2: Write failing client test**

Add to `pkg/bitbucket/environment_test.go`:

```go
func TestEnvironments_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/environments/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["name"] != "QA" {
			t.Errorf("expected name=QA, got %v", body["name"])
		}
		et, ok := body["environment_type"].(map[string]any)
		if !ok || et["name"] != "Test" {
			t.Errorf("expected environment_type.name=Test, got %v", body["environment_type"])
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.Environment{UUID: "{env-9}", Name: "QA", EnvironmentType: bitbucket.EnvironmentType{Name: "Test"}})
	}))
	got, err := client.Environments("testws", "testrepo").Create(context.Background(), bitbucket.CreateEnvironmentInput{
		Name:            "QA",
		EnvironmentType: bitbucket.EnvironmentType{Name: "Test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "QA" {
		t.Errorf("expected name QA, got %s", got.Name)
	}
}
```

Ensure `encoding/json` is imported in the test file.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestEnvironments_Create`
Expected: FAIL — `Create`/`CreateEnvironmentInput` undefined.

- [ ] **Step 4: Implement client Create**

Add to `pkg/bitbucket/environment.go`:

```go
// Create adds a new deployment environment to the repository.
func (r *EnvironmentResource) Create(ctx context.Context, input CreateEnvironmentInput) (Environment, error) {
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Environment{}, err
	}
	return decode[Environment](data)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./pkg/bitbucket/ -run TestEnvironments_Create`
Expected: PASS.

- [ ] **Step 6: Add cmd leaf (stdin-capable)**

In `cmd/env.go`, add `env create` modeled on `pipeline-var create`. Flags: `--name` (`-n` reserved? check; use `--name` only to avoid clashing), `--type` (default none). Validate `name` and `type` in RunE via `requireFlag` when not consumed from stdin. Build `CreateEnvironmentInput{Name: name, EnvironmentType: bitbucket.EnvironmentType{Name: envType}}`:

```go
var (
	envCreateName string
	envCreateType string
)

var envCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a deployment environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		var input bitbucket.CreateEnvironmentInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreateEnvironmentInput {
			return bitbucket.CreateEnvironmentInput{
				Name:            envCreateName,
				EnvironmentType: bitbucket.EnvironmentType{Name: envCreateType},
			}
		})
		if err != nil {
			return err
		}
		if !consumed {
			if err := requireFlag("name", envCreateName); err != nil {
				return err
			}
			if err := requireFlag("type", envCreateType); err != nil {
				return err
			}
		}
		env, err := client.Environments(ws, repo).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(env, func() { render.EnvDetail(env) })
	},
}
```

Register flags in `init()`:

```go
envCreateCmd.Flags().StringVar(&envCreateName, "name", "", "environment name (required)")
envCreateCmd.Flags().StringVar(&envCreateType, "type", "", "environment type: Test, Staging, or Production (required)")
// no MarkFlagRequired -- name/type validated in RunE so stdin JSON works.
```

Add `envCreateCmd` to `envCmd.AddCommand(...)`.

- [ ] **Step 7: Manifest + typeRegistry + golden**

In `cmd/manifest_registry.go`: add to env section
```go
"env create": {Action: actionWrite, OutputType: "Environment", StdinType: "CreateEnvironmentInput", Example: "bb env create --name QA --type Test"},
```
and add to `typeRegistry`:
```go
"CreateEnvironmentInput": bitbucket.CreateEnvironmentInput{},
```
Run `go test ./cmd/ -update`; confirm `env create` in golden.

- [ ] **Step 8: Docs + verify + commit**

Docs: `env list / get / delete` -> `env list / get / create / delete` in CLAUDE.md; README + llms.txt add the create line (document the 3 allowed type values). NOTE `env create` is a WRITE — do NOT live-test. Verify only: `./bb env create` with no flags errors `validation_failed` for missing name; `--help` lists flags.

```bash
go test ./... && go vet ./... && gofmt -l cmd pkg internal
git add pkg/bitbucket/environment.go pkg/bitbucket/environment_test.go pkg/bitbucket/types.go cmd/env.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(env): add env create (stdin-capable)"
```

---

### Task 3: `EnvironmentVariableResource` + `env-var list --env-uuid`

**Files:**
- Create: `pkg/bitbucket/environment_variable.go`
- Create: `pkg/bitbucket/environment_variable_test.go`
- Modify: `pkg/bitbucket/client.go` (add `EnvironmentVariables` constructor)
- Create: `cmd/env_var.go`
- Modify: `cmd/manifest_registry.go` (new env-var section)
- Regen: `cmd/testdata/manifest.golden.json`
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Consumes: `repoPath`, `pagelenDefault`, `decode[paged[PipelineVariable]]`, `render.PipelineVariableList`.
- Produces: `type EnvironmentVariableResource struct { client *Client; workspace, repo, envUUID string }` with `basePath()` = `repoPath(ws,repo) + "/deployments_config/environments/" + url.PathEscape(envUUID) + "/variables/"`.
- Produces: `func (c *Client) EnvironmentVariables(workspace, repo, envUUID string) *EnvironmentVariableResource`.
- Produces: `func (r *EnvironmentVariableResource) List(ctx context.Context) ([]PipelineVariable, error)`.

- [ ] **Step 1: Write failing client test**

Create `pkg/bitbucket/environment_variable_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestEnvironmentVariables_List(t *testing.T) {
	vars := []bitbucket.PipelineVariable{
		{UUID: "{v-1}", Key: "API_URL", Value: "https://x", Secured: false},
		{UUID: "{v-2}", Key: "SECRET", Value: "", Secured: true},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/deployments_config/environments/%7Benv-1%7D/variables/" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		mustEncodeJSON(t, w, map[string]any{"values": vars})
	}))
	got, err := client.EnvironmentVariables("testws", "testrepo", "{env-1}").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Key != "SECRET" || !got[1].Secured {
		t.Errorf("unexpected result: %+v", got)
	}
}
```

(Adjust the encoded path assertion per Task 1's NOTE if the server records it differently. The `encoding/json` import is used by later tasks in this file.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestEnvironmentVariables_List`
Expected: FAIL — `EnvironmentVariables` undefined.

- [ ] **Step 3: Implement the resource + constructor + List**

Create `pkg/bitbucket/environment_variable.go`:

```go
package bitbucket

import (
	"context"
	"net/url"
)

// EnvironmentVariableResource provides operations on deployment-environment
// variables. The API returns the same shape as repository pipeline variables
// (type "pipeline_variable"), so it reuses PipelineVariable and
// CreatePipelineVariableInput.
type EnvironmentVariableResource struct {
	client    *Client
	workspace string
	repo      string
	envUUID   string
}

func (r *EnvironmentVariableResource) basePath() string {
	return repoPath(r.workspace, r.repo) + "/deployments_config/environments/" + url.PathEscape(r.envUUID) + "/variables/"
}

// List returns all variables for the environment. Secured variables are
// returned with an empty value (the API hides it).
func (r *EnvironmentVariableResource) List(ctx context.Context) ([]PipelineVariable, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[PipelineVariable]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
```

Add the constructor to `pkg/bitbucket/client.go` (near the other resource constructors, e.g. after `Environments`):

```go
func (c *Client) EnvironmentVariables(workspace, repo, envUUID string) *EnvironmentVariableResource {
	return &EnvironmentVariableResource{client: c, workspace: workspace, repo: repo, envUUID: envUUID}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/bitbucket/ -run TestEnvironmentVariables_List`
Expected: PASS.

- [ ] **Step 5: Add the `env-var` top-level command + list leaf**

Create `cmd/env_var.go`:

```go
package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
)

var envVarCmd = &cobra.Command{
	Use:   "env-var",
	Short: "Manage deployment environment variables",
}

var envVarListEnvUUID string

var envVarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List variables for a deployment environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := requireFlag("env-uuid", envVarListEnvUUID); err != nil {
			return err
		}
		vars, err := client.EnvironmentVariables(ws, repo, envVarListEnvUUID).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(vars, func() { render.PipelineVariableList(vars) })
	},
}

func init() {
	envVarListCmd.Flags().StringVar(&envVarListEnvUUID, "env-uuid", "", "environment UUID (required)")
	envVarCmd.AddCommand(envVarListCmd)
	rootCmd.AddCommand(envVarCmd)
}
```

(Later tasks add create/update/delete leaves to this file and their flags to a shared or per-command flag set.)

- [ ] **Step 6: Manifest + golden**

Add a new env-var section to `cmd/manifest_registry.go`:

```go
// env-var ----------------------------------------------------------
"env-var list": {Action: actionRead, OutputType: "[]PipelineVariable", Ordering: "unspecified", Example: "bb env-var list --env-uuid '{uuid}'"},
```

Run `go test ./cmd/ -update`; confirm `env-var list` in golden.

- [ ] **Step 7: Docs + live-verify list + full verify + commit**

Docs: add a new `env-var` group to CLAUDE.md hierarchy (`env-var list`); README + llms.txt add the `bb env-var list --env-uuid UUID` line.
Live-verify (safe read): `go build -o bb . && ./bb env-var list --env-uuid '{a00e1c09-e3a3-43ba-9a85-8ac984f333dc}' --format json` returns the prod env's variables (or an empty list). Record the result; if the path 404s, re-check the `deployments_config` segment against `.superpowers/sdd/env-api-research.md`.
Full verify: `go test ./... && go vet ./... && gofmt -l cmd pkg internal`.

```bash
git add pkg/bitbucket/environment_variable.go pkg/bitbucket/environment_variable_test.go pkg/bitbucket/client.go cmd/env_var.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(env-var): add env-var list for a deployment environment"
```

---

### Task 4: `env-var create` + `env-var delete`

**Files:**
- Modify: `pkg/bitbucket/environment_variable.go`
- Modify: `pkg/bitbucket/environment_variable_test.go`
- Modify: `cmd/env_var.go`
- Modify: `cmd/manifest_registry.go` (env-var section)
- Regen: `cmd/testdata/manifest.golden.json`
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Consumes: `CreatePipelineVariableInput`, `render.PipelineVariableDetail`, `stdinInputOr`, `requireFlag`.
- Produces: `func (r *EnvironmentVariableResource) Create(ctx context.Context, input CreatePipelineVariableInput) (PipelineVariable, error)` — `POST basePath()`.
- Produces: `func (r *EnvironmentVariableResource) Delete(ctx context.Context, uuid string) error` — `DELETE basePath()+url.PathEscape(uuid)`.

- [ ] **Step 1: Write failing client tests**

Add to `pkg/bitbucket/environment_variable_test.go`:

```go
func TestEnvironmentVariables_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/deployments_config/environments/%7Benv-1%7D/variables/" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["key"] != "API_URL" || body["value"] != "https://x" {
			t.Errorf("unexpected body: %+v", body)
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.PipelineVariable{UUID: "{v-9}", Key: "API_URL", Value: "https://x"})
	}))
	got, err := client.EnvironmentVariables("testws", "testrepo", "{env-1}").Create(context.Background(), bitbucket.CreatePipelineVariableInput{
		Key:   "API_URL",
		Value: "https://x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "API_URL" {
		t.Errorf("expected key API_URL, got %s", got.Key)
	}
}

func TestEnvironmentVariables_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/deployments_config/environments/%7Benv-1%7D/variables/%7Bv-1%7D" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := client.EnvironmentVariables("testws", "testrepo", "{env-1}").Delete(context.Background(), "{v-1}"); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/bitbucket/ -run TestEnvironmentVariables_Create -run TestEnvironmentVariables_Delete`
(Or `-run 'TestEnvironmentVariables_(Create|Delete)'`.) Expected: FAIL — methods undefined.

- [ ] **Step 3: Implement Create + Delete**

Add to `pkg/bitbucket/environment_variable.go`:

```go
// Create adds a new variable to the environment.
func (r *EnvironmentVariableResource) Create(ctx context.Context, input CreatePipelineVariableInput) (PipelineVariable, error) {
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return PipelineVariable{}, err
	}
	return decode[PipelineVariable](data)
}

// Delete removes an environment variable by UUID.
func (r *EnvironmentVariableResource) Delete(ctx context.Context, uuid string) error {
	_, err := r.client.do(ctx, "DELETE", r.basePath()+url.PathEscape(uuid), nil, nil)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/bitbucket/ -run 'TestEnvironmentVariables_(Create|Delete)'`
Expected: PASS.

- [ ] **Step 5: Add cmd leaves**

In `cmd/env_var.go`, add `env-var create` (stdin-capable, model on `pipeline-var create` but with a required `--env-uuid` flag validated in RunE) and `env-var delete` (`--env-uuid` + `--uuid`, both required via `requireFlag` or `MarkFlagRequired`). `create` builds `CreatePipelineVariableInput{Key,Value,Secured}` and prints `render.PipelineVariableDetail`. `delete` prints `map[string]any{"deleted": true, "uuid": uuid}`. Register flags and `AddCommand` both. Import `"fmt"` and `bitbucket` as needed.

Flags:
```go
envVarCreateCmd.Flags().StringVar(&envVarCreateEnvUUID, "env-uuid", "", "environment UUID (required)")
envVarCreateCmd.Flags().StringVarP(&envVarCreateKey, "key", "k", "", "variable key (required)")
envVarCreateCmd.Flags().StringVarP(&envVarCreateValue, "value", "v", "", "variable value")
envVarCreateCmd.Flags().BoolVar(&envVarCreateSecured, "secured", false, "mark variable as secured")
// env-uuid and key validated in RunE so stdin JSON works.

envVarDeleteCmd.Flags().StringVar(&envVarDeleteEnvUUID, "env-uuid", "", "environment UUID (required)")
envVarDeleteCmd.Flags().StringVar(&envVarDeleteUUID, "uuid", "", "variable UUID (required)")
envVarDeleteCmd.MarkFlagRequired("env-uuid")
envVarDeleteCmd.MarkFlagRequired("uuid")
```

- [ ] **Step 6: Manifest + golden**

Add:
```go
"env-var create": {Action: actionWrite, OutputType: "PipelineVariable", StdinType: "CreatePipelineVariableInput", Example: "bb env-var create --env-uuid '{uuid}' --key API_URL --value https://x"},
"env-var delete": {Action: actionDestructive, OutputType: "ResultMap", Example: "bb env-var delete --env-uuid '{uuid}' --uuid '{var-uuid}'"},
```
Run `go test ./cmd/ -update`.

- [ ] **Step 7: Docs + verify + commit**

Docs: env-var group -> `list / create / delete` in CLAUDE.md; README + llms.txt add both lines. WRITE ops - do NOT live-test; verify no-flag error paths + `--help`.

```bash
go test ./... && go vet ./... && gofmt -l cmd pkg internal
git add pkg/bitbucket/environment_variable.go pkg/bitbucket/environment_variable_test.go cmd/env_var.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(env-var): add env-var create and delete"
```

---

### Task 5: `env-var update` (stdin-capable)

**Files:**
- Modify: `pkg/bitbucket/environment_variable.go`
- Modify: `pkg/bitbucket/environment_variable_test.go`
- Modify: `cmd/env_var.go`
- Modify: `cmd/manifest_registry.go` (env-var section)
- Regen: `cmd/testdata/manifest.golden.json`
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Consumes: `Get`-less design — there is no env-var Get in scope, so the flag-path default-fetch uses `List` to find the current variable by UUID. See the design note.
- Produces: `func (r *EnvironmentVariableResource) Update(ctx context.Context, uuid string, input CreatePipelineVariableInput) (PipelineVariable, error)` — `PUT basePath()+url.PathEscape(uuid)`.

**Design note (mirror pipeline-var update, adapted):** The client `Update` is a THIN PUT of the full `{key,value,secured}` body. The cmd flag path REQUIRES `--value` (a secured variable's value is unreadable, so never reuse a fetched value). Since there is no `env-var get`, default `key`/`secured` when not provided by scanning `List(envUUID)` for the matching `uuid`. If the uuid is not found in the list, return a `not_found`-style error (`fmt.Errorf("variable %s not found in environment", uuid)`). The stdin path PUTs a complete body directly with no lookup.

- [ ] **Step 1: Write failing client test**

Add to `pkg/bitbucket/environment_variable_test.go`:

```go
func TestEnvironmentVariables_Update(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/deployments_config/environments/%7Benv-1%7D/variables/%7Bv-1%7D" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["key"] != "API_URL" || body["value"] != "https://y" || body["secured"] != false {
			t.Errorf("unexpected body: %+v", body)
		}
		mustEncodeJSON(t, w, bitbucket.PipelineVariable{UUID: "{v-1}", Key: "API_URL", Value: "https://y"})
	}))
	got, err := client.EnvironmentVariables("testws", "testrepo", "{env-1}").Update(context.Background(), "{v-1}", bitbucket.CreatePipelineVariableInput{
		Key:   "API_URL",
		Value: "https://y",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != "https://y" {
		t.Errorf("expected value https://y, got %s", got.Value)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestEnvironmentVariables_Update`
Expected: FAIL — `Update` undefined.

- [ ] **Step 3: Implement client Update**

Add to `pkg/bitbucket/environment_variable.go`:

```go
// Update replaces an environment variable by UUID with the given
// representation. The Bitbucket PUT requires the full {key,value,secured} body.
func (r *EnvironmentVariableResource) Update(ctx context.Context, uuid string, input CreatePipelineVariableInput) (PipelineVariable, error) {
	data, err := r.client.do(ctx, "PUT", r.basePath()+url.PathEscape(uuid), input, nil)
	if err != nil {
		return PipelineVariable{}, err
	}
	return decode[PipelineVariable](data)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/bitbucket/ -run TestEnvironmentVariables_Update`
Expected: PASS.

- [ ] **Step 5: Add cmd leaf (stdin-capable, List-based defaults)**

In `cmd/env_var.go`, add `env-var update`. Flag path: require `--env-uuid`, `--uuid`, `--value`; default `key`/`secured` from the matching entry in `List` when not changed:

```go
var (
	envVarUpdateEnvUUID string
	envVarUpdateUUID    string
	envVarUpdateKey     string
	envVarUpdateValue   string
	envVarUpdateSecured bool
)

var envVarUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a deployment environment variable by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := requireFlag("env-uuid", envVarUpdateEnvUUID); err != nil {
			return err
		}
		if err := requireFlag("uuid", envVarUpdateUUID); err != nil {
			return err
		}
		res := client.EnvironmentVariables(ws, repo, envVarUpdateEnvUUID)

		var input bitbucket.CreatePipelineVariableInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreatePipelineVariableInput {
			return bitbucket.CreatePipelineVariableInput{
				Key:     envVarUpdateKey,
				Value:   envVarUpdateValue,
				Secured: envVarUpdateSecured,
			}
		})
		if err != nil {
			return err
		}

		if !consumed {
			// Flag path: --value is required; default key/secured from the
			// current variable (found via List, since there is no env-var get)
			// unless overridden. A secured value is unreadable, so it is never reused.
			if err := requireFlag("value", envVarUpdateValue); err != nil {
				return err
			}
			if !cmd.Flags().Changed("key") || !cmd.Flags().Changed("secured") {
				vars, err := res.List(context.Background())
				if err != nil {
					return err
				}
				var current *bitbucket.PipelineVariable
				for i := range vars {
					if vars[i].UUID == envVarUpdateUUID {
						current = &vars[i]
						break
					}
				}
				if current == nil {
					return fmt.Errorf("variable %s not found in environment %s", envVarUpdateUUID, envVarUpdateEnvUUID)
				}
				if !cmd.Flags().Changed("key") {
					input.Key = current.Key
				}
				if !cmd.Flags().Changed("secured") {
					input.Secured = current.Secured
				}
			}
		}

		v, err := res.Update(context.Background(), envVarUpdateUUID, input)
		if err != nil {
			return err
		}
		return printOutput(v, func() { render.PipelineVariableDetail(v) })
	},
}
```

Register flags (uuid/env-uuid/value validated in RunE, so no MarkFlagRequired):

```go
envVarUpdateCmd.Flags().StringVar(&envVarUpdateEnvUUID, "env-uuid", "", "environment UUID (required)")
envVarUpdateCmd.Flags().StringVar(&envVarUpdateUUID, "uuid", "", "variable UUID (required)")
envVarUpdateCmd.Flags().StringVarP(&envVarUpdateKey, "key", "k", "", "new key (defaults to current)")
envVarUpdateCmd.Flags().StringVarP(&envVarUpdateValue, "value", "v", "", "new value (required unless piping JSON)")
envVarUpdateCmd.Flags().BoolVar(&envVarUpdateSecured, "secured", false, "mark variable as secured (defaults to current)")
```

Add `envVarUpdateCmd` to `envVarCmd.AddCommand(...)`.

- [ ] **Step 6: Manifest + golden**

Add:
```go
"env-var update": {Action: actionWrite, OutputType: "PipelineVariable", StdinType: "CreatePipelineVariableInput", Example: "bb env-var update --env-uuid '{uuid}' --uuid '{var-uuid}' --value newval"},
```
Run `go test ./cmd/ -update`.

- [ ] **Step 7: Docs + verify + commit**

Docs: env-var group -> `list / create / update / delete` in CLAUDE.md; README + llms.txt add the update line (note the secured-value write-only caveat). WRITE op - do NOT live-test; verify no-flag error path + `--help`.

```bash
go test ./... && go vet ./... && gofmt -l cmd pkg internal
git add pkg/bitbucket/environment_variable.go pkg/bitbucket/environment_variable_test.go cmd/env_var.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(env-var): add env-var update by uuid (stdin-capable)"
```

---

## Self-Review

**Spec coverage** (audit #8 sub-items):
- `env get` -> Task 1 (also fixes CLAUDE.md `env get` doc-drift). ✓
- `env create` -> Task 2. ✓
- `env update` -> DEFERRED (user decision; undocumented `/changes/` endpoint). ✓
- `env delete` -> Task 1. ✓
- `env-var list` -> Task 3. ✓
- `env-var create` -> Task 4. ✓
- `env-var update` -> Task 5. ✓
- `env-var delete` -> Task 4. ✓

**Placeholder scan:** none — all steps carry real code/commands.

**Type consistency:** `EnvironmentResource.Get/Create/Delete`, `CreateEnvironmentInput{Name, EnvironmentType}`, new `EnvironmentVariableResource` (List/Create/Update/Delete) reusing `PipelineVariable` + `CreatePipelineVariableInput`, and `client.EnvironmentVariables(ws,repo,envUUID)` are consistent across tasks. `render.EnvDetail` (Task 1) is consumed by Task 2. All path UUIDs use `url.PathEscape`.

**Ordering:** Task 1 (env get/delete + EnvDetail) before Task 2 (env create uses EnvDetail). Task 3 (env-var resource + List) before Tasks 4/5 (create/delete/update reuse the resource; Task 5 update uses List for defaults). Correct.

**Path-encoding:** client tests assert percent-encoded paths (`%7B...%7D`) via `r.URL.EscapedPath()` (not `r.URL.Path`, which the server decodes back to literal braces). This verifies `url.PathEscape` is applied, mirroring `TestPipelines_Log`. See Task 1 Step 1 NOTE.
