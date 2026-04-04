# BB CLI Extended API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the `bb` CLI with pipeline monitoring, branch management, commit history, file reading, user info, PR activity/statuses, and repository listing — in priority order.

**Architecture:** Each new domain follows the established pattern: a `*Resource` struct in `pkg/bitbucket/` with httptest-backed tests, a constructor accessor on `*Client` in `client.go`, and cobra subcommands in `cmd/`. New types go in `types.go`. Commands use package-level flag vars and register via `init()`. The `truncate()` helper is already defined in `cmd/comment.go` and is available across all `package cmd` files.

**Tech Stack:** Go 1.25, `github.com/spf13/cobra v1.10.2`, `net/http/httptest` (stdlib), `encoding/json`, `gopkg.in/yaml.v3 v3.0.1`

---

## File Structure

**New files:**

| File | Responsibility |
|------|---------------|
| `pkg/bitbucket/pipeline.go` | PipelineResource: List, Get, Trigger, Stop, Steps, Log |
| `pkg/bitbucket/pipeline_test.go` | Tests for PipelineResource |
| `pkg/bitbucket/branch.go` | BranchResource: List, Create, Delete |
| `pkg/bitbucket/branch_test.go` | Tests for BranchResource |
| `pkg/bitbucket/commit.go` | CommitResource: List, Get, File |
| `pkg/bitbucket/commit_test.go` | Tests for CommitResource |
| `pkg/bitbucket/user.go` | UserResource: Me |
| `pkg/bitbucket/user_test.go` | Tests for UserResource |
| `pkg/bitbucket/repo.go` | RepoResource: List |
| `pkg/bitbucket/repo_test.go` | Tests for RepoResource |
| `cmd/pipeline.go` | `bb pipeline {list,get,trigger,stop,steps,log}` |
| `cmd/branch.go` | `bb branch {list,create,delete}` |
| `cmd/commit.go` | `bb commit {list,get}` and `bb file get` |
| `cmd/user.go` | `bb user me` |
| `cmd/repo.go` | `bb repo list` |

**Modified files:**

| File | Change |
|------|--------|
| `pkg/bitbucket/types.go` | Append Pipeline*, Branch*, Commit*, User, Repo, Activity*, PRStatus types |
| `pkg/bitbucket/client.go` | Add Pipelines(), Branches(), Commits(), User(), Repos() accessors |
| `pkg/bitbucket/pr.go` | Add Activity() and Statuses() methods to PRResource |
| `pkg/bitbucket/pr_test.go` | Add TestPRs_Activity and TestPRs_Statuses |
| `cmd/pr.go` | Append prActivityCmd, prStatusesCmd, their flag vars, and init() wiring |

---

### Task 1: Add new types to pkg/bitbucket/types.go

**Files:**
- Modify: `pkg/bitbucket/types.go`

- [ ] **Step 1: Append new types after the last existing type (AddCommentInput)**

Open `pkg/bitbucket/types.go` and append the following block at the end of the file:

```go
// Pipeline types

type Pipeline struct {
	UUID        string         `json:"uuid"`
	BuildNumber int            `json:"build_number"`
	State       PipelineState  `json:"state"`
	Target      PipelineTarget `json:"target"`
	CreatedOn   string         `json:"created_on"`
	CompletedOn string         `json:"completed_on"`
}

type PipelineState struct {
	Name   string          `json:"name"`
	Result *PipelineResult `json:"result,omitempty"`
}

type PipelineResult struct {
	Name string `json:"name"`
}

type PipelineTarget struct {
	RefType string          `json:"ref_type"`
	RefName string          `json:"ref_name"`
	Commit  *PipelineCommit `json:"commit,omitempty"`
}

type PipelineCommit struct {
	Hash string `json:"hash"`
}

type PipelineStep struct {
	UUID        string        `json:"uuid"`
	Name        string        `json:"name"`
	State       PipelineState `json:"state"`
	StartedOn   string        `json:"started_on"`
	CompletedOn string        `json:"completed_on"`
}

type TriggerPipelineInput struct {
	Target TriggerTarget `json:"target"`
}

type TriggerTarget struct {
	RefType string `json:"ref_type"`
	Type    string `json:"type"`
	RefName string `json:"ref_name"`
}

// Branch types

type Branch struct {
	Name   string       `json:"name"`
	Target BranchTarget `json:"target"`
	Links  Links        `json:"links"`
}

type BranchTarget struct {
	Hash string `json:"hash"`
}

type CreateBranchInput struct {
	Name   string       `json:"name"`
	Target BranchTarget `json:"target"`
}

// Commit types

type Commit struct {
	Hash    string         `json:"hash"`
	Date    string         `json:"date"`
	Message string         `json:"message"`
	Author  CommitAuthor   `json:"author"`
	Parents []CommitParent `json:"parents"`
}

type CommitAuthor struct {
	Raw  string `json:"raw"`
	User *Actor `json:"user,omitempty"`
}

type CommitParent struct {
	Hash string `json:"hash"`
}

// User type

type User struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
	Links       Links  `json:"links"`
}

// Repo type

type Repo struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	FullName    string `json:"full_name"`
	Links       Links  `json:"links"`
}

// PR Activity types

type Activity struct {
	Comment  *Comment  `json:"comment,omitempty"`
	Approval *Approval `json:"approval,omitempty"`
	Update   *PRUpdate `json:"update,omitempty"`
}

type Approval struct {
	User Actor  `json:"user"`
	Date string `json:"date"`
}

type PRUpdate struct {
	State  string `json:"state"`
	Author Actor  `json:"author"`
	Date   string `json:"date"`
}

// PRStatus type

type PRStatus struct {
	State       string `json:"state"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	CreatedOn   string `json:"created_on"`
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
git commit -m "feat: add pipeline, branch, commit, user, repo, activity, status types"
```

---

### Task 2: PipelineResource

**Files:**
- Create: `pkg/bitbucket/pipeline_test.go`
- Create: `pkg/bitbucket/pipeline.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoints:**
- `GET /repositories/{ws}/{repo}/pipelines/?sort=-created_on&pagelen=25`
- `GET /repositories/{ws}/{repo}/pipelines/{uuid}`
- `POST /repositories/{ws}/{repo}/pipelines/`
- `POST /repositories/{ws}/{repo}/pipelines/{uuid}/stopPipeline`
- `GET /repositories/{ws}/{repo}/pipelines/{uuid}/steps/`
- `GET /repositories/{ws}/{repo}/pipelines/{uuid}/steps/{stepUUID}/log` — returns raw text

