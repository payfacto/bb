# Fix "Open in Browser" on Windows — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace hand-rolled browser-opening code with `github.com/pkg/browser` so "Open in browser" works on Windows (and all other platforms).

**Architecture:** Two files contain independent browser-opening logic with OS switch statements. Both get replaced by a single library call. No new internal packages needed.

**Tech Stack:** Go, `github.com/pkg/browser`, Cobra CLI, charmbracelet/bubbletea TUI

---

### Task 1: Add `github.com/pkg/browser` dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Install the dependency**

Run:
```bash
cd C:/claudecode/bb && go get github.com/pkg/browser
```

Expected: `go.mod` gains a new `require` entry for `github.com/pkg/browser`, `go.sum` is updated.

- [ ] **Step 2: Verify it resolved**

Run:
```bash
cd C:/claudecode/bb && grep "pkg/browser" go.mod
```

Expected: A line like `github.com/pkg/browser v0.0.0-...` in the require block.

- [ ] **Step 3: Commit**

```bash
cd C:/claudecode/bb && git add go.mod go.sum && git commit -m "deps: add github.com/pkg/browser for cross-platform URL opening"
```

---

### Task 2: Replace `openURLCmd` in TUI with `browser.OpenURL`

**Files:**
- Modify: `cmd/tui/sections.go:3-15` (imports)
- Modify: `cmd/tui/sections.go:848-868` (`openURLCmd` function)

- [ ] **Step 1: Add the `browser` import**

In `cmd/tui/sections.go`, add `"github.com/pkg/browser"` to the third-party import group (after the `bubbletea` import, before the `payfacto` imports). The import block should look like:

```go
import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/internal/config"
```

Note: `os/exec` and `runtime` stay — `copyToClipboardCmd` (line 871+) still uses them.

- [ ] **Step 2: Rewrite `openURLCmd`**

Replace the entire `openURLCmd` function (lines 848-868) with:

```go
// openURLCmd opens url in the system default browser.
func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if url == "" {
			return nil
		}
		if err := browser.OpenURL(url); err != nil {
			return actionResultMsg{success: false, message: fmt.Sprintf("open URL: %v", err)}
		}
		return nil
	}
}
```

- [ ] **Step 3: Verify it compiles**

Run:
```bash
cd C:/claudecode/bb && go build -o bb .
```

Expected: Clean build, no errors.

- [ ] **Step 4: Commit**

```bash
cd C:/claudecode/bb && git add cmd/tui/sections.go && git commit -m "fix: use pkg/browser in TUI openURLCmd for Windows support"
```

---

### Task 3: Replace `openBrowser` in OAuth with `browser.OpenURL`

**Files:**
- Modify: `internal/auth/oauth.go:1-17` (imports)
- Modify: `internal/auth/oauth.go:153` (call site)
- Delete: `internal/auth/oauth.go:170-182` (`openBrowser` function)

- [ ] **Step 1: Add the `browser` import and remove unused imports**

In `internal/auth/oauth.go`, add `"github.com/pkg/browser"` to the import block and remove `"os/exec"` and `"runtime"` (only `openBrowser` uses them). The import block should become:

```go
import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/browser"
)
```

- [ ] **Step 2: Replace the call site**

On line 153, change:

```go
if err := openBrowser(authURL); err != nil {
```

to:

```go
if err := browser.OpenURL(authURL); err != nil {
```

- [ ] **Step 3: Delete the `openBrowser` function**

Delete the entire function (lines 170-182):

```go
// openBrowser attempts to open url in the user's default browser.
func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}
```

- [ ] **Step 4: Verify it compiles**

Run:
```bash
cd C:/claudecode/bb && go build -o bb .
```

Expected: Clean build, no errors.

- [ ] **Step 5: Run existing tests**

Run:
```bash
cd C:/claudecode/bb && go test ./...
```

Expected: All tests pass. (No tests directly exercise `openBrowser` or `openURLCmd`, but this confirms nothing else broke.)

- [ ] **Step 6: Commit**

```bash
cd C:/claudecode/bb && git add internal/auth/oauth.go && git commit -m "fix: use pkg/browser in OAuth flow, remove hand-rolled openBrowser"
```

---

### Task 4: Final verification

- [ ] **Step 1: Clean build**

Run:
```bash
cd C:/claudecode/bb && go build -o bb .
```

Expected: Clean build.

- [ ] **Step 2: Run all tests**

Run:
```bash
cd C:/claudecode/bb && go test ./...
```

Expected: All tests pass.

- [ ] **Step 3: Verify no remaining hand-rolled browser logic**

Run:
```bash
cd C:/claudecode/bb && grep -rn "xdg-open\|rundll32\|exec.Command.*open" --include="*.go" .
```

Expected: No matches (only `copyToClipboardCmd` references `exec.Command` but not for browser opening).

- [ ] **Step 4: Spot-check the binary on Windows**

Run `bb` in TUI mode, navigate to a repo, and use "Open in browser". It should open the default browser to the Bitbucket repo page.
