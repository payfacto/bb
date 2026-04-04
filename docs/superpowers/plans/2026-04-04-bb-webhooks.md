# BB CLI Webhooks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `bb webhook list/create/delete` to manage repository webhooks on Bitbucket.

**Architecture:** `WebhookResource` is workspace+repo scoped — same shape as `DeployKeyResource`. All operations use JSON. Delete uses the webhook UUID string (not an integer), which requires `url.PathEscape` since UUIDs are wrapped in braces (e.g., `{abc123}`). The cmd layer follows the `cmd/deploy_key.go` pattern using `workspaceAndRepo()`. The `--events` flag is repeatable (`StringArrayVar`) to allow multiple event types; `--active` defaults to `true`.

**Tech Stack:** Go 1.25, `github.com/spf13/cobra v1.10.2`, `net/http/httptest` (stdlib)

---

## File Structure

**New files:**

| File | Responsibility |
|------|---------------|
| `pkg/bitbucket/webhook.go` | WebhookResource: List, Create, Delete |
| `pkg/bitbucket/webhook_test.go` | Tests for WebhookResource |
| `cmd/webhook.go` | `bb webhook list/create/delete` |

**Modified files:**

| File | Change |
|------|--------|
| `pkg/bitbucket/types.go` | Append `Webhook` and `CreateWebhookInput` types |
| `pkg/bitbucket/client.go` | Add `Webhooks(workspace, repo string) *WebhookResource` accessor |

---

### Task 1: Add Webhook types to pkg/bitbucket/types.go

**Files:**
- Modify: `pkg/bitbucket/types.go`

- [ ] **Step 1: Append Webhook types to types.go**

Open `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/types.go` and append at the end of the file:

```go
// Webhook types

type Webhook struct {
	UUID        string   `json:"uuid"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Events      []string `json:"events"`
	CreatedAt   string   `json:"created_at"`
	Links       Links    `json:"links"`
}

type CreateWebhookInput struct {
	Description string   `json:"description,omitempty"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Events      []string `json:"events"`
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
git commit -m "feat: add Webhook types"
```

---

### Task 2: WebhookResource + accessor + tests

**Files:**
- Create: `pkg/bitbucket/webhook_test.go`
- Create: `pkg/bitbucket/webhook.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoints:**
- `GET /repositories/{workspace}/{repo}/hooks?pagelen=50` — list webhooks
- `POST /repositories/{workspace}/{repo}/hooks` — create (JSON body)
- `DELETE /repositories/{workspace}/{repo}/hooks/{uuid}` — delete by UUID string

**Important:** Webhook UUIDs from Bitbucket are wrapped in braces, e.g. `{abc123-...}`. When building the delete URL path, use `url.PathEscape(uuid)` so the braces are percent-encoded. In the test handler, `r.URL.Path` contains the **decoded** path (Go's httptest automatically decodes percent-encoding), so test against the decoded UUID string including braces.

- [ ] **Step 1: Create webhook_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/webhook_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestWebhooks_List(t *testing.T) {
	hooks := []bitbucket.Webhook{
		{UUID: "{abc123}", Description: "CI hook", URL: "https://ci.example.com/hook", Active: true, Events: []string{"repo:push"}},
		{UUID: "{def456}", Description: "Notify", URL: "https://notify.example.com/hook", Active: false, Events: []string{"pullrequest:created", "pullrequest:merged"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/hooks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": hooks})
	}))
	got, err := client.Webhooks("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 webhooks, got %d", len(got))
	}
	if got[0].UUID != "{abc123}" || got[0].Description != "CI hook" || !got[0].Active {
		t.Errorf("unexpected first webhook: %+v", got[0])
	}
	if len(got[0].Events) != 1 || got[0].Events[0] != "repo:push" {
		t.Errorf("unexpected events: %v", got[0].Events)
	}
	if got[1].UUID != "{def456}" || got[1].Active {
		t.Errorf("unexpected second webhook: %+v", got[1])
	}
}

func TestWebhooks_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/hooks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["url"] != "https://example.com/hook" {
			t.Errorf("expected url=https://example.com/hook, got %v", body["url"])
		}
		if body["active"] != true {
			t.Errorf("expected active=true, got %v", body["active"])
		}
		events, ok := body["events"].([]any)
		if !ok || len(events) != 2 {
			t.Errorf("expected 2 events, got %v", body["events"])
		}
		if body["description"] != "My hook" {
			t.Errorf("expected description=My hook, got %v", body["description"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bitbucket.Webhook{
			UUID:        "{new-hook-uuid}",
			Description: "My hook",
			URL:         "https://example.com/hook",
			Active:      true,
			Events:      []string{"repo:push", "pullrequest:created"},
		})
	}))
	input := bitbucket.CreateWebhookInput{
		Description: "My hook",
		URL:         "https://example.com/hook",
		Active:      true,
		Events:      []string{"repo:push", "pullrequest:created"},
	}
	got, err := client.Webhooks("testws", "testrepo").Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{new-hook-uuid}" || got.Description != "My hook" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestWebhooks_CreateNoDescription(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, hasDesc := body["description"]; hasDesc {
			t.Errorf("expected no 'description' field in body, got %v", body["description"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bitbucket.Webhook{
			UUID:   "{minimal-hook}",
			URL:    "https://example.com/hook",
			Active: true,
			Events: []string{"repo:push"},
		})
	}))
	input := bitbucket.CreateWebhookInput{
		URL:    "https://example.com/hook",
		Active: true,
		Events: []string{"repo:push"},
	}
	got, err := client.Webhooks("testws", "testrepo").Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{minimal-hook}" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestWebhooks_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		// r.URL.Path contains the decoded path — braces are unescaped by httptest
		if r.URL.Path != "/repositories/testws/testrepo/hooks/{abc123}" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Webhooks("testws", "testrepo").Delete(context.Background(), "{abc123}")
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestWebhooks -v 2>&1 | head -10
```

Expected: Compilation error — `client.Webhooks undefined`.

- [ ] **Step 3: Create webhook.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/webhook.go`:

```go
package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// WebhookResource provides operations on repository webhooks.
type WebhookResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *WebhookResource) basePath() string {
	return fmt.Sprintf("%s/hooks", repoPath(r.workspace, r.repo))
}