Note: `url.PathEscape` is used for UUIDs in paths (handles curly braces: `{abc}` → `%7Babc%7D`). The `do` method handles 204 No Content correctly — it returns empty bytes with nil error, which the Stop method ignores.

- [ ] **Step 1: Create pipeline_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/pipeline_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestPipelines_List(t *testing.T) {
	pipelines := []bitbucket.Pipeline{
		{
			UUID:        "{abc-123}",
			BuildNumber: 42,
			State:       bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "SUCCESSFUL"}},
			Target:      bitbucket.PipelineTarget{RefType: "branch", RefName: "main"},
			CreatedOn:   "2024-01-15T10:00:00+00:00",
		},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"values": pipelines})
	}))
	got, err := client.Pipelines("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].BuildNumber != 42 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestPipelines_Get(t *testing.T) {
	pipeline := bitbucket.Pipeline{UUID: "{abc-123}", BuildNumber: 42}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(pipeline)
	}))
	got, err := client.Pipelines("testws", "testrepo").Get(context.Background(), "{abc-123}")
	if err != nil {
		t.Fatal(err)
	}
	if got.BuildNumber != 42 {
		t.Errorf("expected build 42, got %d", got.BuildNumber)
	}
}

func TestPipelines_Trigger(t *testing.T) {
	pipeline := bitbucket.Pipeline{UUID: "{new-uuid}", BuildNumber: 43}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		target, _ := body["target"].(map[string]any)
		if target["ref_name"] != "main" {
			t.Errorf("expected ref_name=main, got %v", target["ref_name"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(pipeline)
	}))
	got, err := client.Pipelines("testws", "testrepo").Trigger(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if got.BuildNumber != 43 {
		t.Errorf("expected build 43, got %d", got.BuildNumber)
	}
}

func TestPipelines_Stop(t *testing.T) {
	stopped := false
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		stopped = true
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Pipelines("testws", "testrepo").Stop(context.Background(), "{abc-123}")
	if err != nil {
		t.Fatal(err)
	}
	if !stopped {
		t.Error("stop handler was not called")
	}
}

func TestPipelines_Steps(t *testing.T) {
	steps := []bitbucket.PipelineStep{
		{UUID: "{step-1}", Name: "build", State: bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "SUCCESSFUL"}}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"values": steps})
	}))
	got, err := client.Pipelines("testws", "testrepo").Steps(context.Background(), "{abc-123}")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "build" {
		t.Errorf("unexpected steps: %+v", got)
	}
}

