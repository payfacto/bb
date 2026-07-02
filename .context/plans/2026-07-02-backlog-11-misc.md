# Backlog #11 (scoped) — pr open + commit statuses Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Add `bb pr open -p ID` (open a PR in the browser) and `bb commit statuses --hash HASH` (list a commit's build/CI statuses). Scoped subset of audit #11 (test-reports and enable/disable+schedules deferred by user decision).

**Architecture:** Thin-client + thin-cmd, matching existing patterns. `pr open` reuses the existing `PRs.Get` for the canonical `links.html.href` and opens it via the already-vendored `github.com/pkg/browser`; no new client method. `commit statuses` adds a `CommitStatus` type + `CommitResource.Statuses` (paginated via `fetchAllPages`) + a render helper.

**Tech Stack:** Go 1.26, Cobra, stdlib `net/http`/`httptest`, `github.com/pkg/browser` (already a dependency, used in `cmd/tui/sections.go` and `internal/auth/oauth.go`).

## Global Constraints

- Tests live only in `pkg/bitbucket/` via `newTestClient`+`mustEncodeJSON`; `cmd/` wiring is intentionally untested.
- New leaves MUST be in `commandRegistry`; new output Go types MUST be in `typeRegistry` (`CommitStatus`/`[]CommitStatus` are new; `PR` already exists). Regenerate golden: `go test ./cmd/ -update`.
- Docs sync REQUIRED: `README.md`, `llms.txt`, `CLAUDE.md` (command hierarchy: `pr ... / open`; `commit list / get / statuses`).
- No em-dashes; plain ASCII only (prose, comments, commit messages).
- `pr open` triggers a browser launch (a side effect, not an API mutation) - safe to run, but in non-TTY/headless contexts the browser may not open; the command MUST always print the URL to stdout so headless callers still get it, and treat a browser-launch failure as non-fatal (note to stderr, exit 0).
- `commit statuses` is a READ - live-verify during TDD is expected and safe (creds configured; real commit hash `8a0226475d7058a89f52e1757693dfd1a02ed491` on `payfactopay/payment-platform-portal`).
- Never push/tag without explicit user sign-off.

---

### Task 1: `bb pr open -p ID`

**Files:**
- Modify: `cmd/pr.go`
- Modify: `cmd/manifest_registry.go` (pr section)
- Regen: `cmd/testdata/manifest.golden.json`
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Consumes: existing `client.PRs(ws,repo).Get(ctx, id) (PR, error)` and `PR.Links.HTML.Href`; `github.com/pkg/browser`.
- Produces: a `pr open` Cobra leaf. No new client method, no new type.

**Note:** `cmd/` is untested by convention, and `pr open`'s only logic is fetch-then-open. No pkg test is added. The manifest entry + docs are the reviewable deliverable.

- [ ] **Step 1: Add the `pr open` command**

In `cmd/pr.go`, add (near the other PR commands; `-p`/`--pr` is the established PR-id flag used by `pr update` etc.):

```go
var prOpenID int

var prOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "Open a pull request in your web browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		pr, err := client.PRs(ws, repo).Get(context.Background(), prOpenID)
		if err != nil {
			return err
		}
		url := pr.Links.HTML.Href
		if url == "" {
			return newCLIError(ErrCodeNotFound, fmt.Sprintf("pull request %d has no web link", prOpenID), nil)
		}
		if err := browser.OpenURL(url); err != nil {
			fmt.Fprintf(os.Stderr, "could not open browser: %v\n", err)
		}
		return printOutput(map[string]any{"url": url, "id": prOpenID}, func() {
			fmt.Println(url)
		})
	},
}
```

Add imports as needed: `"os"`, `"github.com/pkg/browser"` (and `"fmt"`/`"context"` if not already imported in `cmd/pr.go` — they are). Verify `newCLIError`/`ErrCodeNotFound` are in package `cmd` (they are, in `cmd/errors.go`).

Register in `pr.go`'s `init()`:

```go
prOpenCmd.Flags().IntVarP(&prOpenID, "pr", "p", 0, "pull request ID (required)")
prOpenCmd.MarkFlagRequired("pr")
```

Add `prOpenCmd` to the `prCmd.AddCommand(...)` call (find where PR subcommands are registered).

- [ ] **Step 2: Register in manifest + regen golden**

Add to `cmd/manifest_registry.go` pr section:

```go
"pr open": {Action: actionRead, OutputType: "ResultMap", Example: "bb pr open -p 42"},
```

Run `go test ./cmd/ -update`; confirm `pr open` appears in the golden.

- [ ] **Step 3: Docs**

Add `bb pr open -p ID` to the README Commands block and `llms.txt`; add `open` to the `CLAUDE.md` `pr` hierarchy line (e.g. `... / decline / open` or wherever it reads best).

- [ ] **Step 4: Build + verify (do NOT rely on a browser opening in this env)**

Run: `go build -o bb . && go test ./... && go vet ./... && gofmt -l cmd pkg internal`
Expected: build ok, all tests pass, vet/gofmt clean.
Safe check: `./bb pr open` with no `-p` errors on the required flag; `./bb pr open -p 26 --format json` prints `{"id":26,"url":"https://bitbucket.org/..."}` on stdout (a browser may or may not open in this environment; a browser-launch failure prints a stderr note but the command still exits 0 with the URL on stdout). Record the JSON output.

- [ ] **Step 5: Commit**

```bash
git add cmd/pr.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(pr): add pr open to launch a PR in the browser"
```

---

### Task 2: `bb commit statuses --hash HASH`

**Files:**
- Modify: `pkg/bitbucket/types.go` (add `CommitStatus`)
- Modify: `pkg/bitbucket/commit.go`
- Modify: `pkg/bitbucket/commit_test.go`
- Create/modify render: `cmd/render/commit.go` (add `CommitStatusList*`)
- Modify: `cmd/commit.go`
- Modify: `cmd/manifest_registry.go` (commit section + typeRegistry)
- Regen: `cmd/testdata/manifest.golden.json`
- Docs: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Produces: `type CommitStatus struct { Key, Name, Description, State, URL, RefName string }` with json tags `key,name,description,state,url,refname`.
- Produces: `func (r *CommitResource) Statuses(ctx context.Context, hash string) ([]CommitStatus, error)` — `GET commit/{hash}/statuses`, paginated via `fetchAllPages`.
- Produces: `func CommitStatusListString(statuses []bitbucket.CommitStatus) string` + `CommitStatusList`.

- [ ] **Step 1: Add the type**

In `pkg/bitbucket/types.go`, near the `Commit` types, add:

```go
// CommitStatus is a build/CI status attached to a commit (e.g. a pipeline result).
type CommitStatus struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"` // SUCCESSFUL | FAILED | INPROGRESS | STOPPED
	URL         string `json:"url"`
	RefName     string `json:"refname"`
}
```

- [ ] **Step 2: Write failing client test**

Add to `pkg/bitbucket/commit_test.go` (match existing style; ensure `encoding/json` not needed unless asserting body - this is a GET):

```go
func TestCommits_Statuses(t *testing.T) {
	statuses := []bitbucket.CommitStatus{
		{Key: "PIPELINE", Name: "Build #42", State: "SUCCESSFUL", URL: "https://ci/42", RefName: "main"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/commit/abc123/statuses" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		mustEncodeJSON(t, w, map[string]any{"values": statuses})
	}))
	got, err := client.Commits("testws", "testrepo").Statuses(context.Background(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].State != "SUCCESSFUL" || got[0].Key != "PIPELINE" {
		t.Errorf("unexpected result: %+v", got)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestCommits_Statuses`
Expected: FAIL — `Statuses`/`CommitStatus` undefined.

- [ ] **Step 4: Implement the client method**

Add to `pkg/bitbucket/commit.go`:

```go
// Statuses returns the build/CI statuses attached to a commit.
func (r *CommitResource) Statuses(ctx context.Context, hash string) ([]CommitStatus, error) {
	path := fmt.Sprintf("%s/commit/%s/statuses", repoPath(r.workspace, r.repo), url.PathEscape(hash))
	q := url.Values{"pagelen": {pagelenDefault}}
	return fetchAllPages[CommitStatus](ctx, r.client, path, q)
}
```

(`fmt` and `net/url` are already imported in `commit.go`.)

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./pkg/bitbucket/ -run TestCommits_Statuses`
Expected: PASS.

- [ ] **Step 6: Add the render helper**

In `cmd/render/commit.go` (read it first to match style; it has `CommitList`/`CommitDetail`), add a `CommitStatusListString`/`CommitStatusList` pair. Empty -> "No commit statuses found.\n". Columns: KEY, STATE (use `StateBadge` if present in the render package for coloring), NAME, URL. Follow the table idiom used by the sibling list renderers in `cmd/render/`.

- [ ] **Step 7: Add the cmd leaf**

In `cmd/commit.go`, add `commit statuses --hash` mirroring `commit get`'s flag:

```go
var commitStatusesHash string

var commitStatusesCmd = &cobra.Command{
	Use:   "statuses",
	Short: "List build/CI statuses for a commit",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		statuses, err := client.Commits(ws, repo).Statuses(context.Background(), commitStatusesHash)
		if err != nil {
			return err
		}
		return printOutput(statuses, func() { render.CommitStatusList(statuses) })
	},
}
```

Register in `init()`:

```go
commitStatusesCmd.Flags().StringVarP(&commitStatusesHash, "hash", "x", "", "commit hash (required)")
commitStatusesCmd.MarkFlagRequired("hash")
```

Add `commitStatusesCmd` to `commitCmd.AddCommand(...)`.

- [ ] **Step 8: Manifest + typeRegistry + golden**

Add to `cmd/manifest_registry.go` commit section:

```go
"commit statuses": {Action: actionRead, OutputType: "[]CommitStatus", Ordering: "unspecified", Example: "bb commit statuses --hash '{hash}'"},
```

Add to `typeRegistry`:

```go
"CommitStatus":   bitbucket.CommitStatus{},
"[]CommitStatus": []bitbucket.CommitStatus{},
```

Run `go test ./cmd/ -update`; confirm `commit statuses` in golden.

- [ ] **Step 9: Docs + live-verify + full verify + commit**

Docs: CLAUDE.md `commit list / get` -> `commit list / get / statuses`; README + llms.txt add the line.
Live-verify (safe read): `go build -o bb . && ./bb commit statuses --hash 8a0226475d7058a89f52e1757693dfd1a02ed491 --format json` returns an array (possibly empty `[]` if that commit has no statuses - both are valid; if empty, try a recent commit hash from `./bb commit list -b main`). Record the result and confirm the field names decode (key/state/name).
Full verify: `go test ./... && go vet ./... && gofmt -l cmd pkg internal`.

```bash
git add pkg/bitbucket/types.go pkg/bitbucket/commit.go pkg/bitbucket/commit_test.go cmd/render/commit.go cmd/commit.go cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(commit): add commit statuses to list build/CI statuses"
```

---

## Self-Review

**Spec coverage** (scoped audit #11):
- `bb pr open` -> Task 1. ✓
- `bb commit statuses` (read) -> Task 2. ✓
- test-reports, enable/disable, schedules -> DEFERRED (user decision). ✓

**Placeholder scan:** none.

**Type consistency:** `pr open` reuses `PRs.Get`+`PR.Links.HTML.Href` (no new types); `CommitStatus` + `CommitResource.Statuses` + `render.CommitStatusList` consistent across Task 2. `fetchAllPages` now returns non-nil empty (fixed in #10), so an empty statuses result renders `[]`.

**Ordering:** Task 1 and Task 2 are independent; both touch `cmd/manifest_registry.go` + golden + docs, so run sequentially to avoid golden churn.