// List returns all webhooks for the repository.
func (r *WebhookResource) List(ctx context.Context) ([]Webhook, error) {
	q := url.Values{"pagelen": {"50"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Webhook]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Create adds a new webhook to the repository.
func (r *WebhookResource) Create(ctx context.Context, input CreateWebhookInput) (Webhook, error) {
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Webhook{}, err
	}
	return decode[Webhook](data)
}

// Delete removes a webhook by its UUID string.
func (r *WebhookResource) Delete(ctx context.Context, uuid string) error {
	path := fmt.Sprintf("%s/%s", r.basePath(), url.PathEscape(uuid))
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
```

- [ ] **Step 4: Add Webhooks() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add after the existing `Restrictions()` method (before `// repoPath`):

```go
// Webhooks returns a resource for webhook operations on the given repo.
func (c *Client) Webhooks(workspace, repo string) *WebhookResource {
	return &WebhookResource{client: c, workspace: workspace, repo: repo}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestWebhooks -count=1 -v
```

Expected: `TestWebhooks_List`, `TestWebhooks_Create`, `TestWebhooks_CreateNoDescription`, `TestWebhooks_Delete` all PASS.

- [ ] **Step 6: Run full test suite**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -count=1
```

Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/webhook.go pkg/bitbucket/webhook_test.go pkg/bitbucket/client.go
git commit -m "feat: add WebhookResource (list, create, delete)"
```

---

### Task 3: cmd/webhook.go

**Files:**
- Create: `cmd/webhook.go`

**Commands:** `bb webhook list`, `bb webhook create --url <url> --events <event> [--events <event> ...] [--description <s>] [--active=false]`, `bb webhook delete --uuid <uuid>`

These commands need workspace and repo. Follow `cmd/deploy_key.go` pattern: use `workspaceAndRepo()`. Delete takes a string `--uuid` flag. Create uses repeatable `StringArrayVar` for `--events`. The `--active` flag defaults to `true` (use `BoolVar` with default `true`).

Text output for list: `"%-38s  %-8v  %s\n"` — UUID (38 chars), active (bool printed as true/false), url truncated to 60.
Text output for create: `"Webhook created (UUID: %s).\n"` — uuid.
Text output for delete: `"Webhook %s deleted.\n"` — uuid.

`truncate()` is defined in `cmd/comment.go` and is available to all files in the `cmd` package.

- [ ] **Step 1: Create cmd/webhook.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/webhook.go`:

```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage repository webhooks",
}

var webhookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhooks for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		hooks, err := client.Webhooks(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(hooks, func() {
			if len(hooks) == 0 {
				fmt.Println("No webhooks found.")
				return
			}
			for _, h := range hooks {
				fmt.Printf("%-38s  %-8v  %s\n", h.UUID, h.Active, truncate(h.URL, 60))
			}
		})
	},
}