func TestPipelines_Log(t *testing.T) {
	logText := "Step 1: Building...\nStep 2: Done.\n"
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(logText))
	}))
	got, err := client.Pipelines("testws", "testrepo").Log(context.Background(), "{abc-123}", "{step-1}")
	if err != nil {
		t.Fatal(err)
	}
	if got != logText {
		t.Errorf("expected %q, got %q", logText, got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestPipelines -v 2>&1 | head -5
```

Expected: Compilation error mentioning `client.Pipelines undefined`.

- [ ] **Step 3: Create pipeline.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/pipeline.go`:

```go
package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// PipelineResource provides operations on repository pipelines.
type PipelineResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *PipelineResource) basePath() string {
	return fmt.Sprintf("%s/pipelines/", repoPath(r.workspace, r.repo))
}

// List returns the most recent pipelines, newest first.
func (r *PipelineResource) List(ctx context.Context) ([]Pipeline, error) {
	q := url.Values{"sort": {"-created_on"}, "pagelen": {"25"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Pipeline]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Get returns a single pipeline by UUID.
func (r *PipelineResource) Get(ctx context.Context, pipelineUUID string) (Pipeline, error) {
	path := fmt.Sprintf("%s%s", r.basePath(), url.PathEscape(pipelineUUID))
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Pipeline{}, err
	}
	return decode[Pipeline](data)
}

// Trigger starts a new pipeline on the given branch.
func (r *PipelineResource) Trigger(ctx context.Context, branch string) (Pipeline, error) {
	input := TriggerPipelineInput{
		Target: TriggerTarget{
			RefType: "branch",
			Type:    "pipeline_ref_target",
			RefName: branch,
		},
	}
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Pipeline{}, err
	}
	return decode[Pipeline](data)
}

// Stop requests cancellation of a running pipeline.
func (r *PipelineResource) Stop(ctx context.Context, pipelineUUID string) error {
	path := fmt.Sprintf("%s%s/stopPipeline", r.basePath(), url.PathEscape(pipelineUUID))
	_, err := r.client.do(ctx, "POST", path, nil, nil)
	return err
}

// Steps returns all steps of a pipeline.
func (r *PipelineResource) Steps(ctx context.Context, pipelineUUID string) ([]PipelineStep, error) {
	path := fmt.Sprintf("%s%s/steps/", r.basePath(), url.PathEscape(pipelineUUID))
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[PipelineStep]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Log returns the raw log output for a pipeline step (plain text, not JSON).
func (r *PipelineResource) Log(ctx context.Context, pipelineUUID, stepUUID string) (string, error) {
	path := fmt.Sprintf("%s%s/steps/%s/log",
		r.basePath(), url.PathEscape(pipelineUUID), url.PathEscape(stepUUID))
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

- [ ] **Step 4: Add Pipelines() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after the existing `Tasks(...)` method:

```go
// Pipelines returns a resource for pipeline operations on the given repo.
func (c *Client) Pipelines(workspace, repo string) *PipelineResource {
	return &PipelineResource{client: c, workspace: workspace, repo: repo}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestPipelines -v
```

Expected: All 6 `TestPipelines_*` tests PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/pipeline.go pkg/bitbucket/pipeline_test.go pkg/bitbucket/client.go
git commit -m "feat: add PipelineResource (list, get, trigger, stop, steps, log)"
```

---

### Task 3: BranchResource

**Files:**
- Create: `pkg/bitbucket/branch_test.go`
- Create: `pkg/bitbucket/branch.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoints:**
- `GET /repositories/{ws}/{repo}/refs/branches?pagelen=50`
- `POST /repositories/{ws}/{repo}/refs/branches` — body: `{"name":"...", "target":{"hash":"..."}}`
- `DELETE /repositories/{ws}/{repo}/refs/branches/{name}` — returns 204 No Content

Note: `target.hash` accepts a commit hash OR a branch name — Bitbucket resolves branch names automatically. Branch names with `/` (e.g. `feature/foo`) must be `url.PathEscape`d for DELETE.

- [ ] **Step 1: Create branch_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/branch_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestBranches_List(t *testing.T) {
	branches := []bitbucket.Branch{
		{Name: "main", Target: bitbucket.BranchTarget{Hash: "abc123"}},
		{Name: "develop", Target: bitbucket.BranchTarget{Hash: "def456"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/refs/branches" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"values": branches})
	}))
	got, err := client.Branches("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "main" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestBranches_Create(t *testing.T) {
	created := bitbucket.Branch{Name: "feature/new", Target: bitbucket.BranchTarget{Hash: "abc123"}}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "feature/new" {
			t.Errorf("expected name=feature/new, got %v", body["name"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	}))
	got, err := client.Branches("testws", "testrepo").Create(context.Background(), "feature/new", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "feature/new" {
		t.Errorf("expected feature/new, got %s", got.Name)
	}
}

func TestBranches_Delete(t *testing.T) {
	deleted := false
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Branches("testws", "testrepo").Delete(context.Background(), "feature/old")
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Error("delete handler was not called")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestBranches -v 2>&1 | head -5
```

Expected: Compilation error mentioning `client.Branches undefined`.

- [ ] **Step 3: Create branch.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/branch.go`:

```go
package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// BranchResource provides operations on repository branches.
type BranchResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *BranchResource) basePath() string {
	return fmt.Sprintf("%s/refs/branches", repoPath(r.workspace, r.repo))
}

// List returns all branches in the repository.
func (r *BranchResource) List(ctx context.Context) ([]Branch, error) {
	q := url.Values{"pagelen": {"50"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Branch]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Create creates a new branch from the given commit hash or branch name.
func (r *BranchResource) Create(ctx context.Context, name, target string) (Branch, error) {
	input := CreateBranchInput{
		Name:   name,
		Target: BranchTarget{Hash: target},
	}
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Branch{}, err
	}
	return decode[Branch](data)
}

// Delete removes a branch from the repository.
func (r *BranchResource) Delete(ctx context.Context, name string) error {
	path := fmt.Sprintf("%s/%s", r.basePath(), url.PathEscape(name))
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
```

- [ ] **Step 4: Add Branches() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after `Pipelines()`:

```go
// Branches returns a resource for branch operations on the given repo.
func (c *Client) Branches(workspace, repo string) *BranchResource {
	return &BranchResource{client: c, workspace: workspace, repo: repo}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestBranches -v
```

Expected: All 3 `TestBranches_*` tests PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/branch.go pkg/bitbucket/branch_test.go pkg/bitbucket/client.go
git commit -m "feat: add BranchResource (list, create, delete)"
```

---

### Task 4: CommitResource

**Files:**
- Create: `pkg/bitbucket/commit_test.go`
- Create: `pkg/bitbucket/commit.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoints:**
- `GET /repositories/{ws}/{repo}/commits/{branch}?pagelen=25` — list commits on a branch
- `GET /repositories/{ws}/{repo}/commit/{hash}` — singular "commit", not "commits"
- `GET /repositories/{ws}/{repo}/src/{ref}/{path}` — raw file content (not JSON)

- [ ] **Step 1: Create commit_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/commit_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestCommits_List(t *testing.T) {
	commits := []bitbucket.Commit{
		{Hash: "abc123", Message: "Fix bug", Author: bitbucket.CommitAuthor{Raw: "Jay <jay@example.com>"}, Date: "2024-01-15T10:00:00+00:00"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/commits/main" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"values": commits})
	}))
	got, err := client.Commits("testws", "testrepo").List(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Hash != "abc123" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestCommits_Get(t *testing.T) {
	commit := bitbucket.Commit{Hash: "abc123", Message: "Fix bug"}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories/testws/testrepo/commit/abc123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(commit)
	}))
	got, err := client.Commits("testws", "testrepo").Get(context.Background(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash != "abc123" {
		t.Errorf("expected abc123, got %s", got.Hash)
	}
}

func TestCommits_File(t *testing.T) {
	fileContent := "package main\n\nfunc main() {}\n"
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories/testws/testrepo/src/main/main.go" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(fileContent))
	}))
	got, err := client.Commits("testws", "testrepo").File(context.Background(), "main", "main.go")
	if err != nil {
		t.Fatal(err)
	}
	if got != fileContent {
		t.Errorf("expected %q, got %q", fileContent, got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestCommits -v 2>&1 | head -5
```

Expected: Compilation error mentioning `client.Commits undefined`.

- [ ] **Step 3: Create commit.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/commit.go`:

```go
package bitbucket

import (
	"context"
	"fmt"
)

// CommitResource provides operations on commits and file contents within a repository.
type CommitResource struct {
	client    *Client
	workspace string
	repo      string
}

// List returns commits on a branch, newest first.
func (r *CommitResource) List(ctx context.Context, branch string) ([]Commit, error) {
	path := fmt.Sprintf("%s/commits/%s", repoPath(r.workspace, r.repo), branch)
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Commit]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Get returns a single commit by hash.
// Note: Bitbucket uses the singular "/commit/" path for single commit lookup.
func (r *CommitResource) Get(ctx context.Context, hash string) (Commit, error) {
	path := fmt.Sprintf("%s/commit/%s", repoPath(r.workspace, r.repo), hash)
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Commit{}, err
	}
	return decode[Commit](data)
}

// File returns the raw content of a file at the given ref (branch, tag, or commit hash).
// The response is plain text, not JSON.
func (r *CommitResource) File(ctx context.Context, ref, filePath string) (string, error) {
	apiPath := fmt.Sprintf("%s/src/%s/%s", repoPath(r.workspace, r.repo), ref, filePath)
	data, err := r.client.do(ctx, "GET", apiPath, nil, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

- [ ] **Step 4: Add Commits() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after `Branches()`:

```go
// Commits returns a resource for commit history and file access on the given repo.
func (c *Client) Commits(workspace, repo string) *CommitResource {
	return &CommitResource{client: c, workspace: workspace, repo: repo}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestCommits -v
```

Expected: All 3 `TestCommits_*` tests PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/commit.go pkg/bitbucket/commit_test.go pkg/bitbucket/client.go
git commit -m "feat: add CommitResource (list, get, file)"
```

---

### Task 5: UserResource

**Files:**
- Create: `pkg/bitbucket/user_test.go`
- Create: `pkg/bitbucket/user.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoint:**
- `GET /user` — no workspace or repo required; returns the authenticated user's profile

- [ ] **Step 1: Create user_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/user_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestUser_Me(t *testing.T) {
	user := bitbucket.User{
		AccountID:   "abc123",
		DisplayName: "Jay Madore",
		Nickname:    "jay",
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/user" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(user)
	}))
	got, err := client.User().Me(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != "Jay Madore" || got.Nickname != "jay" {
		t.Errorf("unexpected user: %+v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestUser -v 2>&1 | head -5
```

Expected: Compilation error mentioning `client.User undefined`.

- [ ] **Step 3: Create user.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/user.go`:

```go
package bitbucket

import "context"

// UserResource provides operations on the authenticated user.
type UserResource struct {
	client *Client
}

// Me returns the authenticated user's profile.
func (r *UserResource) Me(ctx context.Context) (User, error) {
	data, err := r.client.do(ctx, "GET", "/user", nil, nil)
	if err != nil {
		return User{}, err
	}
	return decode[User](data)
}
```

- [ ] **Step 4: Add User() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after `Commits()`:

```go
// User returns a resource for authenticated user operations.
func (c *Client) User() *UserResource {
	return &UserResource{client: c}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestUser -v
```

Expected: `TestUser_Me` PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/user.go pkg/bitbucket/user_test.go pkg/bitbucket/client.go
git commit -m "feat: add UserResource (me)"
```

---

### Task 6: RepoResource

**Files:**
- Create: `pkg/bitbucket/repo_test.go`
- Create: `pkg/bitbucket/repo.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoint:**
- `GET /repositories/{workspace}?role=member&pagelen=25` — lists all repos the authenticated user can access in the workspace

- [ ] **Step 1: Create repo_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/repo_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestRepos_List(t *testing.T) {
	repos := []bitbucket.Repo{
		{Slug: "whosoncall", Name: "Who's On Call", IsPrivate: true},
		{Slug: "skill-java", Name: "Java Skills", IsPrivate: true},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"values": repos})
	}))
	got, err := client.Repos("testws").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Slug != "whosoncall" {
		t.Errorf("unexpected result: %+v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestRepos -v 2>&1 | head -5
```

Expected: Compilation error mentioning `client.Repos undefined`.

- [ ] **Step 3: Create repo.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/repo.go`:

```go
package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// RepoResource provides operations on repositories within a workspace.
type RepoResource struct {
	client    *Client
	workspace string
}

// List returns all repositories the authenticated user has access to in the workspace.
func (r *RepoResource) List(ctx context.Context) ([]Repo, error) {
	path := fmt.Sprintf("/repositories/%s", r.workspace)
	q := url.Values{"role": {"member"}, "pagelen": {"25"}}
	data, err := r.client.do(ctx, "GET", path, nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Repo]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
```

- [ ] **Step 4: Add Repos() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after `User()`:

```go
// Repos returns a resource for listing repositories in a workspace.
func (c *Client) Repos(workspace string) *RepoResource {
	return &RepoResource{client: c, workspace: workspace}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestRepos -v
```

Expected: `TestRepos_List` PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/repo.go pkg/bitbucket/repo_test.go pkg/bitbucket/client.go
git commit -m "feat: add RepoResource (list)"
```

---

### Task 7: PR Activity and Statuses

**Files:**
- Modify: `pkg/bitbucket/pr.go`
- Modify: `pkg/bitbucket/pr_test.go`

**Bitbucket API endpoints:**
- `GET /repositories/{ws}/{repo}/pullrequests/{prID}/activity`
- `GET /repositories/{ws}/{repo}/pullrequests/{prID}/statuses`

The existing `PRResource` has `prPath(prID)` which returns `/repositories/{ws}/{repo}/pullrequests/{prID}`. New methods append `/activity` and `/statuses` to that path.

- [ ] **Step 1: Add tests to pr_test.go**

Append the following two test functions to the end of `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/pr_test.go`:

```go
func TestPRs_Activity(t *testing.T) {
	activities := []bitbucket.Activity{
		{Approval: &bitbucket.Approval{User: bitbucket.Actor{DisplayName: "Jane"}, Date: "2024-01-15T10:00:00+00:00"}},
		{Comment: &bitbucket.Comment{ID: 1, Content: bitbucket.Content{Raw: "LGTM"}, User: bitbucket.Actor{DisplayName: "Bob"}}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories/testws/testrepo/pullrequests/42/activity" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"values": activities})
	}))
	got, err := client.PRs("testws", "testrepo").Activity(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(got))
	}
	if got[0].Approval == nil || got[0].Approval.User.DisplayName != "Jane" {
		t.Errorf("unexpected activity[0]: %+v", got[0])
	}
}

func TestPRs_Statuses(t *testing.T) {
	statuses := []bitbucket.PRStatus{
		{State: "SUCCESSFUL", Key: "bitbucket-pipelines", Name: "Build", Description: "Pipeline passed"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories/testws/testrepo/pullrequests/42/statuses" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"values": statuses})
	}))
	got, err := client.PRs("testws", "testrepo").Statuses(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].State != "SUCCESSFUL" {
		t.Errorf("unexpected statuses: %+v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run "TestPRs_Activity|TestPRs_Statuses" -v 2>&1 | head -10
```

Expected: Compilation error — `Activity` and `Statuses` methods undefined on `PRResource`.

- [ ] **Step 3: Add Activity() and Statuses() to pr.go**

Append the following two methods to the end of `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/pr.go`:

```go
// Activity returns the full activity timeline for a pull request.
func (r *PRResource) Activity(ctx context.Context, prID int) ([]Activity, error) {
	path := fmt.Sprintf("%s/activity", r.prPath(prID))
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Activity]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Statuses returns the build statuses associated with the pull request's source commit.
func (r *PRResource) Statuses(ctx context.Context, prID int) ([]PRStatus, error) {
	path := fmt.Sprintf("%s/statuses", r.prPath(prID))
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[PRStatus]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
```

- [ ] **Step 4: Run all tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -v
```

Expected: All tests PASS. Count: 15 existing + 2 new = 17 total.

- [ ] **Step 5: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/pr.go pkg/bitbucket/pr_test.go
git commit -m "feat: add PRResource.Activity and PRResource.Statuses"
```

---

### Task 8: cmd/pipeline.go

**Files:**
- Create: `cmd/pipeline.go`

**Commands:** `bb pipeline list`, `bb pipeline get`, `bb pipeline trigger`, `bb pipeline stop`, `bb pipeline steps`, `bb pipeline log`

Pattern notes:
- Follow the existing style from `cmd/pr.go`: package-level flag vars, cobra command vars, `init()` for wiring.
- `workspaceAndRepo()` from `cmd/root.go` is available across `package cmd`.
- `bb pipeline log` always prints raw text (like `bb pr diff`) — ignores `--format`.
- All commands use `context.Background()` (matching existing pr.go pattern).

- [ ] **Step 1: Create cmd/pipeline.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/pipeline.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Manage Bitbucket Pipelines",
}

var pipelineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent pipelines, newest first",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		pipelines, err := client.Pipelines(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(pipelines, func() {
			if len(pipelines) == 0 {
				fmt.Println("No pipelines found.")
				return
			}
			for _, p := range pipelines {
				result := ""
				if p.State.Result != nil {
					result = "/" + p.State.Result.Name
				}
				date := ""
				if len(p.CreatedOn) >= 10 {
					date = p.CreatedOn[:10]
				}
				fmt.Printf("#%-4d  %-30s  %-20s  %s\n",
					p.BuildNumber, p.State.Name+result, p.Target.RefName, date)
			}
		})
	},
}

var pipelineGetUUID string

var pipelineGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get pipeline details",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		p, err := client.Pipelines(ws, repo).Get(context.Background(), pipelineGetUUID)
		if err != nil {
			return err
		}
		return printOutput(p, func() {
			result := ""
			if p.State.Result != nil {
				result = " (" + p.State.Result.Name + ")"
			}
			commit := ""
			if p.Target.Commit != nil {
				commit = p.Target.Commit.Hash
			}
			fmt.Printf("Pipeline #%d\nUUID:      %s\nState:     %s%s\nBranch:    %s\nCommit:    %s\nCreated:   %s\nCompleted: %s\n",
				p.BuildNumber, p.UUID, p.State.Name, result,
				p.Target.RefName, commit, p.CreatedOn, p.CompletedOn)
		})
	},
}

var pipelineTriggerBranch string

var pipelineTriggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Trigger a new pipeline on a branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		p, err := client.Pipelines(ws, repo).Trigger(context.Background(), pipelineTriggerBranch)
		if err != nil {
			return err
		}
		return printOutput(p, func() {
			fmt.Printf("Pipeline #%d triggered on branch '%s'\nUUID: %s\n",
				p.BuildNumber, pipelineTriggerBranch, p.UUID)
		})
	},
}

var pipelineStopUUID string

var pipelineStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Pipelines(ws, repo).Stop(context.Background(), pipelineStopUUID); err != nil {
			return err
		}
		fmt.Printf("Pipeline %s stopped.\n", pipelineStopUUID)
		return nil
	},
}

var pipelineStepsUUID string

var pipelineStepsCmd = &cobra.Command{
	Use:   "steps",
	Short: "List steps of a pipeline",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		steps, err := client.Pipelines(ws, repo).Steps(context.Background(), pipelineStepsUUID)
		if err != nil {
			return err
		}
		return printOutput(steps, func() {
			if len(steps) == 0 {
				fmt.Println("No steps found.")
				return
			}
			for _, s := range steps {
				result := ""
				if s.State.Result != nil {
					result = "/" + s.State.Result.Name
				}
				fmt.Printf("%-40s  %-20s  %s%s\n",
					s.UUID, s.Name, s.State.Name, result)
			}
		})
	},
}

var (
	pipelineLogPipelineUUID string
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
		log, err := client.Pipelines(ws, repo).Log(context.Background(), pipelineLogPipelineUUID, pipelineLogStepUUID)
		if err != nil {
			return err
		}
		fmt.Print(log)
		return nil
	},
}

func init() {
	pipelineGetCmd.Flags().StringVar(&pipelineGetUUID, "pipeline-uuid", "", "pipeline UUID (required)")
	pipelineGetCmd.MarkFlagRequired("pipeline-uuid")

	pipelineTriggerCmd.Flags().StringVar(&pipelineTriggerBranch, "branch", "", "branch to trigger pipeline on (required)")
	pipelineTriggerCmd.MarkFlagRequired("branch")

	pipelineStopCmd.Flags().StringVar(&pipelineStopUUID, "pipeline-uuid", "", "pipeline UUID (required)")
	pipelineStopCmd.MarkFlagRequired("pipeline-uuid")

	pipelineStepsCmd.Flags().StringVar(&pipelineStepsUUID, "pipeline-uuid", "", "pipeline UUID (required)")
	pipelineStepsCmd.MarkFlagRequired("pipeline-uuid")

	pipelineLogCmd.Flags().StringVar(&pipelineLogPipelineUUID, "pipeline-uuid", "", "pipeline UUID (required)")
	pipelineLogCmd.Flags().StringVar(&pipelineLogStepUUID, "step-uuid", "", "step UUID (required)")
	pipelineLogCmd.MarkFlagRequired("pipeline-uuid")
	pipelineLogCmd.MarkFlagRequired("step-uuid")

	pipelineCmd.AddCommand(pipelineListCmd, pipelineGetCmd, pipelineTriggerCmd,
		pipelineStopCmd, pipelineStepsCmd, pipelineLogCmd)
	rootCmd.AddCommand(pipelineCmd)
}
```

- [ ] **Step 2: Build to verify compilation**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go build -o bb .
```

Expected: No output (binary built).

- [ ] **Step 3: Smoke test**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && ./bb pipeline --help
```

Expected output includes: `list`, `get`, `trigger`, `stop`, `steps`, `log` in Available Commands.

- [ ] **Step 4: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add cmd/pipeline.go
git commit -m "feat: add bb pipeline commands (list, get, trigger, stop, steps, log)"
```

---

### Task 9: cmd/branch.go and cmd/commit.go

**Files:**
- Create: `cmd/branch.go`
- Create: `cmd/commit.go`

**Commands:**
- `bb branch list` — list all branches
- `bb branch create --name NAME --from REF` — create branch from branch name or commit hash
- `bb branch delete --name NAME` — delete branch
- `bb commit list --branch BRANCH` — list commits on a branch
- `bb commit get --hash HASH` — get a single commit
- `bb file get --ref REF --path PATH` — get raw file contents

Note: `truncate()` is defined in `cmd/comment.go` and is available to all files in `package cmd` — no import needed.

- [ ] **Step 1: Create cmd/branch.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/branch.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Manage repository branches",
}

var branchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List branches in the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		branches, err := client.Branches(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(branches, func() {
			if len(branches) == 0 {
				fmt.Println("No branches found.")
				return
			}
			for _, b := range branches {
				fmt.Printf("%-40s  %s\n", b.Name, b.Target.Hash)
			}
		})
	},
}

var (
	branchCreateName string
	branchCreateFrom string
)

var branchCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new branch from a branch name or commit hash",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		branch, err := client.Branches(ws, repo).Create(context.Background(), branchCreateName, branchCreateFrom)
		if err != nil {
			return err
		}
		return printOutput(branch, func() {
			fmt.Printf("Branch '%s' created at %s\n", branch.Name, branch.Target.Hash)
		})
	},
}

var branchDeleteName string

var branchDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Branches(ws, repo).Delete(context.Background(), branchDeleteName); err != nil {
			return err
		}
		fmt.Printf("Branch '%s' deleted.\n", branchDeleteName)
		return nil
	},
}

func init() {
	branchCreateCmd.Flags().StringVar(&branchCreateName, "name", "", "name for the new branch (required)")
	branchCreateCmd.Flags().StringVar(&branchCreateFrom, "from", "", "branch name or commit hash to branch from (required)")
	branchCreateCmd.MarkFlagRequired("name")
	branchCreateCmd.MarkFlagRequired("from")

	branchDeleteCmd.Flags().StringVar(&branchDeleteName, "name", "", "branch name to delete (required)")
	branchDeleteCmd.MarkFlagRequired("name")

	branchCmd.AddCommand(branchListCmd, branchCreateCmd, branchDeleteCmd)
	rootCmd.AddCommand(branchCmd)
}
```

- [ ] **Step 2: Create cmd/commit.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/commit.go`:

```go
package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Browse commit history",
}

var commitListBranch string

var commitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List commits on a branch, newest first",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		commits, err := client.Commits(ws, repo).List(context.Background(), commitListBranch)
		if err != nil {
			return err
		}
		return printOutput(commits, func() {
			if len(commits) == 0 {
				fmt.Println("No commits found.")
				return
			}
			for _, c := range commits {
				date := c.Date
				if len(date) >= 10 {
					date = date[:10]
				}
				msg := c.Message
				if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
					msg = msg[:idx]
				}
				fmt.Printf("%s  %s  %-30s  %s\n",
					c.Hash[:8], date, truncate(c.Author.Raw, 30), truncate(msg, 72))
			}
		})
	},
}

var commitGetHash string

var commitGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details of a single commit",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		c, err := client.Commits(ws, repo).Get(context.Background(), commitGetHash)
		if err != nil {
			return err
		}
		return printOutput(c, func() {
			parents := make([]string, len(c.Parents))
			for i, p := range c.Parents {
				parents[i] = p.Hash[:8]
			}
			fmt.Printf("Hash:    %s\nDate:    %s\nAuthor:  %s\nMessage: %s\nParents: %s\n",
				c.Hash, c.Date, c.Author.Raw, c.Message, strings.Join(parents, ", "))
		})
	},
}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Read file contents from the repository",
}

var (
	fileGetRef  string
	fileGetPath string
)

var fileGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get raw file contents at a ref (branch, tag, or commit hash)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		content, err := client.Commits(ws, repo).File(context.Background(), fileGetRef, fileGetPath)
		if err != nil {
			return err
		}
		fmt.Print(content)
		return nil
	},
}

