# `bb search` Namespace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a top-level `bb search` namespace with `code` (native Bitbucket code search), `repos` (BBQL name/description match), and `prs` (repo-scoped BBQL title/description match).

**Architecture:** Follow the existing scoped-resource client pattern. A new `SearchResource` (workspace-scoped) exposes `Code` and `Repos`; PR text search reuses `PRResource.List` via two new `PRListOptions` fields. A new `fetchPagesLimit` helper adds `--limit` support to the shared pagination logic. The cmd layer is thin Cobra wiring; a new `cmd/render/search.go` renders grep-style text output.

**Tech Stack:** Go 1.26, Cobra, stdlib `net/http` + `encoding/json`, `net/http/httptest` for tests, lipgloss (via `cmd/render`).

## Global Constraints

- No em-dashes or non-ASCII typography anywhere (prose, code comments, commit messages). Use regular hyphen `-` and straight quotes.
- Tests live only in `pkg/bitbucket/` and `cmd/render/`. `cmd/` Cobra wiring stays untested.
- Tests use stdlib `net/http/httptest` via `newTestClient(t, handler)` from `pkg/bitbucket/testhelpers_test.go`. No mock frameworks, no assertion libraries. Test package is `bitbucket_test` (black-box).
- `json` is the built-in default output; `--format text` calls a per-command `textFn`; do not change format precedence.
- List methods follow Bitbucket pagination internally. `--limit <= 0` means "all pages".
- Bitbucket Cloud code search facts: workspace-scoped endpoint `/workspaces/{ws}/search/code`, default-branch only, token/word indexed (not regex), modifiers `ext:`, `lang:`, `repo:`, `project:` (same-kind modifiers OR-combine implicitly).
- Documentation sync is REQUIRED in the same change set as the code: `README.md`, `llms.txt`, `CLAUDE.md` (Task 6).
- The `--describe` manifest must list every leaf: add `search code/repos/prs` to `commandRegistry` and any new type to `typeRegistry`, then regenerate the golden snapshot (Task 6).

---

### Task 1: Code search types, `fetchPagesLimit`, and `Search().Code()`

**Files:**
- Modify: `pkg/bitbucket/types.go` (add code-search result types + `CodeSearchOptions`)
- Modify: `pkg/bitbucket/client.go` (add `fetchPagesLimit`, add `Search` accessor)
- Create: `pkg/bitbucket/search.go` (`SearchResource`, `Code`, `searchQuery`)
- Test: `pkg/bitbucket/search_test.go`

**Interfaces:**
- Consumes: `Client.do`, `Client.fetchPage`, `decode`, `paged[T]`, `pagelenSmall`, `parseHTTPError`, `newTestClient` (all existing).
- Produces (later tasks rely on these exact names/types):
  - `func (c *Client) Search(workspace string) *SearchResource`
  - `func (s *SearchResource) Code(ctx context.Context, opts CodeSearchOptions) ([]CodeSearchResult, error)`
  - `func fetchPagesLimit[T any](ctx context.Context, c *Client, path string, q url.Values, limit int) ([]T, error)`
  - Types: `CodeSearchResult`, `CodeSearchContentMatch`, `CodeSearchLine`, `CodeSearchSegment`, `CodeSearchFile`, `CodeSearchCommit`, `CodeSearchRepoRef`, `CodeSearchOptions` (fields `Query, Ext, Lang, Repo, Project string; Limit int`).

- [ ] **Step 1: Write the failing test**

Add to `pkg/bitbucket/search_test.go`:

```go
package bitbucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestSearchCode_BuildsQueryAndDecodes(t *testing.T) {
	var gotPath, gotQuery, gotPagelen string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("search_query")
		gotPagelen = r.URL.Query().Get("pagelen")
		mustEncodeJSON(t, w, map[string]any{
			"values": []map[string]any{
				{
					"type":                "code_search_result",
					"content_match_count": 1,
					"content_matches": []map[string]any{
						{"lines": []map[string]any{
							{"line": 10, "segments": []map[string]any{
								{"text": "func "},
								{"text": "parseConfig", "match": true},
								{"text": "() {"},
							}},
						}},
					},
					"file": map[string]any{
						"path": "src/foo.go",
						"type": "commit_file",
						"commit": map[string]any{
							"hash":       "abc123",
							"repository": map[string]any{"name": "repo", "full_name": "ws/repo"},
						},
					},
				},
			},
		})
	})
	c := newTestClient(t, handler)

	res, err := c.Search("ws").Code(context.Background(), bitbucket.CodeSearchOptions{Query: "parseConfig"})
	if err != nil {
		t.Fatalf("Code: %v", err)
	}
	if gotPath != "/workspaces/ws/search/code" {
		t.Errorf("path = %q, want /workspaces/ws/search/code", gotPath)
	}
	if gotQuery != "parseConfig" {
		t.Errorf("search_query = %q, want parseConfig", gotQuery)
	}
	if gotPagelen == "" {
		t.Errorf("pagelen not set")
	}
	if len(res) != 1 {
		t.Fatalf("got %d results, want 1", len(res))
	}
	if res[0].File.Path != "src/foo.go" {
		t.Errorf("file path = %q, want src/foo.go", res[0].File.Path)
	}
	if res[0].File.Commit == nil || res[0].File.Commit.Repository == nil ||
		res[0].File.Commit.Repository.FullName != "ws/repo" {
		t.Fatalf("repository full_name not decoded: %+v", res[0].File)
	}
	if len(res[0].ContentMatches) != 1 || len(res[0].ContentMatches[0].Lines) != 1 {
		t.Fatalf("content matches not decoded: %+v", res[0].ContentMatches)
	}
	line := res[0].ContentMatches[0].Lines[0]
	if line.Line != 10 || len(line.Segments) != 3 || !line.Segments[1].Match {
		t.Errorf("line decoded wrong: %+v", line)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestSearchCode_BuildsQueryAndDecodes -v`
Expected: FAIL - compile error, `c.Search` undefined / `bitbucket.CodeSearchOptions` undefined.

- [ ] **Step 3: Add the types to `pkg/bitbucket/types.go`**

Append at the end of the file:

```go
// Code search types

// CodeSearchResult is one match returned by the code search API.
type CodeSearchResult struct {
	Type              string                   `json:"type"`
	ContentMatchCount int                      `json:"content_match_count"`
	ContentMatches    []CodeSearchContentMatch `json:"content_matches"`
	PathMatches       []CodeSearchSegment      `json:"path_matches,omitempty"`
	File              CodeSearchFile           `json:"file"`
}

// CodeSearchContentMatch groups consecutive matched lines within a file.
type CodeSearchContentMatch struct {
	Lines []CodeSearchLine `json:"lines"`
}

// CodeSearchLine is a single line with its segments.
type CodeSearchLine struct {
	Line     int                 `json:"line"`
	Segments []CodeSearchSegment `json:"segments"`
}

// CodeSearchSegment is a run of text; Match is true when it is part of a hit.
type CodeSearchSegment struct {
	Text  string `json:"text"`
	Match bool   `json:"match,omitempty"`
}

// CodeSearchFile identifies the matched file and its origin commit/repo.
type CodeSearchFile struct {
	Path   string            `json:"path"`
	Type   string            `json:"type"`
	Commit *CodeSearchCommit `json:"commit,omitempty"`
}

// CodeSearchCommit is the commit a matched file belongs to.
type CodeSearchCommit struct {
	Hash       string             `json:"hash"`
	Repository *CodeSearchRepoRef `json:"repository,omitempty"`
}

// CodeSearchRepoRef is the minimal repository reference in a code search hit.
type CodeSearchRepoRef struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// CodeSearchOptions configures a code search. Query holds the raw search terms
// (passed through verbatim); Ext/Lang/Repo/Project are folded into Bitbucket
// search modifiers (comma-separated values produce repeated modifiers). Limit
// caps the result count; <= 0 returns all matches.
type CodeSearchOptions struct {
	Query   string
	Ext     string
	Lang    string
	Repo    string
	Project string
	Limit   int
}
```

- [ ] **Step 4: Add `fetchPagesLimit` and the `Search` accessor to `pkg/bitbucket/client.go`**

Add the accessor next to the other resource accessors (after the `Repos` method, around line 230):

```go
// Search returns a resource for workspace-scoped search operations.
func (c *Client) Search(workspace string) *SearchResource {
	return &SearchResource{client: c, workspace: workspace}
}
```

Add the helper immediately after `fetchAllPages` (end of file):