var (
	webhookCreateURL         string
	webhookCreateEvents      []string
	webhookCreateDescription string
	webhookCreateActive      bool
)

var webhookCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook for the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		input := bitbucket.CreateWebhookInput{
			Description: webhookCreateDescription,
			URL:         webhookCreateURL,
			Active:      webhookCreateActive,
			Events:      webhookCreateEvents,
		}
		h, err := client.Webhooks(ws, repo).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(h, func() {
			fmt.Printf("Webhook created (UUID: %s).\n", h.UUID)
		})
	},
}

var webhookDeleteUUID string

var webhookDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a webhook by UUID",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Webhooks(ws, repo).Delete(context.Background(), webhookDeleteUUID); err != nil {
			return err
		}
		return printOutput(map[string]any{"result": "deleted", "uuid": webhookDeleteUUID}, func() {
			fmt.Printf("Webhook %s deleted.\n", webhookDeleteUUID)
		})
	},
}

func init() {
	webhookCreateCmd.Flags().StringVar(&webhookCreateURL, "url", "", "webhook endpoint URL (required)")
	webhookCreateCmd.Flags().StringArrayVar(&webhookCreateEvents, "events", nil, "event type to subscribe to, e.g. repo:push (repeatable, required)")
	webhookCreateCmd.Flags().StringVar(&webhookCreateDescription, "description", "", "webhook description")
	webhookCreateCmd.Flags().BoolVar(&webhookCreateActive, "active", true, "whether the webhook is active")
	webhookCreateCmd.MarkFlagRequired("url")
	webhookCreateCmd.MarkFlagRequired("events")

	webhookDeleteCmd.Flags().StringVar(&webhookDeleteUUID, "uuid", "", "webhook UUID to delete (required)")
	webhookDeleteCmd.MarkFlagRequired("uuid")

	webhookCmd.AddCommand(webhookListCmd, webhookCreateCmd, webhookDeleteCmd)
	rootCmd.AddCommand(webhookCmd)
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
./bb webhook --help
./bb webhook create --help
./bb webhook delete --help
./bb --help | grep webhook
```

Expected for `./bb webhook --help`:
```
Manage repository webhooks

Usage:
  bb webhook [command]

Available Commands:
  create      Create a webhook for the repository
  delete      Delete a webhook by UUID
  list        List webhooks for the repository
```

Expected for `./bb webhook create --help` (confirms flags):
```
  --active           whether the webhook is active (default true)
  --description string   webhook description
  --events stringArray   event type to subscribe to, e.g. repo:push (repeatable, required)
  --url string           webhook endpoint URL (required)
```

Expected for `./bb --help | grep webhook`: `  webhook     Manage repository webhooks`

- [ ] **Step 4: Clean up and commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
rm bb
git add cmd/webhook.go
git commit -m "feat: add bb webhook commands (list, create, delete)"
```

---

## Self-Review

**Spec coverage:**
- `bb webhook list` → Task 3 `webhookListCmd` ✅
- `bb webhook create` → Task 3 `webhookCreateCmd` ✅
- `bb webhook delete` → Task 3 `webhookDeleteCmd` ✅
- `WebhookResource.List` → Task 2 ✅
- `WebhookResource.Create` → Task 2 ✅
- `WebhookResource.Delete` → Task 2 ✅
- `Webhook` + `CreateWebhookInput` types → Task 1 ✅
- `Webhooks()` accessor on `*Client` → Task 2 Step 4 ✅

**Placeholder scan:** No TBDs. All code blocks complete and runnable.

**Type consistency:**
- `client.Webhooks("testws", "testrepo")` returns `*WebhookResource` — used in tests and cmd ✅
- `Webhook.UUID` (string), `.Description` (string), `.URL` (string), `.Active` (bool), `.Events` ([]string) — accessed correctly in cmd ✅
- `WebhookResource.Delete(ctx, uuid string)` — cmd passes `webhookDeleteUUID` (StringVar) ✅
- `CreateWebhookInput.Description` has `omitempty` — `TestWebhooks_CreateNoDescription` verifies it's absent from JSON when empty ✅
- `url.PathEscape(uuid)` in `Delete()` handles braces in UUID; `r.URL.Path` in test handler sees decoded path ✅
- `StringArrayVar` for `--events` allows `--events repo:push --events pullrequest:created` (multiple flags) ✅
- `BoolVar(&webhookCreateActive, "active", true, ...)` defaults to `true` — `--active=false` disables ✅