func init() {
	commitListCmd.Flags().StringVar(&commitListBranch, "branch", "", "branch name (required)")
	commitListCmd.MarkFlagRequired("branch")

	commitGetCmd.Flags().StringVar(&commitGetHash, "hash", "", "commit hash (required)")
	commitGetCmd.MarkFlagRequired("hash")

	fileGetCmd.Flags().StringVar(&fileGetRef, "ref", "", "branch name, tag, or commit hash (required)")
	fileGetCmd.Flags().StringVar(&fileGetPath, "path", "", "file path within the repository (required)")
	fileGetCmd.MarkFlagRequired("ref")
	fileGetCmd.MarkFlagRequired("path")

	commitCmd.AddCommand(commitListCmd, commitGetCmd)
	fileCmd.AddCommand(fileGetCmd)
	rootCmd.AddCommand(commitCmd, fileCmd)
}
```

- [ ] **Step 3: Build to verify compilation**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go build -o bb .
```

Expected: No output.

- [ ] **Step 4: Smoke tests**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
./bb branch --help
./bb commit --help
./bb file --help
```

Expected: Each shows its subcommands (list/create/delete, list/get, get respectively).

- [ ] **Step 5: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add cmd/branch.go cmd/commit.go
git commit -m "feat: add bb branch, bb commit, and bb file commands"
```