```go
// fetchPagesLimit fetches pages following "next" links until at least `limit`
// items are collected, then truncates to exactly `limit`. A limit <= 0 fetches
// every page (identical to fetchAllPages).
func fetchPagesLimit[T any](ctx context.Context, c *Client, path string, q url.Values, limit int) ([]T, error) {
	var all []T
	nextURL := ""
	for {
		var data []byte
		var err error
		if nextURL != "" {
			data, err = c.fetchPage(ctx, nextURL)
		} else {
			data, err = c.do(ctx, "GET", path, nil, q)
		}
		if err != nil {
			return nil, err
		}
		page, err := decode[paged[T]](data)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Values...)
		if limit > 0 && len(all) >= limit {
			return all[:limit], nil
		}
		if page.Next == "" {
			break
		}
		nextURL = page.Next
	}
	return all, nil
}
```

- [ ] **Step 5: Create `pkg/bitbucket/search.go`**

```go
package bitbucket

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// SearchResource provides workspace-scoped search operations.
type SearchResource struct {
	client    *Client
	workspace string
}

// Code searches file contents across the workspace's indexed default branches.
// The raw Query is passed to Bitbucket verbatim; Ext/Lang/Repo/Project are
// folded in as search modifiers. Results are capped by opts.Limit (<= 0 = all).
func (s *SearchResource) Code(ctx context.Context, opts CodeSearchOptions) ([]CodeSearchResult, error) {
	path := fmt.Sprintf("/workspaces/%s/search/code", s.workspace)
	q := url.Values{
		"search_query": {opts.searchQuery()},
		"pagelen":      {pagelenSmall},
	}
	return fetchPagesLimit[CodeSearchResult](ctx, s.client, path, q, opts.Limit)
}

// searchQuery assembles the search_query string: raw terms first, then each
// modifier. Comma-separated modifier values become repeated modifiers, which
// Bitbucket OR-combines (e.g. Ext "js,jsx" -> "ext:js ext:jsx").
func (o CodeSearchOptions) searchQuery() string {
	var parts []string
	if q := strings.TrimSpace(o.Query); q != "" {
		parts = append(parts, q)
	}
	addMod := func(key, val string) {
		for _, v := range strings.Split(val, ",") {
			if v = strings.TrimSpace(v); v != "" {
				parts = append(parts, key+":"+v)
			}
		}
	}
	addMod("ext", o.Ext)
	addMod("lang", o.Lang)
	addMod("repo", o.Repo)
	addMod("project", o.Project)
	return strings.Join(parts, " ")
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./pkg/bitbucket/ -run TestSearchCode_BuildsQueryAndDecodes -v`
Expected: PASS

- [ ] **Step 7: Add modifier, pagination/limit, and error tests**

Add `"errors"` to the import block (the APIError test below uses `errors.As`).
Then append to `pkg/bitbucket/search_test.go`:

```go
func TestSearchCode_FoldsModifiers(t *testing.T) {
	var gotQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("search_query")
		mustEncodeJSON(t, w, map[string]any{"values": []any{}})
	})
	c := newTestClient(t, handler)

	_, err := c.Search("ws").Code(context.Background(), bitbucket.CodeSearchOptions{
		Query: "parseConfig", Ext: "go,mod", Lang: "go", Repo: "api", Project: "PLAT",
	})
	if err != nil {
		t.Fatalf("Code: %v", err)
	}
	want := "parseConfig ext:go ext:mod lang:go repo:api project:PLAT"
	if gotQuery != want {
		t.Errorf("search_query = %q, want %q", gotQuery, want)
	}
}

func TestSearchCode_LimitStopsPaging(t *testing.T) {
	var calls int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		// Every page returns 2 results and always advertises a next page.
		next := "http://" + r.Host + "/workspaces/ws/search/code?search_query=x&page=" + r.URL.Query().Get("nextpage")
		mustEncodeJSON(t, w, map[string]any{
			"values": []map[string]any{
				{"file": map[string]any{"path": "a.go"}},
				{"file": map[string]any{"path": "b.go"}},
			},
			"next": next,
		})
	})
	c := newTestClient(t, handler)

	res, err := c.Search("ws").Code(context.Background(), bitbucket.CodeSearchOptions{Query: "x", Limit: 3})
	if err != nil {
		t.Fatalf("Code: %v", err)
	}
	if len(res) != 3 {
		t.Errorf("got %d results, want 3 (limit)", len(res))
	}
	if calls != 2 {
		t.Errorf("made %d requests, want 2 (2 per page, stop after limit reached)", calls)
	}
}

func TestSearchCode_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		mustEncodeJSON(t, w, map[string]any{"error": map[string]any{"message": "no such workspace"}})
	})
	c := newTestClient(t, handler)

	_, err := c.Search("ws").Code(context.Background(), bitbucket.CodeSearchOptions{Query: "x"})
	var apiErr *bitbucket.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
		t.Fatalf("want *APIError status 404, got %v", err)
	}
}
```

