# Bitbucket API Token Authentication — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add first-class Bitbucket Cloud API-token authentication to `bb` before app passwords stop working on 2026-06-09.

**Architecture:** API tokens use HTTP Basic auth (`Basic base64(atlassian_email:api_token)`) — mechanically identical to app passwords. We introduce a third `auth_type` value, `apitoken`, store the Atlassian email in the existing `Config.Username` field, and **leave the HTTP client untouched** (it already emits Basic auth for any non-OAuth config). The work is config semantics, setup UX (CLI + TUI), a deprecation notice in `bb auth status`, tests, and docs.

**Tech Stack:** Go, Cobra (CLI), Bubble Tea + lipgloss (TUI), stdlib `net/http/httptest` for tests. Build: `go build -o bb .` / `make build`. Test: `go test ./...`.

**Branch:** `feat/api-token-auth` (already checked out).

**Spec:** `docs/superpowers/specs/2026-05-29-api-token-auth-design.md`

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `internal/config/config.go` | Config model + auth helpers | Add `IsLegacyAppPassword()` |
| `internal/config/config_auth_test.go` | Config unit tests | Add classification test |
| `pkg/bitbucket/client.go` | HTTP client + auth header | **No change** (verified) |
| `pkg/bitbucket/client_auth_test.go` | Auth header tests | Add api-token Basic-auth assertion |
| `cmd/setup.go` | `bb setup` CLI wizard | Default to API token; relabel; pointer URL |
| `cmd/tui/setup.go` | TUI setup wizard | Relabel fields; write `apitoken` |
| `cmd/render/infra.go` | Auth-status rendering | Add `Deprecation` field + line |
| `cmd/render/infra_test.go` | Render unit test | New: assert deprecation line |
| `cmd/auth.go` | `bb auth status` | Map auth labels; compute deprecation |
| `README.md`, `llms.txt`, `CLAUDE.md` | Docs | Auth methods: 2 → 3, app password deprecated |

> **Note on `cmd/`:** per `CLAUDE.md`, `cmd/` is intentionally untested (thin Cobra wiring). Testable logic lives in `internal/config` (Task 1) and `cmd/render` (Task 5), which get TDD coverage. `cmd/setup.go`, `cmd/tui/setup.go`, and `cmd/auth.go` changes are verified by build + manual run.

---

## Task 1: Config helper `IsLegacyAppPassword()`

**Files:**
- Modify: `internal/config/config.go` (add method after `HasOAuth`, ~line 49)
- Test: `internal/config/config_auth_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/config/config_auth_test.go`:

```go
func TestIsLegacyAppPassword(t *testing.T) {
	cases := []struct {
		name     string
		authType string
		want     bool
	}{
		{"empty legacy default", "", true},
		{"explicit apppassword", "apppassword", true},
		{"apitoken", "apitoken", false},
		{"oauth", "oauth", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{AuthType: tc.authType}
			if got := cfg.IsLegacyAppPassword(); got != tc.want {
				t.Errorf("IsLegacyAppPassword() for %q: got %v, want %v", tc.authType, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestIsLegacyAppPassword -v`
Expected: FAIL — `cfg.IsLegacyAppPassword undefined (type *config.Config has no field or method IsLegacyAppPassword)`

- [ ] **Step 3: Write minimal implementation**

In `internal/config/config.go`, immediately after the `HasOAuth` method (around line 49):

```go
// IsLegacyAppPassword reports whether cfg's auth_type denotes the deprecated
// app-password method — the historical default (empty auth_type) or the
// explicit "apppassword". It classifies the auth_type only and does NOT check
// for a token; callers that warn the user (e.g. bb auth status) should also
// confirm a credential exists before surfacing the 2026-06-09 deprecation
// notice. API token ("apitoken") and OAuth ("oauth") configs return false.
func (cfg *Config) IsLegacyAppPassword() bool {
	return cfg.AuthType == "" || cfg.AuthType == "apppassword"
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestIsLegacyAppPassword -v`
Expected: PASS (all 4 subtests)

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_auth_test.go
git commit -m "feat(config): add IsLegacyAppPassword auth_type classifier

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Verify API-token Basic auth at the client layer