---

### Task 10: cmd/user.go, cmd/repo.go, and extend cmd/pr.go

**Files:**
- Create: `cmd/user.go`
- Create: `cmd/repo.go`
- Modify: `cmd/pr.go`

**Commands:** `bb user me`, `bb repo list`, `bb pr activity --pr-id N`, `bb pr statuses --pr-id N`

Important: `bb repo list` only needs the workspace — NOT the repo slug. Access `cfg.Workspace` directly (it's a package-level var in `package cmd` set by `PersistentPreRunE`). Do NOT call `workspaceAndRepo()` which would error if no repo is configured.

`bb user me` needs no workspace or repo — just use `client` directly.

The `truncate()` helper is defined in `cmd/comment.go` and available here since all are `package cmd`.

- [ ] **Step 1: Create cmd/user.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/user.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Bitbucket user information",
}

var userMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the authenticated user's profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		u, err := client.User().Me(context.Background())
		if err != nil {
			return err
		}
		return printOutput(u, func() {
			fmt.Printf("%s (@%s)\nAccount ID: %s\n", u.DisplayName, u.Nickname, u.AccountID)
			if u.Links.HTML.Href != "" {
				fmt.Printf("Profile:    %s\n", u.Links.HTML.Href)
			}
		})
	},
}

func init() {
	userCmd.AddCommand(userMeCmd)
	rootCmd.AddCommand(userCmd)
}
```

- [ ] **Step 2: Create cmd/repo.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/repo.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "List and browse repositories",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories in the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		// repo list only needs workspace, not repo slug
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		repos, err := client.Repos(ws).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(repos, func() {
			if len(repos) == 0 {
				fmt.Println("No repositories found.")
				return
			}
			for _, r := range repos {
				privacy := "public"
				if r.IsPrivate {
					privacy = "private"
				}
				fmt.Printf("%-30s  %-40s  (%s)\n", r.Slug, truncate(r.Name, 40), privacy)
			}
		})
	},
}

func init() {
	repoCmd.AddCommand(repoListCmd)
	rootCmd.AddCommand(repoCmd)
}
```

