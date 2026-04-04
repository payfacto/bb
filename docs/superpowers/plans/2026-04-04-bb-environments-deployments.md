# BB CLI Environments & Deployments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `bb env list` and `bb deployment list` commands that surface deployment environments and recent deployments for a Bitbucket repository.

**Architecture:** Two independent resources — `EnvironmentResource` (List only) and `DeploymentResource` (List only) — each mirroring the existing resource pattern. `DeploymentState` mirrors `PipelineState` (optional nested result/status). The cmd layer is split across two files (`cmd/env.go`, `cmd/deployment.go`) for clean separation. Environments and Deployments are separate resources with separate endpoints; they share only a UUID reference (deployment knows its environment by UUID).

**Tech Stack:** Go 1.25, `github.com/spf13/cobra v1.10.2`, `net/http/httptest` (stdlib)

---

## File Structure

**New files:**

| File | Responsibility |
|------|---------------|
| `pkg/bitbucket/environment.go` | EnvironmentResource: List |
| `pkg/bitbucket/environment_test.go` | Tests for EnvironmentResource |
| `pkg/bitbucket/deployment.go` | DeploymentResource: List |
| `pkg/bitbucket/deployment_test.go` | Tests for DeploymentResource |
| `cmd/env.go` | `bb env list` |
| `cmd/deployment.go` | `bb deployment list` |

**Modified files:**

| File | Change |
|------|--------|
| `pkg/bitbucket/types.go` | Append Environment and Deployment types |
| `pkg/bitbucket/client.go` | Add `Environments()` and `Deployments()` accessors |

---

### Task 1: Add Environment and Deployment types to pkg/bitbucket/types.go

**Files:**
- Modify: `pkg/bitbucket/types.go`

- [ ] **Step 1: Append types to types.go**

Open `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/types.go` and append the following block at the end of the file (after the Tag types section):

```go
// Environment types

type Environment struct {
	UUID            string          `json:"uuid"`
	Name            string          `json:"name"`
	EnvironmentType EnvironmentType `json:"environment_type"`
	Lock            EnvironmentLock `json:"lock"`
}

type EnvironmentType struct {
	Name string `json:"name"` // "Production", "Staging", "Test"
}

type EnvironmentLock struct {
	Name string `json:"name"` // "UNLOCKED", "LOCKED"
}

// Deployment types

type Deployment struct {
	UUID           string           `json:"uuid"`
	State          DeploymentState  `json:"state"`
	Environment    DeploymentEnvRef `json:"environment"`
	Deployable     Deployable       `json:"deployable"`
	LastUpdateTime string           `json:"last_update_time"`
}

type DeploymentState struct {
	Name   string            `json:"name"`
	Status *DeploymentStatus `json:"status,omitempty"`
}

type DeploymentStatus struct {
	Name string `json:"name"` // "SUCCESSFUL", "FAILED"
}

type DeploymentEnvRef struct {
	UUID string `json:"uuid"`
}

type Deployable struct {
	Commit   *DeployableCommit    `json:"commit,omitempty"`
	Pipeline *DeployablePipeline  `json:"pipeline,omitempty"`
}

type DeployableCommit struct {
	Hash string `json:"hash"`
}

type DeployablePipeline struct {
	UUID string `json:"uuid"`
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go build ./...
```

Expected: No output (no errors).

- [ ] **Step 3: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/types.go
git commit -m "feat: add Environment and Deployment types"
```

---

### Task 2: EnvironmentResource

**Files:**
- Create: `pkg/bitbucket/environment_test.go`
- Create: `pkg/bitbucket/environment.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoint:**
- `GET /repositories/{ws}/{repo}/environments/?pagelen=50` — returns paged list of environments

Note the trailing slash on the path — this mirrors the pipeline resource pattern in this codebase.

