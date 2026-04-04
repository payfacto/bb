# BB CLI Deploy Keys Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `bb deploy-key list/add/delete` to manage SSH deploy keys on a Bitbucket repository.

**Architecture:** `DeployKeyResource` is workspace+repo scoped — same shape as `TagResource`. All operations use JSON (no multipart). The cmd layer follows the `cmd/tag.go` pattern using `workspaceAndRepo()`. Delete uses an integer key ID rather than a name string.

**Tech Stack:** Go 1.25, `github.com/spf13/cobra v1.10.2`, `net/http/httptest` (stdlib)

---

## File Structure

**New files:**

| File | Responsibility |
|------|---------------|
| `pkg/bitbucket/deploy_key.go` | DeployKeyResource: List, Add, Delete |
| `pkg/bitbucket/deploy_key_test.go` | Tests for DeployKeyResource |
| `cmd/deploy_key.go` | `bb deploy-key list/add/delete` |

**Modified files:**

| File | Change |
|------|--------|
| `pkg/bitbucket/types.go` | Append `DeployKey` and `AddDeployKeyInput` types |
| `pkg/bitbucket/client.go` | Add `DeployKeys(workspace, repo string) *DeployKeyResource` accessor |

---

### Task 1: Add DeployKey types to pkg/bitbucket/types.go

**Files:**
- Modify: `pkg/bitbucket/types.go`

- [ ] **Step 1: Append DeployKey types to types.go**

Open `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/types.go` and append at the end of the file:

```go
// DeployKey types

type DeployKey struct {
	ID        int    `json:"id"`
	Label     string `json:"label"`
	Key       string `json:"key"`
	CreatedOn string `json:"created_on"`
	Links     Links  `json:"links"`
}

type AddDeployKeyInput struct {
	Label string `json:"label"`
	Key   string `json:"key"`
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
git commit -m "feat: add DeployKey types"
```

---

### Task 2: DeployKeyResource + accessor + tests

**Files:**
- Create: `pkg/bitbucket/deploy_key_test.go`
- Create: `pkg/bitbucket/deploy_key.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoints:**
- `GET /repositories/{workspace}/{repo}/deploy-keys?pagelen=50` — list keys
- `POST /repositories/{workspace}/{repo}/deploy-keys` — add key (JSON body)
- `DELETE /repositories/{workspace}/{repo}/deploy-keys/{key_id}` — delete by integer ID

- [ ] **Step 1: Create deploy_key_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/deploy_key_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestDeployKeys_List(t *testing.T) {
	keys := []bitbucket.DeployKey{
		{ID: 1, Label: "CI server", Key: "ssh-rsa AAAAB3NzaC1yc2E ci@example.com"},
		{ID: 2, Label: "Deploy bot", Key: "ssh-rsa AAAAB3NzaC1yc2E bot@example.com"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/deploy-keys" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": keys})
	}))
	got, err := client.DeployKeys("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[0].Label != "CI server" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[1].ID != 2 {
		t.Errorf("expected ID=2, got %d", got[1].ID)
	}
}

func TestDeployKeys_Add(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/deploy-keys" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["label"] != "My Key" {
			t.Errorf("expected label=My Key, got %s", body["label"])
		}
		if body["key"] != "ssh-rsa AAAAB3NzaC1yc2E test@example.com" {
			t.Errorf("unexpected key: %s", body["key"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bitbucket.DeployKey{
			ID:    42,
			Label: "My Key",
			Key:   "ssh-rsa AAAAB3NzaC1yc2E test@example.com",
		})
	}))
	got, err := client.DeployKeys("testws", "testrepo").Add(context.Background(), "My Key", "ssh-rsa AAAAB3NzaC1yc2E test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 42 || got.Label != "My Key" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestDeployKeys_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/deploy-keys/7" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.DeployKeys("testws", "testrepo").Delete(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestDeployKeys -v 2>&1 | head -10
```

Expected: Compilation error — `client.DeployKeys undefined`.

- [ ] **Step 3: Create deploy_key.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/deploy_key.go`:

```go
package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// DeployKeyResource provides operations on repository deploy keys.
type DeployKeyResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *DeployKeyResource) basePath() string {
	return fmt.Sprintf("%s/deploy-keys", repoPath(r.workspace, r.repo))
}