- [ ] **Step 3: Add prActivityCmd and prStatusesCmd to cmd/pr.go**

In `/Users/jay/code/cli-bitbucket-cloud/cmd/pr.go`, add the following two package-level flag vars after the existing `prDeclineID` var:

```go
var prActivityID int
var prStatusesID int
```

Then add the two command vars after `prDeclineCmd`:

```go
var prActivityCmd = &cobra.Command{
	Use:   "activity",
	Short: "Show the activity timeline for a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		activities, err := client.PRs(ws, r).Activity(context.Background(), prActivityID)
		if err != nil {
			return err
		}
		return printOutput(activities, func() {
			if len(activities) == 0 {
				fmt.Println("No activity found.")
				return
			}
			for _, a := range activities {
				switch {
				case a.Approval != nil:
					date := a.Approval.Date
					if len(date) >= 10 {
						date = date[:10]
					}
					fmt.Printf("[approval]  %s approved  (%s)\n",
						a.Approval.User.DisplayName, date)
				case a.Comment != nil:
					fmt.Printf("[comment]   %s: %s\n",
						a.Comment.User.DisplayName, truncate(a.Comment.Content.Raw, 80))
				case a.Update != nil:
					date := a.Update.Date
					if len(date) >= 10 {
						date = date[:10]
					}
					fmt.Printf("[update]    %s → %s  (%s)\n",
						a.Update.Author.DisplayName, a.Update.State, date)
				}
			}
		})
	},
}

var prStatusesCmd = &cobra.Command{
	Use:   "statuses",
	Short: "Show build statuses for a pull request",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		statuses, err := client.PRs(ws, r).Statuses(context.Background(), prStatusesID)
		if err != nil {
			return err
		}
		return printOutput(statuses, func() {
			if len(statuses) == 0 {
				fmt.Println("No statuses found.")
				return
			}
			for _, s := range statuses {
				fmt.Printf("%-12s  %-30s  %s\n",
					s.State, truncate(s.Name, 30), s.Description)
			}
		})
	},
}
```