- [ ] **Step 8: Run all search tests**

Run: `go test ./pkg/bitbucket/ -run TestSearchCode -v`
Expected: PASS (4 tests)

- [ ] **Step 9: Commit**

```bash
git add pkg/bitbucket/types.go pkg/bitbucket/client.go pkg/bitbucket/search.go pkg/bitbucket/search_test.go
git commit -m "feat(search): add code search client with fetchPagesLimit"
```

---

### Task 2: `Search().Repos()`

**Files:**
- Modify: `pkg/bitbucket/search.go` (add `Repos` method)
- Test: `pkg/bitbucket/search_test.go`

**Interfaces:**
- Consumes: `fetchPagesLimit`, `pagelenSmall`, `Repo` (existing).
- Produces: `func (s *SearchResource) Repos(ctx context.Context, term string, limit int) ([]Repo, error)`

- [ ] **Step 1: Write the failing test**

Append to `pkg/bitbucket/search_test.go`:

```go
func TestSearchRepos_BuildsBBQLAndDecodes(t *testing.T) {
	var gotPath, gotQ string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQ = r.URL.Query().Get("q")
		mustEncodeJSON(t, w, map[string]any{
			"values": []map[string]any{
				{"slug": "payments", "name": "Payments", "description": "billing"},
			},
		})
	})
	c := newTestClient(t, handler)

	repos, err := c.Search("ws").Repos(context.Background(), "pay", 0)
	if err != nil {
		t.Fatalf("Repos: %v", err)
	}
	if gotPath != "/repositories/ws" {
		t.Errorf("path = %q, want /repositories/ws", gotPath)
	}
	want := `name ~ "pay" OR description ~ "pay"`
	if gotQ != want {
		t.Errorf("q = %q, want %q", gotQ, want)
	}
	if len(repos) != 1 || repos[0].Slug != "payments" {
		t.Fatalf("repos decoded wrong: %+v", repos)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestSearchRepos -v`
Expected: FAIL - `s.Repos` undefined.

- [ ] **Step 3: Add the `Repos` method to `pkg/bitbucket/search.go`**

```go
// Repos finds repositories in the workspace whose name or description matches
// term (BBQL "~" contains). Results are capped by limit (<= 0 = all).
func (s *SearchResource) Repos(ctx context.Context, term string, limit int) ([]Repo, error) {
	path := fmt.Sprintf("/repositories/%s", s.workspace)
	q := url.Values{
		"q":       {fmt.Sprintf(`name ~ "%s" OR description ~ "%s"`, term, term)},
		"pagelen": {pagelenSmall},
	}
	return fetchPagesLimit[Repo](ctx, s.client, path, q, limit)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/bitbucket/ -run TestSearchRepos -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/bitbucket/search.go pkg/bitbucket/search_test.go
git commit -m "feat(search): add repository name/description search"
```

---

### Task 3: PR text search (`PRListOptions.Query` + `Limit`)

**Files:**
- Modify: `pkg/bitbucket/types.go` (add `Query`, `Limit` to `PRListOptions`)
- Modify: `pkg/bitbucket/pr.go` (fold `Query` into BBQL; switch `List` to `fetchPagesLimit`)
- Test: `pkg/bitbucket/pr_test.go`

**Interfaces:**
- Consumes: `fetchPagesLimit` (Task 1), existing `PRResource.List`.
- Produces: `PRListOptions` gains `Query string` and `Limit int`. `List` honors both. Existing zero-value callers are unaffected (`Limit == 0` -> all pages).

- [ ] **Step 1: Write the failing test**

Append to `pkg/bitbucket/pr_test.go`:

```go
func TestPRList_QueryFiltersTitleAndDescription(t *testing.T) {
	var gotQ string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQ = r.URL.Query().Get("q")
		mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{{"id": 1, "title": "fix login"}}})
	})
	c := newTestClient(t, handler)

	prs, err := c.PRs("ws", "repo").List(context.Background(), bitbucket.PRListOptions{Query: "login"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := `(title ~ "login" OR description ~ "login")`
	if gotQ != want {
		t.Errorf("q = %q, want %q", gotQ, want)
	}
	if len(prs) != 1 {
		t.Fatalf("got %d PRs, want 1", len(prs))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/bitbucket/ -run TestPRList_QueryFiltersTitleAndDescription -v`
