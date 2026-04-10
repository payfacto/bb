# Fix "Open in Browser" on Windows

**Date:** 2026-04-10
**Status:** Approved

## Problem

The TUI's `openURLCmd` function in `cmd/tui/sections.go` only handles `darwin` and `linux`. On Windows (and all other platforms), it returns `"open URL: unsupported platform"`. This breaks all 8 "Open in browser" actions in the TUI:

- Projects list (line 117)
- Tags list (line 300)
- Commits list (line 540)
- Environments list (line 611)
- Repo detail view (line 936)
- Branches list (line 1203)
- Pull Requests list (line 1283)
- Issues list (line 1699)

A separate, correctly-implemented `openBrowser` function exists in `internal/auth/oauth.go` (used for OAuth flow) that does support Windows via `rundll32`. The two implementations drifted.

## Solution

Replace all hand-rolled browser-opening logic with the `github.com/pkg/browser` library. This library provides `browser.OpenURL(url) error` and handles darwin, windows, linux, WSL, and other edge cases.

## Changes

### 1. Add dependency

```
go get github.com/pkg/browser
```

### 2. `cmd/tui/sections.go` — rewrite `openURLCmd`

Replace the OS-switching logic with a single call to `browser.OpenURL(url)`. The function signature and return type (`tea.Cmd` returning `actionResultMsg` on error) stay the same.

Before:
```go
func openURLCmd(url string) tea.Cmd {
    return func() tea.Msg {
        if url == "" {
            return nil
        }
        var cmd *exec.Cmd
        switch runtime.GOOS {
        case "darwin":
            cmd = exec.Command("open", url)
        case "linux":
            cmd = exec.Command("xdg-open", url)
        default:
            return actionResultMsg{success: false, message: "open URL: unsupported platform"}
        }
        if err := cmd.Start(); err != nil {
            return actionResultMsg{success: false, message: fmt.Sprintf("open URL: %v", err)}
        }
        return nil
    }
}
```

After:
```go
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

### 3. `internal/auth/oauth.go` — delete `openBrowser`, use library

Delete the local `openBrowser` function (lines 170-182). Replace its single call site (line 153) with `browser.OpenURL(authURL)`.

### 4. Clean up unused imports

Remove `os/exec` and `runtime` from `cmd/tui/sections.go` if no longer used by other functions (note: `copyToClipboardCmd` still uses both, so they likely stay). Remove them from `internal/auth/oauth.go` if no longer needed.

## Out of scope

- `copyToClipboardCmd` in `cmd/tui/sections.go` has a similar OS switch for clipboard. Different concern, not broken — leave it alone.
- No new tests. The TUI and OAuth paths are not unit-tested today; browser-opening is inherently side-effectful and covered by the library's own tests.
