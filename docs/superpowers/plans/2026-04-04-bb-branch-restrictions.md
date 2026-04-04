# BB CLI Branch Restrictions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `bb restriction list/create/delete` to manage branch restrictions on a Bitbucket repository.

**Architecture:** `BranchRestrictionResource` is workspace+repo scoped — same shape as `DeployKeyResource`. All operations use JSON. The cmd layer follows the `cmd/deploy_key.go` pattern using `workspaceAndRepo()`. Delete uses an integer restriction ID. Create always sends `branch_match_kind: "glob"` (no flag needed — glob is the only practical CLI use case). The optional `--value` flag uses `-1` as sentinel for "not provided"; any value ≥ 0 is sent in the POST body.

**Tech Stack:** Go 1.25, `github.com/spf13/cobra v1.10.2`, `net/http/httptest` (stdlib)

---

## File Structure

**New files:**

| File | Responsibility |
|------|---------------|
| `pkg/bitbucket/branch_restriction.go` | BranchRestrictionResource: List, Create, Delete |
| `pkg/bitbucket/branch_restriction_test.go` | Tests for BranchRestrictionResource |
| `cmd/restriction.go` | `bb restriction list/create/delete` |

**Modified files:**

| File | Change |
|------|--------|
| `pkg/bitbucket/types.go` | Append `BranchRestriction` and `CreateBranchRestrictionInput` types |
| `pkg/bitbucket/client.go` | Add `Restrictions(workspace, repo string) *BranchRestrictionResource` accessor |

---

### Task 1: Add BranchRestriction types to pkg/bitbucket/types.go

**Files:**
- Modify: `pkg/bitbucket/types.go`

- [ ] **Step 1: Append BranchRestriction types to types.go**

Open `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/types.go` and append at the end of the file:

```go
// BranchRestriction types

type BranchRestriction struct {
	ID              int    `json:"id"`
	Kind            string `json:"kind"`
	BranchMatchKind string `json:"branch_match_kind"`
	Pattern         string `json:"pattern"`
	Value           *int   `json:"value,omitempty"`
	Links           Links  `json:"links"`
}

type CreateBranchRestrictionInput struct {
	Kind            string `json:"kind"`
	BranchMatchKind string `json:"branch_match_kind"`
	Pattern         string `json:"pattern"`
	Value           *int   `json:"value,omitempty"`
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
git commit -m "feat: add BranchRestriction types"
```

---

### Task 2: BranchRestrictionResource + accessor + tests

**Files:**
- Create: `pkg/bitbucket/branch_restriction_test.go`
- Create: `pkg/bitbucket/branch_restriction.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoints:**
- `GET /repositories/{workspace}/{repo}/branch-restrictions?pagelen=50` — list restrictions
- `POST /repositories/{workspace}/{repo}/branch-restrictions` — create restriction (JSON body)
- `DELETE /repositories/{workspace}/{repo}/branch-restrictions/{id}` — delete by integer ID

- [ ] **Step 1: Create branch_restriction_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/branch_restriction_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestBranchRestrictions_List(t *testing.T) {
	val1 := 2
	restrictions := []bitbucket.BranchRestriction{
		{ID: 1, Kind: "require_approvals_to_merge", BranchMatchKind: "glob", Pattern: "main", Value: &val1},
		{ID: 2, Kind: "force", BranchMatchKind: "glob", Pattern: "main"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/branch-restrictions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": restrictions})
	}))
	got, err := client.Restrictions("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 restrictions, got %d", len(got))
	}
	if got[0].ID != 1 || got[0].Kind != "require_approvals_to_merge" || got[0].Pattern != "main" {
		t.Errorf("unexpected first restriction: %+v", got[0])
	}
	if got[0].Value == nil || *got[0].Value != 2 {
		t.Errorf("expected Value=2, got %v", got[0].Value)
	}
	if got[1].ID != 2 || got[1].Kind != "force" {
		t.Errorf("unexpected second restriction: %+v", got[1])
	}
	if got[1].Value != nil {
		t.Errorf("expected nil Value for restriction 2, got %v", got[1].Value)
	}
}

func TestBranchRestrictions_Create(t *testing.T) {
	val := 3
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/branch-restrictions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["kind"] != "require_approvals_to_merge" {
			t.Errorf("expected kind=require_approvals_to_merge, got %v", body["kind"])
		}
		if body["branch_match_kind"] != "glob" {
			t.Errorf("expected branch_match_kind=glob, got %v", body["branch_match_kind"])
		}
		if body["pattern"] != "main" {
			t.Errorf("expected pattern=main, got %v", body["pattern"])
		}
		if body["value"] != float64(3) {
			t.Errorf("expected value=3, got %v", body["value"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bitbucket.BranchRestriction{
			ID:              10,
			Kind:            "require_approvals_to_merge",
			BranchMatchKind: "glob",
			Pattern:         "main",
			Value:           &val,
		})
	}))
	input := bitbucket.CreateBranchRestrictionInput{
		Kind:            "require_approvals_to_merge",
		BranchMatchKind: "glob",
		Pattern:         "main",
		Value:           &val,
	}
	got, err := client.Restrictions("testws", "testrepo").Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 10 || got.Kind != "require_approvals_to_merge" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got.Value == nil || *got.Value != 3 {
		t.Errorf("expected Value=3, got %v", got.Value)
	}
}

func TestBranchRestrictions_CreateNoValue(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, hasValue := body["value"]; hasValue {
			t.Errorf("expected no 'value' field in body, got %v", body["value"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bitbucket.BranchRestriction{
			ID:              11,
			Kind:            "force",
			BranchMatchKind: "glob",
			Pattern:         "main",
		})
	}))
	input := bitbucket.CreateBranchRestrictionInput{
		Kind:            "force",
		BranchMatchKind: "glob",
		Pattern:         "main",
	}
	got, err := client.Restrictions("testws", "testrepo").Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 11 || got.Kind != "force" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestBranchRestrictions_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/branch-restrictions/5" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Restrictions("testws", "testrepo").Delete(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestBranchRestrictions -v 2>&1 | head -10
```