Expected: FAIL - `Query` field unknown in `PRListOptions`.

- [ ] **Step 3: Add fields to `PRListOptions` in `pkg/bitbucket/types.go`**

Replace the `PRListOptions` struct (lines 22-28) with:

```go
// PRListOptions holds the filters for listing pull requests. All fields are
// optional; the zero value lists open PRs in the endpoint's default order.
type PRListOptions struct {
	State        string // OPEN, MERGED, DECLINED, SUPERSEDED (empty = API default)
	SourceBranch string // exact source branch name to filter by
	Sort         string // Bitbucket sort field, "-" prefix for descending (e.g. -updated_on)
	Since        string // lower bound on created_on (ISO-8601); empty = no lower bound
	Until        string // upper bound on created_on (ISO-8601); empty = no upper bound
	Query        string // text matched against title/description via BBQL "~"; empty = no text filter
	Limit        int    // cap on results; <= 0 returns all pages
}
```

- [ ] **Step 4: Fold `Query` into BBQL and switch `List` to `fetchPagesLimit` in `pkg/bitbucket/pr.go`**

In `List` (lines 26-50), add a clause for `Query` after the `Until` clause and change the final return. The clause block becomes:

```go
	var clauses []string
	if opts.SourceBranch != "" {
		clauses = append(clauses, fmt.Sprintf(`source.branch.name="%s"`, opts.SourceBranch))
	}
	if opts.Since != "" {
		clauses = append(clauses, fmt.Sprintf(`created_on>="%s"`, opts.Since))
	}
	if opts.Until != "" {
		clauses = append(clauses, fmt.Sprintf(`created_on<="%s"`, opts.Until))
	}
	if opts.Query != "" {
		clauses = append(clauses, fmt.Sprintf(`(title ~ "%s" OR description ~ "%s")`, opts.Query, opts.Query))
	}
	if len(clauses) > 0 {
		q.Set("q", strings.Join(clauses, " AND "))
	}

	return fetchPagesLimit[PR](ctx, r.client, repoPath(r.workspace, r.repo)+"/pullrequests", q, opts.Limit)
```

- [ ] **Step 5: Run the new test plus the existing PR list tests to verify nothing regressed**

Run: `go test ./pkg/bitbucket/ -run TestPRList -v`
Expected: PASS (new test passes; pre-existing `List` tests still pass because `Limit == 0` fetches all pages exactly as `fetchAllPages` did).

- [ ] **Step 6: Commit**

```bash
git add pkg/bitbucket/types.go pkg/bitbucket/pr.go pkg/bitbucket/pr_test.go
git commit -m "feat(search): add text query and limit to PR list options"
```

---

### Task 4: Code search text renderer

**Files:**
- Create: `cmd/render/search.go`
- Test: `cmd/render/search_test.go`

**Interfaces:**
- Consumes: `bitbucket.CodeSearchResult`, `IDStyle`, `DimStyle`, `truncate` (existing in `cmd/render`).
- Produces: `func CodeSearchResultsString(results []bitbucket.CodeSearchResult) string` and `func CodeSearchResults(results []bitbucket.CodeSearchResult)`.

- [ ] **Step 1: Write the failing test**

Create `cmd/render/search_test.go`:

```go
package render

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestCodeSearchResultsString_GrepStyle(t *testing.T) {
	results := []bitbucket.CodeSearchResult{
		{
			File: bitbucket.CodeSearchFile{
				Path:   "src/foo.go",
				Commit: &bitbucket.CodeSearchCommit{Repository: &bitbucket.CodeSearchRepoRef{FullName: "ws/repo"}},
			},
			ContentMatches: []bitbucket.CodeSearchContentMatch{
				{Lines: []bitbucket.CodeSearchLine{
					{Line: 10, Segments: []bitbucket.CodeSearchSegment{
						{Text: "func "}, {Text: "parseConfig", Match: true}, {Text: "() {"},
					}},
				}},
			},
		},
	}
	out := CodeSearchResultsString(results)
	if !strings.Contains(out, "ws/repo/src/foo.go:10:") {
		t.Errorf("missing grep-style location header in:\n%s", out)
	}
	if !strings.Contains(out, "func parseConfig() {") {
		t.Errorf("missing reconstructed matched line in:\n%s", out)
	}
}

func TestCodeSearchResultsString_Empty(t *testing.T) {
	if got := CodeSearchResultsString(nil); got != "No code matches found.\n" {
		t.Errorf("empty render = %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/render/ -run TestCodeSearchResultsString -v`