The HTTP client needs **no change** — this task adds a regression test proving an `apitoken` config (Atlassian email + API token) produces `Basic base64(email:token)` and no Bearer header.

**Files:**
- Test: `pkg/bitbucket/client_auth_test.go` (append)
- Modify: none

- [ ] **Step 1: Write the failing test**

Append to `pkg/bitbucket/client_auth_test.go` (note: this file imports `strings` and `net/http/httptest` already; add `encoding/base64` to the import block):

```go
func TestClientAPITokenUsesEmailBasicAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"account_id":"123","username":"user","display_name":"User"}`))
	}))
	defer srv.Close()

	// API token auth: username field holds the Atlassian email, token is the API token.
	cfg := &config.Config{
		Username: "jmadore@payfacto.com",
		Token:    "api-token-123",
		AuthType: "apitoken",
	}
	c := bitbucket.NewWithBaseURL(cfg, srv.URL)

	_, _ = c.User().Me(context.Background())

	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatalf("Authorization header: got %q, want Basic ...", gotAuth)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(gotAuth, "Basic "))
	if err != nil {
		t.Fatalf("decode basic auth: %v", err)
	}
	if string(raw) != "jmadore@payfacto.com:api-token-123" {
		t.Errorf("basic auth payload: got %q, want %q", string(raw), "jmadore@payfacto.com:api-token-123")
	}
}
```

Update the import block at the top of the file to include `encoding/base64`:

```go
import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)
```

- [ ] **Step 2: Run test to verify it passes immediately (no production change needed)**

Run: `go test ./pkg/bitbucket/ -run TestClientAPITokenUsesEmailBasicAuth -v`
Expected: PASS. (This is a characterization test — it confirms the existing client already does the right thing for `apitoken`. If it fails, STOP: the no-client-change assumption is wrong.)

- [ ] **Step 3: Commit**

```bash
git add pkg/bitbucket/client_auth_test.go
git commit -m "test(client): assert apitoken config yields email Basic auth

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: `bb setup` CLI defaults to API token

**Files:**
- Modify: `cmd/setup.go:38-67` (prompt block + auth_type + keyring message)

- [ ] **Step 1: Update the prompt block and auth_type**

In `cmd/setup.go`, replace the current prompt + auth_type lines (currently lines 38–46):

```go
	ws := promptLine(r, "Workspace", existing.Workspace)
	defaultRepo := promptLine(r, "Default repo (optional)", existing.Repo)
	user := promptLine(r, "Username (email)", existing.Username)
	tok := promptPassword("App password (token)", existing.Token)

	authType := existing.AuthType
	if tok != "" {
		authType = "apppassword"
	}
```

with:

```go
	ws := promptLine(r, "Workspace", existing.Workspace)
	defaultRepo := promptLine(r, "Default repo (optional)", existing.Repo)
	fmt.Println("Create an API token (with scopes) at https://id.atlassian.com/manage-profile/security/api-tokens")
	user := promptLine(r, "Atlassian account email", existing.Username)
	tok := promptPassword("API token", existing.Token)

	authType := existing.AuthType
	if tok != "" {
		authType = "apitoken"
	}
```

- [ ] **Step 2: Update the keyring success message**

In `cmd/setup.go`, the success branch after `auth.SetToken(user, tok)` (currently around line 65) prints `"App password stored in OS keyring."`. Replace that string with:

```go
				fmt.Println("API token stored in OS keyring.")
```

(Leave the warning text in the error branch generic; if it mentions "app password", change it to "API token".)

- [ ] **Step 3: Build to verify it compiles**

Run: `go build -o bb .`
Expected: builds with no errors.

- [ ] **Step 4: Manual verification**

Run: `go run . setup` and confirm the prompts read `Atlassian account email` and `API token`, the id.atlassian.com pointer prints, and the written `~/.bbcloud.yaml` contains `auth_type: apitoken`. (Use a throwaway `--config` path: `go run . --config ./tmp-bbcloud.yaml setup`, then inspect the file, then delete it.)

- [ ] **Step 5: Commit**