Then extend the `init()` function in `cmd/pr.go` by adding these lines before the closing brace:

```go
	prActivityCmd.Flags().IntVar(&prActivityID, "pr-id", 0, "pull request ID")
	prActivityCmd.MarkFlagRequired("pr-id")

	prStatusesCmd.Flags().IntVar(&prStatusesID, "pr-id", 0, "pull request ID")
	prStatusesCmd.MarkFlagRequired("pr-id")

	prCmd.AddCommand(prActivityCmd, prStatusesCmd)
```

- [ ] **Step 4: Build to verify compilation**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go build -o bb .
```

Expected: No output (binary built).

- [ ] **Step 5: Smoke tests**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
./bb user --help
./bb repo --help
./bb pr activity --help
./bb pr statuses --help
```

For `./bb pr activity --help`, expected output includes:
```
Show the activity timeline for a pull request

Usage:
  bb pr activity [flags]

Flags:
  -h, --help       help for activity
      --pr-id int   pull request ID
```

- [ ] **Step 6: Run all tests**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./...
```

Expected: All tests PASS (17 in `pkg/bitbucket`, build OK for `cmd/`).

- [ ] **Step 7: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add cmd/user.go cmd/repo.go cmd/pr.go
git commit -m "feat: add bb user me, bb repo list, bb pr activity, bb pr statuses"
```