Expected: FAIL - `CodeSearchResultsString` undefined.

- [ ] **Step 3: Create `cmd/render/search.go`**

```go
package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// CodeSearchResultsString returns grep-style text for code search results:
// one line per matched line, formatted as "<repo>/<path>:<line>: <content>".
func CodeSearchResultsString(results []bitbucket.CodeSearchResult) string {
	if len(results) == 0 {
		return "No code matches found.\n"
	}
	var sb strings.Builder
	for _, res := range results {
		loc := res.File.Path
		if c := res.File.Commit; c != nil && c.Repository != nil {
			repo := c.Repository.FullName
			if repo == "" {
				repo = c.Repository.Name
			}
			if repo != "" {
				loc = repo + "/" + res.File.Path
			}
		}
		for _, cm := range res.ContentMatches {
			for _, ln := range cm.Lines {
				var line strings.Builder
				for _, seg := range ln.Segments {
					line.WriteString(seg.Text)
				}
				text := strings.TrimRight(line.String(), "\r\n")
				sb.WriteString(fmt.Sprintf("%s %s\n",
					IDStyle.Render(fmt.Sprintf("%s:%d:", loc, ln.Line)),
					truncate(text, 200)))
			}
		}
	}
	return sb.String()
}

// CodeSearchResults prints the grep-style code search output to stdout.
func CodeSearchResults(results []bitbucket.CodeSearchResult) { fmt.Print(CodeSearchResultsString(results)) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/render/ -run TestCodeSearchResultsString -v`
Expected: PASS (2 tests). Note: `go test` stdout is not a TTY, so lipgloss renders plain text and the `strings.Contains` assertions match.

- [ ] **Step 5: Commit**

```bash
git add cmd/render/search.go cmd/render/search_test.go
git commit -m "feat(search): add grep-style code search renderer"
```

---

### Task 5: Cobra command wiring (`bb search code/repos/prs`)

**Files:**
- Create: `cmd/search.go`

**Interfaces:**
- Consumes: `workspaceOnly`, `workspaceAndRepo`, `printOutput`, `client`, `render.CodeSearchResults`, `render.RepoList`, `render.PRList`, `bitbucket.CodeSearchOptions`, `bitbucket.PRListOptions`, `SearchResource.Code`, `SearchResource.Repos`, `PRResource.List`.
- Produces: registered Cobra leaves `search code`, `search repos`, `search prs`.

- [ ] **Step 1: Create `cmd/search.go`**

