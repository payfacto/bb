# pr create Auto-Detect (repo / workspace / source-branch) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `bb pr create` infer `--repo`/`--workspace` from the git `origin` remote and `--from-branch` from the current branch when those values are unset, so users and agents stop passing flags git already knows.

**Architecture:** A new `internal/git` package provides thin `exec` wrappers (`OriginURL`, `CurrentBranch`) plus a pure URL parser (`ParseBitbucketRemote`). Two pure resolver helpers in `cmd` (`inferWorkspaceRepo`, `inferFromBranch`) take injected git-lookup functions, fill only the empty inputs, and return human-readable notes. `prCreateCmd.RunE` is thin wiring: it calls the resolvers, applies the results, and prints notes to stderr. Config/env/flags always win; only `bitbucket.org` remotes are parsed for workspace/repo.

**Tech Stack:** Go 1.26, stdlib only (`os/exec`, `context`, `strings`), Cobra. Tests use stdlib `testing` with table-driven cases (no third-party assert lib).

## Global Constraints

- No new third-party dependencies. Git access is stdlib `os/exec` only.
- ASCII punctuation only in all code, comments, and docs. No em-dashes, en-dashes, smart quotes, or the ellipsis character.
- Inferred-value notes are written to **stderr only**; stdout must stay clean so GCF/JSON output remains parseable.
- Non-empty resolved config/flag values are **never** overridden by inference (precedence unchanged).
- Only `bitbucket.org` remotes are parsed for workspace/repo. A non-bitbucket origin skips ws/repo inference (branch inference is host-agnostic and still runs).
- No new flags, no new types, no action-class change. The `--describe` manifest golden (`cmd/testdata/manifest.golden.json`) MUST NOT change; if it does, something regressed.
- Each git invocation runs under a 3s `context.WithTimeout` so `pr create` can never hang on a wedged git.
- Docs sync (README.md, llms.txt, CLAUDE.md) is part of the final task, per the repo's REQUIRED doc-sync rule.
- Never push to git/Bitbucket without explicit user sign-off.

---

### Task 1: `internal/git` package (pure remote parser + exec wrappers)

**Files:**
- Create: `internal/git/git.go`
- Test: `internal/git/git_test.go`

**Interfaces:**
- Consumes: nothing (leaf package).
- Produces:
  - `func OriginURL() (string, error)` - trimmed URL of the `origin` remote.
  - `func CurrentBranch() (string, error)` - trimmed output of `git rev-parse --abbrev-ref HEAD` ("HEAD" when detached).
  - `func ParseBitbucketRemote(remoteURL string) (workspace, repo string, ok bool)` - pure; `ok=false` for non-bitbucket.org hosts or unparseable input.

- [ ] **Step 1: Write the failing test for `ParseBitbucketRemote`**

Create `internal/git/git_test.go`:

```go
package git_test

import (
	"testing"

	"github.com/payfacto/bb/internal/git"
)

func TestParseBitbucketRemote(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantWs  string
		wantRe  string
		wantOK  bool
	}{
		{"ssh scp with .git", "git@bitbucket.org:payfacto/bb.git", "payfacto", "bb", true},
		{"ssh scp without .git", "git@bitbucket.org:payfacto/bb", "payfacto", "bb", true},
		{"https", "https://bitbucket.org/payfacto/bb.git", "payfacto", "bb", true},
		{"https with userinfo", "https://jmadore@bitbucket.org/payfacto/bb.git", "payfacto", "bb", true},
		{"ssh url form", "ssh://git@bitbucket.org/payfacto/bb.git", "payfacto", "bb", true},
		{"trailing slash", "https://bitbucket.org/payfacto/bb/", "payfacto", "bb", true},
		{"leading and trailing whitespace", "  git@bitbucket.org:payfacto/bb.git\n", "payfacto", "bb", true},
		{"github ssh rejected", "git@github.com:payfacto/bb.git", "", "", false},
		{"github https rejected", "https://github.com/payfacto/bb.git", "", "", false},
		{"empty", "", "", "", false},
		{"no separator", "not-a-url", "", "", false},
		{"host only", "https://bitbucket.org/", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, re, ok := git.ParseBitbucketRemote(tt.url)
			if ws != tt.wantWs || re != tt.wantRe || ok != tt.wantOK {
				t.Errorf("ParseBitbucketRemote(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.url, ws, re, ok, tt.wantWs, tt.wantRe, tt.wantOK)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/git/ -run TestParseBitbucketRemote -v`