```bash
git add cmd/setup.go
git commit -m "feat(setup): default bb setup to API token auth

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: TUI setup wizard relabel + apitoken

**Files:**
- Modify: `cmd/tui/setup.go` — field placeholders (lines 65-74), auth_type (lines 207-210), view labels (line 267)

- [ ] **Step 1: Relabel the identity and secret fields**

In `cmd/tui/setup.go`, in `newSetupView`, change the username field placeholder (line 66) and password field placeholder (line 71):

```go
	fields[setupFieldUsername] = textinput.New()
	fields[setupFieldUsername].Placeholder = "Atlassian account email"
	fields[setupFieldUsername].SetValue(existing.Username)
	fields[setupFieldUsername].CharLimit = setupFieldCharLimit

	fields[setupFieldPassword] = textinput.New()
	fields[setupFieldPassword].Placeholder = "API token"
	fields[setupFieldPassword].EchoMode = textinput.EchoPassword
	fields[setupFieldPassword].EchoCharacter = '*'
	fields[setupFieldPassword].CharLimit = setupPasswordCharLimit
```

- [ ] **Step 2: Write `apitoken` on save**

In `cmd/tui/setup.go`, in the `save()` closure (currently lines 207-210), change the auth_type assignment:

```go
		authType := existing.AuthType
		if pass != "" {
			authType = "apitoken"
		}
```

- [ ] **Step 3: Update the view labels**

In `cmd/tui/setup.go`, in `View()` (line 267), change the labels slice:

```go
	labels := []string{"Workspace", "Default repo", "Email", "API token"}
```

- [ ] **Step 4: Build to verify it compiles**

Run: `go build -o bb .`
Expected: builds with no errors.

- [ ] **Step 5: Manual verification**

Run: `go run . --config ./tmp-bbcloud.yaml` (no subcommand → TUI). If unconfigured it opens the setup wizard; confirm the field labels read `Email` and `API token`. Save and confirm the temp config has `auth_type: apitoken`. Delete the temp file afterward.

- [ ] **Step 6: Commit**

```bash
git add cmd/tui/setup.go
git commit -m "feat(tui): relabel setup wizard for API token auth

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Auth-status deprecation notice

Render a distinct deprecation line in `bb auth status` for legacy app-password users, and map the auth-type label to friendly text. The renderer gets a new `Deprecation` field (TDD-covered); `cmd/auth.go` computes it.

**Files:**
- Modify: `cmd/render/infra.go:240-260` (`AuthStatusInfo` + `AuthStatusString`)
- Create: `cmd/render/infra_test.go`
- Modify: `cmd/auth.go:118-160` (`authStatusCmd`)

- [ ] **Step 1: Write the failing render test**

Create `cmd/render/infra_test.go`:

```go
package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
)

func TestAuthStatusShowsDeprecationWhenSet(t *testing.T) {
	out := render.AuthStatusString(render.AuthStatusInfo{
		Username:    "jmadore@payfacto.com",
		Workspace:   "payfacto",
		AuthType:    "app password",
		TokenStatus: "abcd****wxyz (from OS keyring)",
		Deprecation: "DEPRECATED — app passwords stop working 2026-06-09",
	})
	if !strings.Contains(out, "2026-06-09") {
		t.Errorf("expected deprecation notice in output, got:\n%s", out)
	}
}

func TestAuthStatusOmitsDeprecationWhenEmpty(t *testing.T) {
	out := render.AuthStatusString(render.AuthStatusInfo{
		Username:    "jmadore@payfacto.com",
		Workspace:   "payfacto",
		AuthType:    "API token",
		TokenStatus: "abcd****wxyz (from OS keyring)",
		Deprecation: "",
	})
	if strings.Contains(out, "DEPRECATED") {
		t.Errorf("did not expect deprecation notice, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/render/ -run TestAuthStatus -v`
Expected: FAIL — `unknown field 'Deprecation' in struct literal of type render.AuthStatusInfo`

- [ ] **Step 3: Add the `Deprecation` field and render line**

In `cmd/render/infra.go`, add the field to `AuthStatusInfo` (after `TokenStatus`):

