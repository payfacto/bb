# BB CLI Workspace Members Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `bb member list` to display all members of the configured workspace.

**Architecture:** `MemberResource` is workspace-scoped (no repo slug) — same shape as `RepoResource`. The `WorkspaceMember` type wraps the existing `User` type (already defined). The cmd layer follows the `cmd/repo.go` pattern: reads `cfg.Workspace` directly rather than calling `workspaceAndRepo()`, since no repo slug is needed.

**Tech Stack:** Go 1.25, `github.com/spf13/cobra v1.10.2`, `net/http/httptest` (stdlib)

---

## File Structure

**New files:**

| File | Responsibility |
|------|---------------|
| `pkg/bitbucket/member.go` | MemberResource: List |
| `pkg/bitbucket/member_test.go` | Tests for MemberResource |
| `cmd/member.go` | `bb member list` |

**Modified files:**

| File | Change |
|------|--------|
| `pkg/bitbucket/types.go` | Append `WorkspaceMember` type |
| `pkg/bitbucket/client.go` | Add `Members(workspace string) *MemberResource` accessor |

---

### Task 1: Add WorkspaceMember type to pkg/bitbucket/types.go

**Files:**
- Modify: `pkg/bitbucket/types.go`

The `User` type already exists in types.go. `WorkspaceMember` wraps it.

- [ ] **Step 1: Append WorkspaceMember type to types.go**

Open `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/types.go` and append at the end of the file:

```go
// WorkspaceMember type

type WorkspaceMember struct {
	User  User  `json:"user"`
	Links Links `json:"links"`
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
git commit -m "feat: add WorkspaceMember type"
```

---

### Task 2: MemberResource

**Files:**
- Create: `pkg/bitbucket/member_test.go`
- Create: `pkg/bitbucket/member.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoint:**
- `GET /workspaces/{workspace}/members?pagelen=50`

This resource is workspace-scoped only (no repo slug) — mirrors the `RepoResource` pattern in `pkg/bitbucket/repo.go`.

- [ ] **Step 1: Create member_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/member_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestMembers_List(t *testing.T) {
	members := []bitbucket.WorkspaceMember{
		{User: bitbucket.User{AccountID: "acc-1", DisplayName: "Alice Smith", Nickname: "alice"}},
		{User: bitbucket.User{AccountID: "acc-2", DisplayName: "Bob Jones", Nickname: "bob"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/workspaces/testws/members" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": members})
	}))
	got, err := client.Members("testws").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].User.DisplayName != "Alice Smith" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[0].User.Nickname != "alice" {
		t.Errorf("expected Nickname=alice, got %s", got[0].User.Nickname)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestMembers -v 2>&1 | head -5
```

Expected: Compilation error — `client.Members undefined`.

- [ ] **Step 3: Create member.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/member.go`:

```go
package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// MemberResource provides operations on workspace members.
type MemberResource struct {
	client    *Client
	workspace string
}

func (r *MemberResource) basePath() string {
	return fmt.Sprintf("/workspaces/%s/members", r.workspace)
}

// List returns all members of the workspace.
func (r *MemberResource) List(ctx context.Context) ([]WorkspaceMember, error) {
	q := url.Values{"pagelen": {"50"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[WorkspaceMember]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
```

- [ ] **Step 4: Add Members() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after the existing `Repos()` method:

```go
// Members returns a resource for workspace member operations.
func (c *Client) Members(workspace string) *MemberResource {
	return &MemberResource{client: c, workspace: workspace}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestMembers -count=1 -v
```

Expected: `TestMembers_List` PASS.

- [ ] **Step 6: Run full test suite**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -count=1
```

Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/member.go pkg/bitbucket/member_test.go pkg/bitbucket/client.go
git commit -m "feat: add MemberResource (list)"
```

---

### Task 3: cmd/member.go

**Files:**
- Create: `cmd/member.go`

**Command:** `bb member list`

This command needs workspace only (not repo). Follow the `cmd/repo.go` pattern: read `cfg.Workspace` directly. Do NOT use `workspaceAndRepo()` and do NOT override `PersistentPreRunE` — `cfg` is already populated by the standard `PersistentPreRunE` in `cmd/root.go`, workspace is available via `cfg.Workspace`.

Text output: one member per line showing `DisplayName` (left-padded to 30 chars), `Nickname` (left-padded to 20 chars), `AccountID`.

- [ ] **Step 1: Create cmd/member.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/member.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var memberCmd = &cobra.Command{
	Use:   "member",
	Short: "List workspace members",
}

var memberListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all members of the workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws := cfg.Workspace
		if ws == "" {
			return fmt.Errorf("no workspace configured — run 'bb setup' or pass --workspace")
		}
		members, err := client.Members(ws).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(members, func() {
			if len(members) == 0 {
				fmt.Println("No members found.")
				return
			}
			for _, m := range members {
				fmt.Printf("%-30s  %-20s  %s\n",
					truncate(m.User.DisplayName, 30), m.User.Nickname, m.User.AccountID)
			}
		})
	},
}

func init() {
	memberCmd.AddCommand(memberListCmd)
	rootCmd.AddCommand(memberCmd)
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
./bb member --help
./bb member list --help
./bb --help | grep member
```

Expected for `./bb member --help`:
```
List workspace members

Usage:
  bb member [command]

Available Commands:
  list        List all members of the workspace
```

Expected for `./bb --help | grep member`: `member      List workspace members`

- [ ] **Step 4: Clean up and commit**

```bash
rm bb
git add cmd/member.go
git commit -m "feat: add bb member commands (list)"
```

---

## Self-Review

**Spec coverage:**
- `bb member list` → Task 3 `memberListCmd` ✅
- `MemberResource.List` → Task 2 ✅
- `WorkspaceMember` type → Task 1 ✅
- `Members()` accessor on `*Client` → Task 2 Step 4 ✅

**Placeholder scan:** No TBDs. All code blocks complete and runnable.

**Type consistency:**
- `client.Members("testws")` returns `*MemberResource` — used in test and cmd ✅
- `WorkspaceMember.User` is `User` (existing type with AccountID, DisplayName, Nickname) — accessed as `m.User.DisplayName`, `m.User.Nickname`, `m.User.AccountID` in cmd ✅
- `truncate(m.User.DisplayName, 30)` — `truncate` is defined in `cmd/comment.go`, available to all `cmd` package files ✅