```go
package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search code, repositories, and pull requests",
}

var (
	searchCodeLimit   int
	searchCodeExt     string
	searchCodeLang    string
	searchCodeRepo    string
	searchCodeProject string
)

var searchCodeCmd = &cobra.Command{
	Use:   "code <query>...",
	Short: "Search file contents across the workspace",
	Long: "Search file contents across the workspace using Bitbucket's code search index.\n\n" +
		"Notes and limits (this is not git grep):\n" +
		"  - searches the indexed default branch of each repository only\n" +
		"  - token/word based, not regex\n" +
		"  - large, binary, and generated files may not be indexed\n" +
		"  - requires the workspace to have code search enabled\n\n" +
		"The query is passed to Bitbucket verbatim, so modifiers work inline\n" +
		"(e.g. 'bb search code ext:go parseConfig'). The --ext/--lang/--repo-filter/--project\n" +
		"flags are conveniences folded into the query; comma-separated values are OR-combined.",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		results, err := client.Search(ws).Code(context.Background(), bitbucket.CodeSearchOptions{
			Query:   strings.Join(args, " "),
			Ext:     searchCodeExt,
			Lang:    searchCodeLang,
			Repo:    searchCodeRepo,
			Project: searchCodeProject,
			Limit:   searchCodeLimit,
		})
		if err != nil {
			return err
		}
		return printOutput(results, func() { render.CodeSearchResults(results) })
	},
}

var searchReposLimit int

var searchReposCmd = &cobra.Command{
	Use:   "repos <query>...",
	Short: "Search repositories by name or description",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := workspaceOnly()
		if err != nil {
			return err
		}
		repos, err := client.Search(ws).Repos(context.Background(), strings.Join(args, " "), searchReposLimit)
		if err != nil {
			return err
		}
		return printOutput(repos, func() { render.RepoList(repos) })
	},
}

var (
	searchPrsLimit int
	searchPrsState string
)

var searchPrsCmd = &cobra.Command{
	Use:   "prs <query>...",
	Short: "Search pull requests by title or description (current repo)",
	Long: "Search pull requests in the current repository by title or description.\n\n" +
		"Bitbucket has no workspace-wide PR search, so this is scoped to a single\n" +
		"repository: set --repo or a default repo via 'bb setup'. For richer\n" +
		"per-repo filtering (state, branch, dates) use 'bb pr list'.",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		prs, err := client.PRs(ws, repo).List(context.Background(), bitbucket.PRListOptions{
			Query: strings.Join(args, " "),
			State: searchPrsState,
			Limit: searchPrsLimit,
		})
		if err != nil {
			return err
		}
		return printOutput(prs, func() { render.PRList(prs) })
	},
}

func init() {
	searchCodeCmd.Flags().IntVarP(&searchCodeLimit, "limit", "L", 100, "maximum results (0 = all)")
	searchCodeCmd.Flags().StringVar(&searchCodeExt, "ext", "", "filter by file extension (comma-separated, e.g. go,mod)")
	searchCodeCmd.Flags().StringVar(&searchCodeLang, "lang", "", "filter by language (comma-separated, e.g. go,python)")
	searchCodeCmd.Flags().StringVar(&searchCodeRepo, "repo-filter", "", "limit to repository slug(s) (comma-separated)")
	searchCodeCmd.Flags().StringVar(&searchCodeProject, "project", "", "limit to project key(s) (comma-separated)")

	searchReposCmd.Flags().IntVarP(&searchReposLimit, "limit", "L", 100, "maximum results (0 = all)")

	searchPrsCmd.Flags().IntVarP(&searchPrsLimit, "limit", "L", 100, "maximum results (0 = all)")
	searchPrsCmd.Flags().StringVar(&searchPrsState, "state", "", "filter by state: OPEN, MERGED, DECLINED, SUPERSEDED")

	searchCmd.AddCommand(searchCodeCmd, searchReposCmd, searchPrsCmd)
	rootCmd.AddCommand(searchCmd)
}
```

Note on flag naming: the global persistent flag `--repo` / `-r` already exists for the target repository. To avoid a collision on the `search code` command, the code-search repository modifier is exposed as `--repo-filter` (it folds into the `repo:` modifier). This is a deliberate, documented divergence from gh's `--repo`.

- [ ] **Step 2: Build to verify wiring compiles**

Run: `go build -o bb .`
Expected: builds with no errors.

- [ ] **Step 3: Smoke-test help output**

Run: `./bb search --help` then `./bb search code --help`
Expected: `search` lists `code`, `repos`, `prs`; `search code --help` shows the limits note and the `--ext/--lang/--repo-filter/--project/--limit` flags.

- [ ] **Step 4: Commit**

```bash
git add cmd/search.go
git commit -m "feat(search): wire bb search code/repos/prs commands"
```

---

### Task 6: Manifest registration, golden snapshot, docs sync, full verification

**Files:**
- Modify: `cmd/manifest_registry.go` (registry + type entries)
- Modify: `cmd/testdata/manifest.golden.json` (regenerated, not hand-edited)
- Modify: `README.md`, `llms.txt`, `CLAUDE.md`

**Interfaces:**
- Consumes: `bitbucket.CodeSearchResult` for the type registry.
- Produces: a manifest that lists all three search leaves; passing `TestEveryLeafIsRegistered`, `TestRegistryReferencesOnlyRealCommands`, `TestManifestSnapshot`.

- [ ] **Step 1: Run the manifest invariant test to see it fail**

Run: `go test ./cmd/ -run TestEveryLeafIsRegistered -v`
Expected: FAIL - the three new `search ...` leaves are not in `commandRegistry`.

- [ ] **Step 2: Add registry entries in `cmd/manifest_registry.go`**

Add a new block to `commandRegistry` (after the `repo` block is fine):

```go
	// search -----------------------------------------------------------
	"search code":  {Action: actionRead, OutputType: "[]CodeSearchResult", Example: "bb search code parseConfig --ext go"},
	"search repos": {Action: actionRead, OutputType: "[]Repo", Example: "bb search repos payments"},
	"search prs":   {Action: actionRead, OutputType: "[]PR", Example: "bb search prs 'fix login' --state OPEN"},
```