Expected: FAIL - build error (`undefined: git.ParseBitbucketRemote` / no package `git`).

- [ ] **Step 3: Write `internal/git/git.go`**

Create `internal/git/git.go`:

```go
// Package git provides minimal, dependency-free helpers for reading the
// local git context: the origin remote URL, the current branch, and a pure
// parser that extracts a Bitbucket workspace/repo from a remote URL.
package git

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// gitTimeout bounds each git invocation so callers can never hang on a wedged
// git process.
const gitTimeout = 3 * time.Second

// OriginURL returns the URL of the "origin" remote. It errors if git is
// unavailable, the working directory is not a repo, or origin is unset.
func OriginURL() (string, error) {
	return runGit("remote", "get-url", "origin")
}

// CurrentBranch returns the current branch via `git rev-parse --abbrev-ref
// HEAD`. In detached-HEAD state git prints "HEAD"; callers treat that as
// "no branch".
func CurrentBranch() (string, error) {
	return runGit("rev-parse", "--abbrev-ref", "HEAD")
}

func runGit(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ParseBitbucketRemote extracts the Bitbucket workspace and repo slug from a
// git remote URL. It recognises scp-style SSH (git@bitbucket.org:ws/repo.git),
// ssh:// URLs, and HTTPS (https://[user@]bitbucket.org/ws/repo[.git]), with or
// without a trailing ".git". ok is false for any non-bitbucket.org host or an
// unparseable URL.
func ParseBitbucketRemote(remoteURL string) (workspace, repo string, ok bool) {
	s := strings.TrimSpace(remoteURL)
	if s == "" {
		return "", "", false
	}
	// Strip scheme (https://, ssh://) if present.
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	// Strip userinfo (e.g. "git@" or "user@").
	if i := strings.Index(s, "@"); i >= 0 {
		s = s[i+1:]
	}
	// s is now "host<sep>path" where sep is ':' (scp-style) or '/' (URL-style).
	i := strings.IndexAny(s, ":/")
	if i < 0 {
		return "", "", false
	}
	host, path := s[:i], s[i+1:]
	if host != "bitbucket.org" {
		return "", "", false
	}
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/git/ -run TestParseBitbucketRemote -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "feat(git): add internal/git remote parser and lookup helpers"
```

---

### Task 2: `cmd` pure resolvers (`inferWorkspaceRepo`, `inferFromBranch`)

**Files:**
- Create: `cmd/pr_infer.go`
- Test: `cmd/pr_infer_test.go`

**Interfaces:**
- Consumes: `internal/git.ParseBitbucketRemote` (Task 1).
- Produces:
  - `func inferWorkspaceRepo(ws, repo string, getOrigin func() (string, error)) (string, string, []string)` - returns resolved ws/repo and one stderr note per inferred value.
  - `func inferFromBranch(branch string, getBranch func() (string, error)) (string, string)` - returns the resolved branch and a single note (empty string when nothing inferred).

- [ ] **Step 1: Write the failing tests**

Create `cmd/pr_infer_test.go`:

```go
package cmd

import "testing"

func okOrigin(url string) func() (string, error) {
	return func() (string, error) { return url, nil }
}

func errOrigin() (string, error) { return "", errNotARepo }

// errNotARepo is a sentinel used only by tests to simulate git failing.
var errNotARepo = errTest("not a git repository")

type errTest string

func (e errTest) Error() string { return string(e) }

func TestInferWorkspaceRepo(t *testing.T) {
	const bbURL = "git@bitbucket.org:payfacto/bb.git"

	tests := []struct {
		name       string
		ws, repo   string
		getOrigin  func() (string, error)
		wantWs     string
		wantRepo   string
		wantNotes  int
	}{
		{"both set - no lookup", "acme", "widgets", okOrigin(bbURL), "acme", "widgets", 0},
		{"repo empty - fill repo", "acme", "", okOrigin(bbURL), "acme", "bb", 1},
		{"ws empty - fill ws", "", "widgets", okOrigin(bbURL), "payfacto", "widgets", 1},
		{"both empty - fill both", "", "", okOrigin(bbURL), "payfacto", "bb", 2},
		{"both empty - github origin skips", "", "", okOrigin("git@github.com:payfacto/bb.git"), "", "", 0},
		{"both empty - origin error", "", "", errOrigin, "", "", 0},
		{"both empty - empty origin", "", "", okOrigin(""), "", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, repo, notes := inferWorkspaceRepo(tt.ws, tt.repo, tt.getOrigin)
			if ws != tt.wantWs || repo != tt.wantRepo {
				t.Errorf("got ws=%q repo=%q, want ws=%q repo=%q", ws, repo, tt.wantWs, tt.wantRepo)
			}
			if len(notes) != tt.wantNotes {
				t.Errorf("got %d notes %v, want %d", len(notes), notes, tt.wantNotes)
			}
		})
	}
}

func TestInferFromBranch(t *testing.T) {
	tests := []struct {
		name       string
		branch     string
		getBranch  func() (string, error)
		wantBranch string
		wantNote   bool
	}{
		{"branch set - no lookup", "feature/x", okOrigin("main"), "feature/x", false},
		{"empty - filled", "", okOrigin("feature/x"), "feature/x", true},
		{"empty - detached HEAD", "", okOrigin("HEAD"), "", false},
		{"empty - lookup error", "", errOrigin, "", false},
		{"empty - empty output", "", okOrigin(""), "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch, note := inferFromBranch(tt.branch, tt.getBranch)
			if branch != tt.wantBranch {
				t.Errorf("got branch=%q, want %q", branch, tt.wantBranch)
			}
			if (note != "") != tt.wantNote {
				t.Errorf("got note=%q, wantNote=%v", note, tt.wantNote)
			}
		})
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/ -run 'TestInferWorkspaceRepo|TestInferFromBranch' -v`
Expected: FAIL - build error (`undefined: inferWorkspaceRepo`, `undefined: inferFromBranch`).

- [ ] **Step 3: Write `cmd/pr_infer.go`**

Create `cmd/pr_infer.go`:

```go
package cmd

import (
	"fmt"

	"github.com/payfacto/bb/internal/git"
)

// inferWorkspaceRepo fills empty workspace/repo values from the git origin
// remote when it is a bitbucket.org URL. Non-empty inputs are never overridden
// (config/flags win). It returns the resolved values plus one note per inferred
// value for the caller to print on stderr. getOrigin is injected so this logic
// is testable without a real git checkout; an error or empty return from
// getOrigin simply means "nothing inferred".
func inferWorkspaceRepo(ws, repo string, getOrigin func() (string, error)) (string, string, []string) {
	if ws != "" && repo != "" {
		return ws, repo, nil
	}
	url, err := getOrigin()
	if err != nil || url == "" {
		return ws, repo, nil
	}
	gw, gr, ok := git.ParseBitbucketRemote(url)
	if !ok {
		return ws, repo, nil
	}
	var notes []string
	if ws == "" {
		ws = gw
		notes = append(notes, fmt.Sprintf("note: inferred --workspace=%s from git origin", gw))
	}
	if repo == "" {
		repo = gr
		notes = append(notes, fmt.Sprintf("note: inferred --repo=%s from git origin", gr))
	}
	return ws, repo, notes
}

// inferFromBranch fills an empty source-branch value from the current git
// branch. A detached HEAD (git prints "HEAD"), an error, or empty output yields
// no inference. getBranch is injected for testability. The returned note is
// empty when nothing was inferred.
func inferFromBranch(branch string, getBranch func() (string, error)) (string, string) {
	if branch != "" {
		return branch, ""
	}
	b, err := getBranch()
	if err != nil || b == "" || b == "HEAD" {
		return branch, ""
	}
	return b, fmt.Sprintf("note: inferred --from-branch=%s from current branch", b)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./cmd/ -run 'TestInferWorkspaceRepo|TestInferFromBranch' -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add cmd/pr_infer.go cmd/pr_infer_test.go
git commit -m "feat(pr): add pure workspace/repo/branch inference resolvers"
```

---

### Task 3: Wire inference into `pr create` RunE + docs sync

**Files:**
- Modify: `cmd/pr.go` (imports; `prCreateCmd.RunE`, lines ~79-121)
- Modify: `README.md` (Pull Requests section, after the code block near line 173)
- Modify: `llms.txt` (Pull Requests section, after the pr code block near line 128)
- Modify: `CLAUDE.md` (add a one-line note about `pr create` auto-detect)
- Modify: `.context/INDEX.md` (add this plan to the `plans/` list)

**Interfaces:**
- Consumes: `inferWorkspaceRepo`, `inferFromBranch` (Task 2); `git.OriginURL`, `git.CurrentBranch` (Task 1); existing `workspaceAndRepo`, `requireFlag`, `stdinInputOr`.
- Produces: no new exported symbols (RunE wiring only).

- [ ] **Step 1: Add imports to `cmd/pr.go`**

The current import block is:

```go
import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)
```

Replace it with (adds `os` and `internal/git`):

```go
import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/internal/git"
	"github.com/payfacto/bb/pkg/bitbucket"
)
```

- [ ] **Step 2: Rewrite `prCreateCmd.RunE` to run inference**

Replace the entire `RunE` body of `prCreateCmd` (currently `cmd/pr.go:79-120`) with:

```go
	RunE: func(cmd *cobra.Command, args []string) error {
		// Auto-detect workspace/repo from the git origin remote when unset.
		// Config/flags always win; notes go to stderr so stdout stays clean.
		newWs, newRepo, notes := inferWorkspaceRepo(cfg.Workspace, cfg.Repo, git.OriginURL)
		cfg.Workspace, cfg.Repo = newWs, newRepo
		for _, n := range notes {
			fmt.Fprintln(os.Stderr, n)
		}
		ws, r, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		description, err := resolveTextBody(prCreateDescription, prCreateDescriptionFile, "description", "description-file")
		if err != nil {
			return err
		}
		var input bitbucket.CreatePRInput
		consumed, err := stdinInputOr(&input, func() bitbucket.CreatePRInput {
			return bitbucket.CreatePRInput{
				Title:             prCreateTitle,
				Description:       description,
				Source:            bitbucket.NewEndpoint(prCreateFromBranch),
				Destination:       bitbucket.NewEndpoint(prCreateToBranch),
				CloseSourceBranch: prCreateCloseSource,
				Draft:             prCreateDraft,
			}
		})
		if err != nil {
			return err
		}
		if !consumed {
			// Auto-detect the source branch from the current git branch when
			// omitted, then patch the already-built input.
			if b, note := inferFromBranch(prCreateFromBranch, git.CurrentBranch); note != "" {
				prCreateFromBranch = b
				input.Source = bitbucket.NewEndpoint(b)
				fmt.Fprintln(os.Stderr, note)
			}
			if err := requireFlag("title", prCreateTitle); err != nil {
				return err
			}
			if err := requireFlag("from-branch", prCreateFromBranch); err != nil {
				return err
			}
			if err := requireFlag("to-branch", prCreateToBranch); err != nil {
				return err
			}
		}
		pr, err := client.PRs(ws, r).Create(context.Background(), input)
		if err != nil {
			return err
		}
		return printOutput(pr, func() {
			fmt.Printf("PR created: #%d — %s\n", pr.ID, pr.Links.HTML.Href)
		})
	},
```

(Note: the `PR created: #%d — %s` line already exists verbatim in the file and uses an em-dash in a Printf string; leave that pre-existing line exactly as-is to avoid an unrelated diff.)

- [ ] **Step 3: Build and run the full test suite**

Run: `go build ./... && go test ./...`
Expected: PASS. In particular `cmd` (including `TestManifestSnapshot`) is green - the manifest golden must NOT have changed since no flags/types were added.

- [ ] **Step 4: Live smoke test - branch inference + stderr routing (no PR created)**

This repo's `origin` is a github.com mirror, so ws/repo inference is correctly skipped; branch inference still runs. From the repo root:

Run: `go run . pr create --workspace x --repo y 2>/tmp/bb_stderr.txt; echo "---stderr---"; cat /tmp/bb_stderr.txt`
Expected: the command errors on the missing `--title` (validation_failed), and **stderr** contains `note: inferred --from-branch=<current-branch> from current branch`. No PR is created (title/to-branch missing). Confirms branch inference fires and the note lands on stderr, not stdout.

- [ ] **Step 5: Sync README.md**

In `README.md`, immediately after the Pull Requests fenced code block (the block ending with `bb pr add-reviewer ...`, around line 174), add this paragraph:

```markdown
`bb pr create` auto-detects `--workspace`/`--repo` from the git `origin` remote
(bitbucket.org remotes only) and `--from-branch` from the current branch when
those are omitted. Explicit config, env vars, and flags always take precedence;
each inferred value is noted on stderr so JSON/GCF output on stdout stays clean.
```

- [ ] **Step 6: Sync llms.txt**

In `llms.txt`, immediately after the Pull Requests pr code block (around line 128, after the `bb pr create ...` / `bb pr ...` lines), add this line:

```
# pr create infers --workspace/--repo from the git origin (bitbucket.org only) and --from-branch from the current branch when omitted; notes go to stderr.
```

- [ ] **Step 7: Sync CLAUDE.md**

In `CLAUDE.md`, in the Command hierarchy block, change the `pr` `list / get / create ...` context by adding a short note directly beneath the command tree's `pr` grouping. Concretely, after the closing ``` ``` ``` of the command-hierarchy code block, add this line to the surrounding prose:

```markdown
`bb pr create` infers `--workspace`/`--repo` from the git `origin` remote
(bitbucket.org only) and `--from-branch` from the current branch when omitted
(config/flags win; inference notes print to stderr). Git access lives in
`internal/git`; the pure resolvers are `inferWorkspaceRepo`/`inferFromBranch` in
`cmd/pr_infer.go`.
```

- [ ] **Step 8: Update `.context/INDEX.md`**

In `.context/INDEX.md`, under the `### plans/` list, add this entry at the top of the list:

```markdown
- [plans/2026-07-01-pr-create-autodetect.md](plans/2026-07-01-pr-create-autodetect.md) - TDD plan (3 tasks) for backlog #7: `bb pr create` auto-detect of `--workspace`/`--repo` (git origin, bitbucket.org only) and `--from-branch` (current branch); notes to stderr.
```

- [ ] **Step 9: Final verification (fmt, vet, full suite)**

Run: `gofmt -l cmd/pr.go cmd/pr_infer.go internal/git/git.go && go vet ./... && go test ./...`
Expected: `gofmt -l` prints nothing (all formatted), `go vet` clean, all tests PASS.

- [ ] **Step 10: Commit**

```bash
git add cmd/pr.go README.md llms.txt CLAUDE.md .context/INDEX.md
git commit -m "feat(pr): auto-detect repo/workspace/branch in pr create"
```

---

## Self-Review

**1. Spec coverage (design decisions -> tasks):**
- Infer `--repo` + `--workspace` from origin, each only when empty -> Task 2 `inferWorkspaceRepo` + Task 3 wiring. Covered.
- Infer `--from-branch` from current branch, detached-HEAD safe -> Task 2 `inferFromBranch` (HEAD/empty guarded) + Task 3 patch of `input.Source`. Covered.
- Only bitbucket.org remotes parsed for ws/repo -> Task 1 `ParseBitbucketRemote` (`ok=false` off-host) + tests. Covered.
- Notes on stderr, stdout clean -> Task 3 `fmt.Fprintln(os.Stderr, ...)` + Step 4 live smoke asserts stderr. Covered.
- `--to-branch` not inferred -> unchanged; still `requireFlag("to-branch")`. Covered (by omission, explicit in RunE).
- No breaking change / no manifest change -> Task 3 Step 3 asserts `TestManifestSnapshot` stays green; no flags/types added. Covered.
- stdin path unaffected -> branch inference lives inside `if !consumed`; ws/repo inference is independent of stdin. Covered.
- 3s git timeout -> Task 1 `runGit` uses `context.WithTimeout`. Covered.
- Docs sync -> Task 3 Steps 5-8 (README, llms.txt, CLAUDE.md, INDEX). Covered.

**2. Placeholder scan:** No TBD/TODO/"handle edge cases"/"similar to". All steps carry full code or exact commands.

**3. Type consistency:** `inferWorkspaceRepo(ws, repo string, getOrigin func() (string, error)) (string, string, []string)` and `inferFromBranch(branch string, getBranch func() (string, error)) (string, string)` are used identically in Task 2 (definition + tests) and Task 3 (call sites: `inferWorkspaceRepo(cfg.Workspace, cfg.Repo, git.OriginURL)`, `inferFromBranch(prCreateFromBranch, git.CurrentBranch)`). `git.OriginURL`/`git.CurrentBranch` match `func() (string, error)`. `ParseBitbucketRemote(string) (string, string, bool)` consistent between Task 1 definition, tests, and Task 2 consumer.