Expected: Compilation error — `client.Restrictions undefined`.

- [ ] **Step 3: Create branch_restriction.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/branch_restriction.go`:

```go
package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// BranchRestrictionResource provides operations on repository branch restrictions.
type BranchRestrictionResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *BranchRestrictionResource) basePath() string {
	return fmt.Sprintf("%s/branch-restrictions", repoPath(r.workspace, r.repo))
}

// List returns all branch restrictions for the repository.
func (r *BranchRestrictionResource) List(ctx context.Context) ([]BranchRestriction, error) {
	q := url.Values{"pagelen": {"50"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[BranchRestriction]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Create adds a new branch restriction to the repository.
func (r *BranchRestrictionResource) Create(ctx context.Context, input CreateBranchRestrictionInput) (BranchRestriction, error) {
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return BranchRestriction{}, err
	}
	return decode[BranchRestriction](data)
}

// Delete removes a branch restriction by its integer ID.
func (r *BranchRestrictionResource) Delete(ctx context.Context, id int) error {
	path := fmt.Sprintf("%s/%d", r.basePath(), id)
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
```

- [ ] **Step 4: Add Restrictions() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after the existing `Issues()` method (before `// repoPath`):

```go
// Restrictions returns a resource for branch restriction operations on the given repo.
func (c *Client) Restrictions(workspace, repo string) *BranchRestrictionResource {
	return &BranchRestrictionResource{client: c, workspace: workspace, repo: repo}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestBranchRestrictions -count=1 -v
```

Expected: `TestBranchRestrictions_List`, `TestBranchRestrictions_Create`, `TestBranchRestrictions_CreateNoValue`, `TestBranchRestrictions_Delete` all PASS.

- [ ] **Step 6: Run full test suite**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -count=1
```

Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/branch_restriction.go pkg/bitbucket/branch_restriction_test.go pkg/bitbucket/client.go
git commit -m "feat: add BranchRestrictionResource (list, create, delete)"
```

---

### Task 3: cmd/restriction.go

**Files:**
- Create: `cmd/restriction.go`

**Commands:** `bb restriction list`, `bb restriction create --kind <kind> --pattern <pattern> [--value <n>]`, `bb restriction delete --id <id>`

These commands need workspace and repo. Follow `cmd/deploy_key.go` pattern: use `workspaceAndRepo()`. Delete takes an integer `--id` flag. Create always sends `branch_match_kind: "glob"`. The `--value` flag uses `-1` as sentinel for "not set"; the RunE sets `input.Value = &v` only when `v >= 0`.

Text output for list: `"%-6d  %-40s  %-6s  %s\n"` — ID (6 chars), kind (40 chars), value (6 chars, blank if nil), pattern.
Text output for create: `"Branch restriction %d (%s on '%s') created.\n"` — ID, kind, pattern.
Text output for delete: `"Branch restriction %d deleted.\n"` — ID.

`truncate()` is defined in `cmd/comment.go` and is available to all files in the `cmd` package.

- [ ] **Step 1: Create cmd/restriction.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/restriction.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

var restrictionCmd = &cobra.Command{
	Use:   "restriction",
	Short: "Manage repository branch restrictions",
}

var restrictionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List branch restrictions for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		restrictions, err := client.Restrictions(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(restrictions, func() {
			if len(restrictions) == 0 {
				fmt.Println("No branch restrictions found.")
				return
			}
			for _, r := range restrictions {
				valueStr := ""
				if r.Value != nil {
					valueStr = fmt.Sprintf("%d", *r.Value)
				}
				fmt.Printf("%-6d  %-40s  %-6s  %s\n", r.ID, truncate(r.Kind, 40), valueStr, r.Pattern)
			}
		})
	},
}

var (
	restrictionCreateKind    string
	restrictionCreatePattern string
	restrictionCreateValue   int
)

var restrictionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a branch restriction",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		input := bitbucket.CreateBranchRestrictionInput{
			Kind:            restrictionCreateKind,
			BranchMatchKind: "glob",
			Pattern:         restrictionCreatePattern,
		}
		if restrictionCreateValue >= 0 {
			v := restrictionCreateValue
			input.Value = &v
		}
		r, err := client.Restrictions(ws, repo).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(r, func() {
			fmt.Printf("Branch restriction %d (%s on '%s') created.\n", r.ID, r.Kind, r.Pattern)
		})
	},
}