Add to `typeRegistry`:

```go
	"CodeSearchResult":   bitbucket.CodeSearchResult{},
	"[]CodeSearchResult": []bitbucket.CodeSearchResult{},
```

- [ ] **Step 3: Verify the invariant tests pass**

Run: `go test ./cmd/ -run 'TestEveryLeafIsRegistered|TestRegistryReferencesOnlyRealCommands|TestEveryRegisteredTypeResolves' -v`
Expected: PASS

- [ ] **Step 4: Regenerate the golden manifest snapshot**

Run: `go test ./cmd/ -update`
Then: `go test ./cmd/ -run TestManifestSnapshot -v`
Expected: snapshot regenerated to include `search` leaves; test PASS. Review the diff in `cmd/testdata/manifest.golden.json` to confirm only `search`-related additions appear.

- [ ] **Step 5: Update `README.md`**

In the Commands reference block, add (next to the other top-level commands):

```
bb search code <query>...   # search file contents across the workspace (default branch, indexed)
    [--ext go,mod] [--lang go] [--repo-filter slug] [--project KEY] [--limit 100]
bb search repos <query>...  # find repos by name/description [--limit 100]
bb search prs <query>...    # find PRs by title/description in the current repo [--state OPEN] [--limit 100]
```

Add a one-line note: "Code search is workspace-wide over indexed default branches (token-based, not regex); `search prs` is scoped to a single repository."

- [ ] **Step 6: Update `llms.txt`**

Add the condensed reference mirroring README flag shapes:

```
bb search code <query>... [--ext] [--lang] [--repo-filter] [--project] [--limit] - search file contents (workspace, default branch, indexed, not regex)
bb search repos <query>... [--limit] - search repos by name/description
bb search prs <query>... [--state] [--limit] - search PRs by title/description (current repo only)
```

- [ ] **Step 7: Update `CLAUDE.md`**

In the "Command hierarchy" tree, add under the top level:

```
├── search code / repos / prs
```

In the "Client pattern" block, add:

```go
client.Search(workspace).Code(ctx, bitbucket.CodeSearchOptions{Query, Ext, Lang, Repo, Project, Limit})
client.Search(workspace).Repos(ctx, term, limit)
```

And note in the PRs line that `PRListOptions` now also carries `Query` and `Limit`.

- [ ] **Step 8: Run the full verification suite**

Run: `go build -o bb . && go vet ./... && go test ./...`
Expected: build succeeds, vet clean, all tests pass.

- [ ] **Step 9: Commit**

```bash
git add cmd/manifest_registry.go cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md
git commit -m "feat(search): register search manifest entries and sync docs"
```

---

## Self-Review

**Spec coverage:**
- `search code` native endpoint, hybrid raw-query + verified modifier flags, `--limit` default 100: Tasks 1, 5.
- `search repos` BBQL name/description: Task 2, 5.
- `search prs` repo-scoped, title/description: Tasks 3, 5.
- `fetchPagesLimit` shared helper: Task 1.
- json/gcf default, grep-style text renderer: Task 4, 5 (`printOutput` handles json/gcf automatically).
- `APIError -> CLIError` mapping: inherited (no new code); `search prs` missing-repo error comes from `workspaceAndRepo`.
- `--describe` manifest + golden: Task 6.
- Doc sync (README, llms.txt, CLAUDE.md): Task 6.
- Out of scope (commits/issues, global PR search, regex): not implemented, as specified.

**Verified modifiers:** Bitbucket Cloud supports `ext:`, `lang:`, `repo:`, `project:` (Atlassian Support docs). All four convenience flags map to real modifiers, satisfying the spec's "only ship verified flags" rule. The `--repo-filter` rename avoids collision with the global `--repo` flag.

**Placeholder scan:** No TBD/TODO/"handle edge cases"; every code step contains full code.

**Type consistency:** `CodeSearchOptions` fields (`Query, Ext, Lang, Repo, Project, Limit`) match between Task 1 (definition), Task 5 (cmd usage), and Task 6 (manifest example). `fetchPagesLimit` signature matches across Tasks 1 (def), 2, 3 (consumers). Renderer name `CodeSearchResults` matches between Task 4 (def) and Task 5 (call). `PRListOptions.Query`/`Limit` match between Task 3 (def) and Task 5 (use).