```go
// AuthStatusInfo holds pre-resolved auth status fields for rendering.
type AuthStatusInfo struct {
	Username    string
	Workspace   string
	AuthType    string
	TokenStatus string // e.g. "abcd****efgh (from OS keyring)"
	Deprecation string // optional; non-empty renders a warning line
}
```

Then in `AuthStatusString`, before the final `return`, append the deprecation line when set:

```go
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Token"), info.TokenStatus))
	if info.Deprecation != "" {
		sb.WriteString("\n")
		sb.WriteString(errorStyle.Render("  " + info.Deprecation))
		sb.WriteString("\n")
	}
	return sb.String()
```

> Verify `errorStyle` is exported/visible in package `render`. If the package's warning style has a different name, use the existing one (grep `cmd/render/styles.go` for a red/warning lipgloss style such as `errorStyle` or `badgeDeclined`). If only `badgeDeclined` exists, use `badgeDeclined.Render("  " + info.Deprecation)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/render/ -run TestAuthStatus -v`
Expected: PASS (both subtests)

- [ ] **Step 5: Wire the label mapping + deprecation into `bb auth status`**

In `cmd/auth.go`, in `authStatusCmd`'s `RunE`, replace the auth-method block (currently lines 133-136):

```go
		authMethod := existing.AuthType
		if authMethod == "" {
			authMethod = "apppassword (legacy)"
		}
```

with a friendly mapping:

```go
		var authMethod string
		switch existing.AuthType {
		case "apitoken":
			authMethod = "API token"
		case "oauth":
			authMethod = "OAuth 2.0"
		default: // "" or "apppassword"
			authMethod = "app password"
		}
```

Then compute the deprecation string after `tokenStatus` is resolved (after the `switch` that sets `tokenStatus`, before the `render.AuthStatus(...)` call). A token is present when the keyring returned one (`err == nil`) or `BITBUCKET_TOKEN` is set:

```go
		var deprecation string
		hasToken := err == nil || os.Getenv("BITBUCKET_TOKEN") != ""
		if existing.IsLegacyAppPassword() && hasToken {
			deprecation = "DEPRECATED — app passwords stop working 2026-06-09; run 'bb setup' to switch to an API token"
		}
```

> `err` here is the variable already assigned by `tok, err := auth.GetToken(existing.Username)` earlier in the function (line ~139). Confirm it is still in scope at this point; it is, since the `switch` uses it.

Finally add `Deprecation: deprecation,` to the `render.AuthStatusInfo{...}` literal:

```go
		render.AuthStatus(render.AuthStatusInfo{
			Username:    existing.Username,
			Workspace:   existing.Workspace,
			AuthType:    authMethod,
			TokenStatus: tokenStatus,
			Deprecation: deprecation,
		})
```

- [ ] **Step 6: Build + manual verification**

Run: `go build -o bb .`
Expected: compiles.

Manual: create a temp config with `auth_type: apppassword` and a keyring/env token, run `go run . --config ./tmp-bbcloud.yaml auth status`, and confirm the deprecation line appears. Then set `auth_type: apitoken` and confirm it does NOT appear and the auth type reads `API token`. Delete the temp file.

- [ ] **Step 7: Commit**

```bash
git add cmd/render/infra.go cmd/render/infra_test.go cmd/auth.go
git commit -m "feat(auth): show app-password deprecation notice in auth status

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Documentation sync (required by CLAUDE.md)

Update the three docs that `CLAUDE.md` mandates whenever auth behavior changes. No command was added/renamed, so the `--describe` manifest and golden snapshot do not change.

**Files:**
- Modify: `README.md` (authentication section + `bb setup` walkthrough + CI env-var note)
- Modify: `llms.txt` (auth method reference line)
- Modify: `CLAUDE.md` (auth-methods narrative, if present)

- [ ] **Step 1: Locate the auth content in each doc**

Run:
```bash
grep -n -i "app password\|auth_type\|authentication\|BITBUCKET_TOKEN\|bb setup\|oauth" README.md
grep -n -i "app password\|auth\|token\|setup" llms.txt
grep -n -i "app password\|authentication\|auth_type" CLAUDE.md
```
Expected: line numbers for the auth sections in each file. Read the surrounding blocks before editing.

- [ ] **Step 2: Update README.md**

In the authentication section, change the description from two methods (app password, OAuth) to three, and mark app password deprecated. Use this content (adapt wording to match the doc's existing voice/heading style):

```markdown
## Authentication