---

## Self-Review

**Spec coverage:** All 8 priority features are covered:
- Pipeline list/get/trigger/stop/steps/log → Tasks 2, 8 ✅
- Branch list/create/delete → Tasks 3, 9 ✅
- Commit list/get + file get → Tasks 4, 9 ✅
- User me → Tasks 5, 10 ✅
- Repo list → Tasks 6, 10 ✅
- PR activity → Tasks 7, 10 ✅
- PR statuses → Tasks 7, 10 ✅

**Placeholder scan:** No TBDs. All steps contain complete, runnable code.

**Type consistency:**
- `PipelineState.Result *PipelineResult` — used consistently in types, tests, and command text output ✅
- `BranchTarget.Hash` — used in `CreateBranchInput.Target` and branch text output ✅
- `CommitAuthor.Raw` — used in `commitListCmd` text output ✅
- `Activity.Comment *Comment`, `Activity.Approval *Approval`, `Activity.Update *PRUpdate` — all pointer fields, nil-checked in switch ✅
- `truncate()` — defined in `cmd/comment.go`, used in `cmd/commit.go`, `cmd/repo.go`, `cmd/pr.go` — all `package cmd` ✅
- `prPath(prID)` in `PRResource` — `Activity` and `Statuses` both append to it correctly ✅
- `cfg.Workspace` accessed directly in `cmd/repo.go` — `cfg` is package-level in `package cmd` ✅