var restrictionDeleteID int

var restrictionDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a branch restriction by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Restrictions(ws, repo).Delete(context.Background(), restrictionDeleteID); err != nil {
			return err
		}
		return printOutput(map[string]any{"result": "deleted", "id": restrictionDeleteID}, func() {
			fmt.Printf("Branch restriction %d deleted.\n", restrictionDeleteID)
		})
	},
}

func init() {
	restrictionCreateCmd.Flags().StringVar(&restrictionCreateKind, "kind", "", "restriction kind, e.g. push, force, delete, require_approvals_to_merge (required)")
	restrictionCreateCmd.Flags().StringVar(&restrictionCreatePattern, "pattern", "", "branch glob pattern, e.g. main or feature/* (required)")
	restrictionCreateCmd.Flags().IntVar(&restrictionCreateValue, "value", -1, "integer value for restrictions that require one (e.g. number of approvals)")
	restrictionCreateCmd.MarkFlagRequired("kind")
	restrictionCreateCmd.MarkFlagRequired("pattern")

	restrictionDeleteCmd.Flags().IntVar(&restrictionDeleteID, "id", 0, "branch restriction ID to delete (required)")
	restrictionDeleteCmd.MarkFlagRequired("id")

	restrictionCmd.AddCommand(restrictionListCmd, restrictionCreateCmd, restrictionDeleteCmd)
	rootCmd.AddCommand(restrictionCmd)
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
./bb restriction --help
./bb restriction create --help
./bb restriction delete --help
./bb --help | grep restriction
```

Expected for `./bb restriction --help`:
```
Manage repository branch restrictions

Usage:
  bb restriction [command]

Available Commands:
  create      Create a branch restriction
  delete      Delete a branch restriction by ID
  list        List branch restrictions for the repository
```

Expected for `./bb restriction create --help` (confirms flags):
```
  --kind string      restriction kind, e.g. push, force, delete, require_approvals_to_merge (required)
  --pattern string   branch glob pattern, e.g. main or feature/* (required)
  --value int        integer value for restrictions that require one (e.g. number of approvals) (default -1)
```

Expected for `./bb --help | grep restriction`: `  restriction  Manage repository branch restrictions`

- [ ] **Step 4: Clean up and commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
rm bb
git add cmd/restriction.go
git commit -m "feat: add bb restriction commands (list, create, delete)"
```

---

## Self-Review

**Spec coverage:**
- `bb restriction list` → Task 3 `restrictionListCmd` ✅
- `bb restriction create` → Task 3 `restrictionCreateCmd` ✅
- `bb restriction delete` → Task 3 `restrictionDeleteCmd` ✅
- `BranchRestrictionResource.List` → Task 2 ✅
- `BranchRestrictionResource.Create` → Task 2 ✅
- `BranchRestrictionResource.Delete` → Task 2 ✅
- `BranchRestriction` + `CreateBranchRestrictionInput` types → Task 1 ✅
- `Restrictions()` accessor on `*Client` → Task 2 Step 4 ✅

**Placeholder scan:** No TBDs. All code blocks complete and runnable.

**Type consistency:**
- `client.Restrictions("testws", "testrepo")` returns `*BranchRestrictionResource` — used in tests and cmd ✅
- `BranchRestriction.ID` (int), `.Kind` (string), `.Pattern` (string), `.Value` (*int) — accessed as `r.ID`, `r.Kind`, `r.Pattern`, `r.Value` in cmd ✅
- `BranchRestrictionResource.Delete(ctx, id int)` — cmd passes `restrictionDeleteID` which is `int` (IntVar) ✅
- `CreateBranchRestrictionInput.Value` (*int, omitempty) — set only when `restrictionCreateValue >= 0` in cmd ✅
- `truncate(r.Kind, 40)` — `truncate` defined in `cmd/comment.go` ✅
- Sentinel `-1` for `--value` means "omit from body"; `>= 0` means include ✅
- `TestBranchRestrictions_CreateNoValue` verifies `value` is absent from JSON body when `input.Value` is nil ✅