`bb` supports three authentication methods:

- **API token (recommended).** Create an API token *with scopes* at
  <https://id.atlassian.com/manage-profile/security/api-tokens>, then run
  `bb setup` and enter your **Atlassian account email** plus the token.
  Stored in your OS keyring; sent as HTTP Basic auth.
- **OAuth 2.0.** Browser-based flow via `bb auth login` (requires an OAuth
  consumer). Sent as a Bearer token.
- **App password (DEPRECATED).** Atlassian disables app passwords on
  **2026-06-09**. Existing app-password configs keep working until then and show
  a deprecation notice in `bb auth status`. Migrate by running `bb setup` with
  an API token.

### Non-interactive / CI

Set the Atlassian email and API token via environment variables — no `bb setup`
needed:

```bash
export BITBUCKET_USER="you@example.com"   # Atlassian account email
export BITBUCKET_TOKEN="<api-token>"      # API token with scopes
```

The `--username` and `--token` flags override these per-invocation.
```

Update the `bb setup` walkthrough so its prompts read "Atlassian account email" and "API token".

- [ ] **Step 3: Update llms.txt**

Update the auth reference line(s) so they describe API token as the primary method: email + API-token via `bb setup`, env vars `BITBUCKET_USER` (email) / `BITBUCKET_TOKEN` (API token), OAuth via `bb auth login`, and app password marked deprecated (2026-06-09). Match the file's existing terse one-line-per-item style.

- [ ] **Step 4: Update CLAUDE.md**

If `CLAUDE.md` contains a narrative describing the auth methods, update it from two to three methods and mark app password deprecated. The "Command hierarchy" tree does NOT change (no new/renamed commands). If no auth narrative exists in CLAUDE.md, make no change and note that in the commit body.

- [ ] **Step 5: Verify the full suite still passes**

Run: `go test ./...`
Expected: PASS (including the unchanged `--describe` manifest snapshot and `TestEveryLeafIsRegistered`).

- [ ] **Step 6: Commit**

```bash
git add README.md llms.txt CLAUDE.md
git commit -m "docs: document API token auth, deprecate app passwords

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Final verification

- [ ] **Run the full test suite**

Run: `go test ./...`
Expected: all packages PASS.

- [ ] **Vet and build**

Run: `go vet ./... && go build -o bb .`
Expected: no vet warnings; binary builds.

- [ ] **End-to-end smoke (throwaway config)**

```bash
go run . --config ./tmp-bbcloud.yaml setup        # enter email + API token
go run . --config ./tmp-bbcloud.yaml auth status   # auth type = API token, no deprecation line
go run . --config ./tmp-bbcloud.yaml user me        # confirms the token actually authenticates
rm ./tmp-bbcloud.yaml
```
Expected: `auth status` shows `API token`; `user me` returns your account. (Requires a real API token + network.)

---

## Self-Review (completed during planning)

- **Spec coverage:** §1 config → Task 1; §2 client-no-change → Task 2 (regression test); §3 setup CLI → Task 3; §4 TUI → Task 4; §5 deprecation/status → Task 5; §6 non-interactive → Task 6 README; §7 tests → Tasks 1,2,5 + Final; §8 docs → Task 6. No gaps.
- **Placeholder scan:** every code step shows complete code; no TBD/TODO.
- **Type consistency:** `IsLegacyAppPassword()` (Task 1) is the exact name used in Task 5; `AuthStatusInfo.Deprecation` (Task 5 Step 3) matches its use in Step 1 test and Step 5 wiring; `auth_type` value `"apitoken"` consistent across Tasks 1–5.
- **Known refinement vs spec:** the spec's illustrative `IsLegacyAppPassword()` also checked `Token != ""`. Implementation classifies on `auth_type` only because `bb auth status` resolves the token separately (via keyring); the caller (Task 5 Step 5) combines the classifier with a `hasToken` check to preserve the spec's intent (warn only configured legacy users).