- [ ] **Step 1: Create environment_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/environment_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestEnvironments_List(t *testing.T) {
	envs := []bitbucket.Environment{
		{
			UUID:            "{env-prod}",
			Name:            "Production",
			EnvironmentType: bitbucket.EnvironmentType{Name: "Production"},
			Lock:            bitbucket.EnvironmentLock{Name: "UNLOCKED"},
		},
		{
			UUID:            "{env-stg}",
			Name:            "Staging",
			EnvironmentType: bitbucket.EnvironmentType{Name: "Staging"},
			Lock:            bitbucket.EnvironmentLock{Name: "UNLOCKED"},
		},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/environments/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": envs})
	}))
	got, err := client.Environments("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Production" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[0].EnvironmentType.Name != "Production" {
		t.Errorf("expected EnvironmentType.Name=Production, got %s", got[0].EnvironmentType.Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestEnvironments -v 2>&1 | head -5
```

Expected: Compilation error — `client.Environments undefined`.

- [ ] **Step 3: Create environment.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/environment.go`:

```go
package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// EnvironmentResource provides operations on repository deployment environments.
type EnvironmentResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *EnvironmentResource) basePath() string {
	return fmt.Sprintf("%s/environments/", repoPath(r.workspace, r.repo))
}

// List returns all deployment environments in the repository.
func (r *EnvironmentResource) List(ctx context.Context) ([]Environment, error) {
	q := url.Values{"pagelen": {"50"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Environment]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
```

- [ ] **Step 4: Add Environments() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after the existing `Tags()` method:

```go
// Environments returns a resource for deployment environment operations on the given repo.
func (c *Client) Environments(workspace, repo string) *EnvironmentResource {
	return &EnvironmentResource{client: c, workspace: workspace, repo: repo}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestEnvironments -count=1 -v
```

Expected: `TestEnvironments_List` PASS.

- [ ] **Step 6: Run full test suite**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -count=1
```

Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/environment.go pkg/bitbucket/environment_test.go pkg/bitbucket/client.go
git commit -m "feat: add EnvironmentResource (list)"
```

---

### Task 3: DeploymentResource

**Files:**
- Create: `pkg/bitbucket/deployment_test.go`
- Create: `pkg/bitbucket/deployment.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoint:**
- `GET /repositories/{ws}/{repo}/deployments/?sort=-last_update_time&pagelen=25`

Note the trailing slash on the path. The `DeploymentState` mirrors `PipelineState` — it has an optional nested `Status` (for COMPLETED states with SUCCESSFUL/FAILED).

- [ ] **Step 1: Create deployment_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/deployment_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestDeployments_List(t *testing.T) {
	deployments := []bitbucket.Deployment{
		{
			UUID:  "{dep-1}",
			State: bitbucket.DeploymentState{Name: "COMPLETED", Status: &bitbucket.DeploymentStatus{Name: "SUCCESSFUL"}},
			Environment: bitbucket.DeploymentEnvRef{UUID: "{env-prod}"},
			Deployable: bitbucket.Deployable{
				Commit:   &bitbucket.DeployableCommit{Hash: "abc123"},
				Pipeline: &bitbucket.DeployablePipeline{UUID: "{pipe-1}"},
			},
			LastUpdateTime: "2024-01-15T10:00:00+00:00",
		},
		{
			UUID:  "{dep-2}",
			State: bitbucket.DeploymentState{Name: "IN_PROGRESS"},
			Environment: bitbucket.DeploymentEnvRef{UUID: "{env-stg}"},
			Deployable:  bitbucket.Deployable{},
			LastUpdateTime: "2024-01-14T09:00:00+00:00",
		},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/deployments/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("sort") != "-last_update_time" {
			t.Errorf("expected sort=-last_update_time, got %s", r.URL.Query().Get("sort"))
		}
		if r.URL.Query().Get("pagelen") != "25" {
			t.Errorf("expected pagelen=25, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": deployments})
	}))
	got, err := client.Deployments("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].UUID != "{dep-1}" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[0].State.Status == nil || got[0].State.Status.Name != "SUCCESSFUL" {
		t.Errorf("expected State.Status.Name=SUCCESSFUL, got %+v", got[0].State.Status)
	}
	if got[0].Deployable.Commit == nil || got[0].Deployable.Commit.Hash != "abc123" {
		t.Errorf("expected Deployable.Commit.Hash=abc123, got %+v", got[0].Deployable.Commit)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestDeployments -v 2>&1 | head -5
```

Expected: Compilation error — `client.Deployments undefined`.

- [ ] **Step 3: Create deployment.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/deployment.go`:

```go
package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// DeploymentResource provides operations on repository deployments.
type DeploymentResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *DeploymentResource) basePath() string {
	return fmt.Sprintf("%s/deployments/", repoPath(r.workspace, r.repo))
}

// List returns the most recent deployments, newest first.
func (r *DeploymentResource) List(ctx context.Context) ([]Deployment, error) {
	q := url.Values{"sort": {"-last_update_time"}, "pagelen": {"25"}}
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

- [ ] **Step 4: Add Deployments() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after the `Environments()` method:

```go
// Deployments returns a resource for deployment operations on the given repo.
func (c *Client) Deployments(workspace, repo string) *DeploymentResource {
	return &DeploymentResource{client: c, workspace: workspace, repo: repo}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestDeployments -count=1 -v
```

Expected: `TestDeployments_List` PASS.

- [ ] **Step 6: Run full test suite**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -count=1
```

Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/deployment.go pkg/bitbucket/deployment_test.go pkg/bitbucket/client.go
git commit -m "feat: add DeploymentResource (list)"
```

---

### Task 4: cmd/env.go

**Files:**
- Create: `cmd/env.go`

**Command:** `bb env list`

- [ ] **Step 1: Create cmd/env.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/env.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage deployment environments",
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deployment environments in the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		envs, err := client.Environments(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(envs, func() {
			if len(envs) == 0 {
				fmt.Println("No environments found.")
				return
			}
			for _, e := range envs {
				lock := ""
				if e.Lock.Name == "LOCKED" {
					lock = "  [LOCKED]"
				}
				fmt.Printf("%-40s  %-12s  %s%s\n",
					e.UUID, e.EnvironmentType.Name, e.Name, lock)
			}
		})
	},
}

func init() {
	envCmd.AddCommand(envListCmd)
	rootCmd.AddCommand(envCmd)
}
```

- [ ] **Step 2: Build to verify compilation**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go build -o bb .
```

Expected: No output (binary built).

- [ ] **Step 3: Smoke tests**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
./bb env --help
./bb env list --help
./bb --help | grep env
```

Expected for `./bb env --help`:
```
Manage deployment environments

Usage:
  bb env [command]

Available Commands:
  list        List deployment environments in the repository
```

Expected for `./bb --help | grep env`: `env        Manage deployment environments`

- [ ] **Step 4: Clean up and commit**

```bash
rm bb
git add cmd/env.go
git commit -m "feat: add bb env commands (list)"
```

---

### Task 5: cmd/deployment.go

**Files:**
- Create: `cmd/deployment.go`

**Command:** `bb deployment list`

Text output columns: state/status, environment UUID (truncated to 12 chars for readability), commit hash (first 8 chars), date (first 10 chars of `last_update_time`).

- [ ] **Step 1: Create cmd/deployment.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/deployment.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var deploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "View repository deployments",
}

var deploymentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent deployments, newest first",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		deployments, err := client.Deployments(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(deployments, func() {
			if len(deployments) == 0 {
				fmt.Println("No deployments found.")
				return
			}
			for _, d := range deployments {
				status := d.State.Name
				if d.State.Status != nil {
					status += "/" + d.State.Status.Name
				}
				commit := ""
				if d.Deployable.Commit != nil && len(d.Deployable.Commit.Hash) >= 8 {
					commit = d.Deployable.Commit.Hash[:8]
				}
				date := ""
				if len(d.LastUpdateTime) >= 10 {
					date = d.LastUpdateTime[:10]
				}
				envUUID := truncate(d.Environment.UUID, 38)
				fmt.Printf("%-38s  %-30s  %-8s  %s\n",
					envUUID, status, commit, date)
			}
		})
	},
}

func init() {
	deploymentCmd.AddCommand(deploymentListCmd)
	rootCmd.AddCommand(deploymentCmd)
}
```

- [ ] **Step 2: Check that `truncate` helper exists**

The `truncate` helper is used in `cmd/commit.go`. Verify it's available to this package:

```bash
grep -n "func truncate" /Users/jay/code/cli-bitbucket-cloud/cmd/
```

If `truncate` is defined somewhere in the `cmd` package (any `.go` file in `cmd/`), it's available. If it doesn't exist, replace `truncate(d.Environment.UUID, 38)` with a simple slice: `d.Environment.UUID` (environment UUIDs from Bitbucket have a fixed `{uuid}` format that fits on one line).

- [ ] **Step 3: Build to verify compilation**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go build -o bb .
```

Expected: No output (binary built).

- [ ] **Step 4: Smoke tests**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
./bb deployment --help
./bb deployment list --help
./bb --help | grep deployment
```

Expected for `./bb deployment --help`:
```
View repository deployments

Usage:
  bb deployment [command]

Available Commands:
  list        List recent deployments, newest first
```

Expected for `./bb --help | grep deployment`: `deployment  View repository deployments`

- [ ] **Step 5: Clean up and commit**

```bash
rm bb
git add cmd/deployment.go
git commit -m "feat: add bb deployment commands (list)"
```

---

## Self-Review

**Spec coverage:**
- `bb env list` → Task 4 `envListCmd` ✅
- `bb deployment list` → Task 5 `deploymentListCmd` ✅
- `EnvironmentResource.List` → Task 2 ✅
- `DeploymentResource.List` → Task 3 ✅
- `Environment`, `EnvironmentType`, `EnvironmentLock` types → Task 1 ✅
- `Deployment`, `DeploymentState`, `DeploymentStatus`, `DeploymentEnvRef`, `Deployable`, `DeployableCommit`, `DeployablePipeline` types → Task 1 ✅
- `Environments()` accessor on `*Client` → Task 2 Step 4 ✅
- `Deployments()` accessor on `*Client` → Task 3 Step 4 ✅

**Placeholder scan:** No TBDs. All code blocks are complete and runnable. Task 5 Step 2 includes a conditional note about `truncate` — this is a verification step, not a placeholder.

**Type consistency:**
- `client.Environments("testws", "testrepo")` returns `*EnvironmentResource` — used consistently in test and cmd ✅
- `client.Deployments("testws", "testrepo")` returns `*DeploymentResource` — used consistently in test and cmd ✅
- `Deployment.State` is `DeploymentState` with optional `*DeploymentStatus` — matches test fixture and cmd output logic ✅
- `Deployment.Deployable.Commit` is `*DeployableCommit` (pointer, may be nil) — nil-checked in cmd output ✅
- `Deployment.Environment` is `DeploymentEnvRef{UUID string}` — accessed as `d.Environment.UUID` in cmd ✅