// List returns all deploy keys for the repository.
func (r *DeployKeyResource) List(ctx context.Context) ([]DeployKey, error) {
	q := url.Values{"pagelen": {"50"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[DeployKey]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Add creates a new deploy key with the given label and SSH public key.
func (r *DeployKeyResource) Add(ctx context.Context, label, key string) (DeployKey, error) {
	input := AddDeployKeyInput{Label: label, Key: key}
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return DeployKey{}, err
	}
	return decode[DeployKey](data)
}

// Delete removes a deploy key by its integer ID.
func (r *DeployKeyResource) Delete(ctx context.Context, id int) error {
	path := fmt.Sprintf("%s/%d", r.basePath(), id)
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
```

- [ ] **Step 4: Add DeployKeys() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after the existing `Downloads()` method:

```go
// DeployKeys returns a resource for deploy key operations on the given repo.
func (c *Client) DeployKeys(workspace, repo string) *DeployKeyResource {
	return &DeployKeyResource{client: c, workspace: workspace, repo: repo}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestDeployKeys -count=1 -v
```

Expected: `TestDeployKeys_List`, `TestDeployKeys_Add`, `TestDeployKeys_Delete` all PASS.

- [ ] **Step 6: Run full test suite**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -count=1
```

Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/deploy_key.go pkg/bitbucket/deploy_key_test.go pkg/bitbucket/client.go
git commit -m "feat: add DeployKeyResource (list, add, delete)"
```

---

### Task 3: cmd/deploy_key.go

**Files:**
- Create: `cmd/deploy_key.go`

**Commands:** `bb deploy-key list`, `bb deploy-key add --label <label> --key <pubkey>`, `bb deploy-key delete --id <id>`

These commands need workspace and repo. Follow `cmd/tag.go` pattern: use `workspaceAndRepo()`. Delete takes an integer `--id` flag (use `int` var + `strconv` is not needed — cobra handles int flags directly with `IntVar`).

Text output for list: `"%-6d  %-30s  %s\n"` — ID (6 chars), label (truncated to 30), key (truncated to 50).
Text output for add: `"Deploy key '%s' added (ID: %d).\n"` — label, ID.
Text output for delete: `"Deploy key %d deleted.\n"` — ID.

`truncate()` is defined in `cmd/comment.go` and is available to all files in the `cmd` package.

- [ ] **Step 1: Create cmd/deploy_key.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/deploy_key.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var deployKeyCmd = &cobra.Command{
	Use:   "deploy-key",
	Short: "Manage repository deploy keys",
}

var deployKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List deploy keys for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		keys, err := client.DeployKeys(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(keys, func() {
			if len(keys) == 0 {
				fmt.Println("No deploy keys found.")
				return
			}
			for _, k := range keys {
				fmt.Printf("%-6d  %-30s  %s\n", k.ID, truncate(k.Label, 30), truncate(k.Key, 50))
			}
		})
	},
}

var (
	deployKeyAddLabel string
	deployKeyAddKey   string
)

var deployKeyAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a deploy key to the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		k, err := client.DeployKeys(ws, repo).Add(context.Background(), deployKeyAddLabel, deployKeyAddKey)
		if err != nil {
			return err
		}
		return printOutput(k, func() {
			fmt.Printf("Deploy key '%s' added (ID: %d).\n", k.Label, k.ID)
		})
	},
}

var deployKeyDeleteID int

var deployKeyDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a deploy key by ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.DeployKeys(ws, repo).Delete(context.Background(), deployKeyDeleteID); err != nil {
			return err
		}
		return printOutput(map[string]any{"result": "deleted", "id": deployKeyDeleteID}, func() {
			fmt.Printf("Deploy key %d deleted.\n", deployKeyDeleteID)
		})
	},
}

func init() {
	deployKeyAddCmd.Flags().StringVar(&deployKeyAddLabel, "label", "", "label for the deploy key (required)")
	deployKeyAddCmd.Flags().StringVar(&deployKeyAddKey, "key", "", "SSH public key string (required)")
	deployKeyAddCmd.MarkFlagRequired("label")
	deployKeyAddCmd.MarkFlagRequired("key")

	deployKeyDeleteCmd.Flags().IntVar(&deployKeyDeleteID, "id", 0, "deploy key ID to delete (required)")
	deployKeyDeleteCmd.MarkFlagRequired("id")

	deployKeyCmd.AddCommand(deployKeyListCmd, deployKeyAddCmd, deployKeyDeleteCmd)
	rootCmd.AddCommand(deployKeyCmd)
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
./bb deploy-key --help
./bb deploy-key add --help
./bb deploy-key delete --help
./bb --help | grep deploy
```

Expected for `./bb deploy-key --help`:
```
Manage repository deploy keys

Usage:
  bb deploy-key [command]

Available Commands:
  add         Add a deploy key to the repository
  delete      Delete a deploy key by ID
  list        List deploy keys for the repository
```

Expected for `./bb deploy-key add --help` (confirms flags):
```
  --key string     SSH public key string (required)
  --label string   label for the deploy key (required)
```

Expected for `./bb --help | grep deploy`: `  deploy-key  Manage repository deploy keys`

- [ ] **Step 4: Clean up and commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
rm bb
git add cmd/deploy_key.go
git commit -m "feat: add bb deploy-key commands (list, add, delete)"
```

---

## Self-Review

**Spec coverage:**
- `bb deploy-key list` → Task 3 `deployKeyListCmd` ✅
- `bb deploy-key add` → Task 3 `deployKeyAddCmd` ✅
- `bb deploy-key delete` → Task 3 `deployKeyDeleteCmd` ✅
- `DeployKeyResource.List` → Task 2 ✅
- `DeployKeyResource.Add` → Task 2 ✅
- `DeployKeyResource.Delete` → Task 2 ✅
- `DeployKey` + `AddDeployKeyInput` types → Task 1 ✅
- `DeployKeys()` accessor on `*Client` → Task 2 Step 4 ✅

**Placeholder scan:** No TBDs. All code blocks complete and runnable.

**Type consistency:**
- `client.DeployKeys("testws", "testrepo")` returns `*DeployKeyResource` — used in tests and cmd ✅
- `DeployKey.ID` (int), `DeployKey.Label` (string), `DeployKey.Key` (string) — accessed as `k.ID`, `k.Label`, `k.Key` in cmd ✅
- `DeployKeyResource.Delete(ctx, id int)` — cmd passes `deployKeyDeleteID` which is `int` (IntVar) ✅
- `truncate(k.Label, 30)`, `truncate(k.Key, 50)` — `truncate` defined in `cmd/comment.go` ✅
- `AddDeployKeyInput{Label: label, Key: key}` — matches struct defined in Task 1 ✅
