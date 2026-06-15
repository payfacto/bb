# Secret Input Reveal-Then-Mask Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When entering a secret (API token, app password, OAuth consumer secret), show the input in cleartext while typing/pasting and re-mask the whole field to `*` after 2 seconds of idle — in both the CLI prompts and the TUI setup wizard.

**Architecture:** Approach 1 (native idiom per surface). TUI toggles the bubbles `textinput` `EchoMode` and uses `tea.Tick` as a debounce; the CLI uses a self-contained raw-mode rune reader with a 2s idle `time.Timer`. Both render through one pure, unit-tested helper `render.SecretLine` that builds an ANSI-free redraw string.

**Tech Stack:** Go, `golang.org/x/term` (raw mode), `github.com/charmbracelet/bubbles/textinput` + `bubbletea` (TUI), `github.com/charmbracelet/lipgloss` (render package). Build: `go build -o bb .`. Test: `go test ./...`.

**Branch:** `feat/secret-input-reveal` (already checked out off updated `main`).

**Spec:** `docs/superpowers/specs/2026-05-29-secret-input-reveal-design.md`

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `cmd/render/secretline.go` | Pure ANSI-free redraw-string builder | Create |
| `cmd/render/secretline_test.go` | Unit tests for `SecretLine` | Create |
| `cmd/secretinput.go` | CLI raw-mode reveal-then-mask reader | Create |
| `cmd/setup.go` | `promptPassword` delegates to the reader on a TTY | Modify (`promptPassword` only) |
| `cmd/tui/setup.go` | TUI reveal-on-key + mask-on-tick/blur | Modify |

> **Testing convention:** per `CLAUDE.md`, `cmd/` wiring and `cmd/tui` are intentionally not unit-tested, but `cmd/render` IS (it has tests). So the *pure* logic (`SecretLine`) is TDD-tested; the imperative raw-mode loop (`cmd/secretinput.go`) and the TUI `Update` changes are verified by build + manual run. No changes to `pkg/bitbucket`.

> **No doc-sync:** no command or flag is added/removed/renamed, so README/llms.txt/CLAUDE.md need no updates and the `--describe` manifest snapshot is unaffected.

---

## Task 1: Pure redraw-string helper `render.SecretLine`

**Files:**
- Create: `cmd/render/secretline.go`
- Test: `cmd/render/secretline_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/render/secretline_test.go`:

```go
package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
)

func TestSecretLineGrowing(t *testing.T) {
	// prevCells 0 → no padding, no backspaces.
	out, cells := render.SecretLine("Token: ", "ab", 0)
	if out != "\rToken: ab" {
		t.Errorf("out = %q, want %q", out, "\rToken: ab")
	}
	if want := len([]rune("Token: ab")); cells != want {
		t.Errorf("cells = %d, want %d", cells, want)
	}
}

func TestSecretLineShrinkingPadsAndBackspaces(t *testing.T) {
	// Previous line "Token: ab" = 9 cells; now shown shrinks to "a" → 8 cells.
	// One leftover char must be erased: one space then one backspace.
	out, cells := render.SecretLine("Token: ", "a", 9)
	want := "\rToken: a" + " " + "\b"
	if out != want {
		t.Errorf("out = %q, want %q", out, want)
	}
	if cells != 8 {
		t.Errorf("cells = %d, want 8", cells)
	}
}

func TestSecretLineEqualLengthNoPad(t *testing.T) {
	// "P: " (3) + "xy" (2) = 5 cells, prev 5 → no pad/backspace.
	out, cells := render.SecretLine("P: ", "xy", 5)
	if strings.Contains(out, " \b") {
		t.Errorf("expected no pad/backspace, got %q", out)
	}
	if out != "\rP: xy" {
		t.Errorf("out = %q, want %q", out, "\rP: xy")
	}
	if cells != 5 {
		t.Errorf("cells = %d, want 5", cells)
	}
}

func TestSecretLineEmptyShownErasesPrevious(t *testing.T) {
	// "Token: " = 7 cells, prev 9 → pad 2 spaces + 2 backspaces.
	out, cells := render.SecretLine("Token: ", "", 9)
	want := "\rToken: " + strings.Repeat(" ", 2) + strings.Repeat("\b", 2)
	if out != want {
		t.Errorf("out = %q, want %q", out, want)
	}
	if cells != 7 {
		t.Errorf("cells = %d, want 7", cells)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/render/ -run TestSecretLine -v`
Expected: FAIL — `undefined: render.SecretLine` (build error).

- [ ] **Step 3: Implement the helper**

Create `cmd/render/secretline.go`:

```go
package render

import "strings"

// SecretLine builds the ANSI-free terminal redraw string for a single-line
// secret prompt. The rendered line is prompt+shown. prevCells is the rune width
// of the previously rendered line; when the new line is shorter, SecretLine
// pads with spaces to erase the leftover characters and then emits the same
// number of backspaces so the cursor ends at the end of the visible text.
// It returns the string to write and the new cell count to pass as prevCells
// on the next call. It emits no ANSI escape sequences (only '\r', ' ', '\b'),
// so it is safe on terminals without VT processing (e.g. legacy Windows cmd).
func SecretLine(prompt, shown string, prevCells int) (string, int) {
	line := prompt + shown
	cells := len([]rune(line))
	pad := 0
	if prevCells > cells {
		pad = prevCells - cells
	}
	out := "\r" + line + strings.Repeat(" ", pad) + strings.Repeat("\b", pad)
	return out, cells
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/render/ -run TestSecretLine -v`
Expected: PASS (all 4 subtests).

- [ ] **Step 5: Commit**

```bash
git add cmd/render/secretline.go cmd/render/secretline_test.go
git commit -m "feat(render): add ANSI-free SecretLine redraw helper

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: CLI raw-mode reveal reader + wire `promptPassword`

**Files:**
- Create: `cmd/secretinput.go`
- Modify: `cmd/setup.go` (`promptPassword` only)

(`cmd/` is not unit-tested per repo convention — verify with build, vet, and a manual run.)

- [ ] **Step 1: Create the raw-mode reader**

Create `cmd/secretinput.go`:

```go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/payfacto/bb/cmd/render"
)

// secretRevealDelay is how long after the last keystroke a revealed secret
// stays in cleartext before re-masking to '*'. Keep in sync with the TUI's
// secretRevealDelay in cmd/tui/setup.go.
const secretRevealDelay = 2 * time.Second

// readSecretRevealing reads a secret from in (which the caller has confirmed is
// a terminal), echoing it in cleartext while the user types or pastes and
// re-masking the whole field to '*' after secretRevealDelay of idle time. It
// returns the entered secret without a trailing newline. On Ctrl-C it restores
// the terminal and exits with code 130. If raw mode cannot be enabled it falls
// back to a plain hidden read via term.ReadPassword (no reveal).
func readSecretRevealing(in *os.File, prompt string) string {
	fd := int(in.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback: no reveal, plain hidden read.
		fmt.Print(prompt)
		b, _ := term.ReadPassword(fd)
		fmt.Println()
		return string(b)
	}
	defer term.Restore(fd, oldState)

	out := os.Stdout

	// Reader goroutine: forward runes until the stream errors/EOFs, then close.
	runeCh := make(chan rune)
	go func() {
		r := bufio.NewReader(in)
		for {
			ch, _, rerr := r.ReadRune()
			if rerr != nil {
				close(runeCh)
				return
			}
			runeCh <- ch
		}
	}()

	var buf []rune
	prevCells := 0
	draw := func(shown string) {
		s, cells := render.SecretLine(prompt, shown, prevCells)
		fmt.Fprint(out, s)
		prevCells = cells
	}

	// Initial render shows the prompt before any input.
	draw("")

	idle := time.NewTimer(secretRevealDelay)
	idle.Stop()

	for {
		select {
		case ch, ok := <-runeCh:
			if !ok { // stream closed / EOF → submit what we have
				draw(strings.Repeat("*", len(buf)))
				fmt.Fprint(out, "\r\n")
				return string(buf)
			}
			switch ch {
			case '\r', '\n': // Enter → submit
				draw(strings.Repeat("*", len(buf)))
				fmt.Fprint(out, "\r\n")
				idle.Stop()
				return string(buf)
			case 0x03: // Ctrl-C
				term.Restore(fd, oldState)
				fmt.Fprint(out, "\r\n")
				os.Exit(130)
			case 0x7f, 0x08: // Backspace / Delete
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
				}
			default:
				if ch >= 0x20 { // ignore other control runes
					buf = append(buf, ch)
				}
			}
			draw(string(buf)) // cleartext while active
			idle.Reset(secretRevealDelay)
		case <-idle.C:
			draw(strings.Repeat("*", len(buf))) // mask after idle
		}
	}
}
```

- [ ] **Step 2: Build to verify the new file compiles**

Run: `go build -o bb .`
Expected: compiles (the function is unused until Step 3 — Go allows unused package-level functions, so this builds).

- [ ] **Step 3: Wire `promptPassword` to use the reader on a TTY**

In `cmd/setup.go`, replace the entire `promptPassword` function (currently lines 98–126) with:

```go
// promptPassword prompts for a secret. On a terminal it reveals input while
// typing and masks it after secretRevealDelay (see readSecretRevealing);
// otherwise it falls back to a plain line read for pipes/CI.
func promptPassword(label, current string) string {
	prompt := label + ": "
	if current != "" {
		prompt = label + " [****]: "
	}

	if term.IsTerminal(int(os.Stdin.Fd())) {
		val := readSecretRevealing(os.Stdin, prompt)
		if val == "" {
			return current
		}
		return val
	}

	// Non-terminal fallback (pipes, CI).
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	input, err := r.ReadString('\n')
	if err != nil {
		return current
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return current
	}
	return input
}
```

Note: `cmd/setup.go` already imports `bufio`, `fmt`, `os`, `strings`, and `golang.org/x/term`, so no import changes are needed there. The `time` import lives only in `cmd/secretinput.go`.

- [ ] **Step 4: Build and vet**

Run: `go build -o bb . && go vet ./cmd/...`
Expected: compiles; no vet warnings.

- [ ] **Step 5: Manual verification (interactive TTY)**

Run against a throwaway config so the real one is untouched:
`go run . --config ./tmp-secret.yaml setup`

Confirm, at the **API token** prompt:
- Typing shows cleartext; ~2s after you stop, the field collapses to `*` of the same length.
- Pasting a token shows the full value, then masks after ~2s.
- Backspace removes the last character and the line redraws correctly (no leftover glyphs).
- Pressing Enter accepts the value; the rest of setup proceeds. Inspect `./tmp-secret.yaml` to confirm the value was captured (e.g. `auth_type: apitoken` written when a token was entered).

Then verify the non-TTY fallback still works:
`printf 'ws\n\nme@example.com\nMYTOKEN\n' | go run . --config ./tmp-secret.yaml setup`
Expected: completes without raw-mode errors; `MYTOKEN` is captured.

Clean up: `rm -f ./tmp-secret.yaml ./bb`

- [ ] **Step 6: Commit**

```bash
git add cmd/secretinput.go cmd/setup.go
git commit -m "feat(cli): reveal-then-mask secret prompts on a terminal

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: TUI reveal-on-key + mask-on-idle/blur

**Files:**
- Modify: `cmd/tui/setup.go`

(`cmd/tui` is not unit-tested per repo convention — verify with build, vet, and a manual run.)

- [ ] **Step 1: Add the `time` import**

In `cmd/tui/setup.go`, add `"time"` to the standard-library import group. The import block becomes:

```go
import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)
```

- [ ] **Step 2: Add the delay constant, the reveal generation field, and the mask message**

In `cmd/tui/setup.go`, add a package-level constant next to the other `const` blocks (after the `setupTextFieldCount`/`setupLabelWidth` const block):

```go
// secretRevealDelay is how long the token field stays in cleartext after the
// last keystroke before re-masking. Keep in sync with the CLI's
// secretRevealDelay in cmd/secretinput.go.
const secretRevealDelay = 2 * time.Second

// maskPasswordMsg is delivered by a tea.Tick to re-mask the token field; gen
// lets stale ticks (superseded by a newer keystroke) be ignored.
type maskPasswordMsg struct{ gen int }
```

Add a `revealGen` field to the `setupModel` struct (after the `message` field):

```go
	revealGen     int
```

- [ ] **Step 3: Handle `maskPasswordMsg` in `Update`**

In `cmd/tui/setup.go`, inside `Update`'s top-level `switch msg := msg.(type)`, add a new case alongside the existing `saveResultMsg` case:

```go
	case maskPasswordMsg:
		if msg.gen == m.revealGen {
			m.fields[setupFieldPassword].EchoMode = textinput.EchoPassword
		}
		return m, nil
```

- [ ] **Step 4: Reveal on keystroke in the focused-field forward block**

In `cmd/tui/setup.go`, the bottom of `Update` currently forwards to the focused field:

```go
	// Forward to focused text field only.
	if m.focus < setupTextFieldCount {
		var cmd tea.Cmd
		m.fields[m.focus], cmd = m.fields[m.focus].Update(msg)
		return m, cmd
	}
	return m, nil
```

Replace that block with:

```go
	// Forward to focused text field only.
	if m.focus < setupTextFieldCount {
		var cmd tea.Cmd
		m.fields[m.focus], cmd = m.fields[m.focus].Update(msg)
		// Reveal the token field while it is being edited; a tea.Tick re-masks
		// it secretRevealDelay after the last keystroke.
		if m.focus == setupFieldPassword {
			if _, ok := msg.(tea.KeyMsg); ok {
				m.fields[setupFieldPassword].EchoMode = textinput.EchoNormal
				m.revealGen++
				gen := m.revealGen
				return m, tea.Batch(cmd, tea.Tick(secretRevealDelay, func(time.Time) tea.Msg {
					return maskPasswordMsg{gen: gen}
				}))
			}
		}
		return m, cmd
	}
	return m, nil
```

Note: navigation keys (Tab/Shift+Tab/Up/Down/Enter/Esc/Left/Right) are intercepted earlier in the `tea.KeyMsg` switch and never reach this block, so they do not trigger a reveal. Only content keys (runes, backspace) routed to the field do.

- [ ] **Step 5: Mask on blur in `nextField` and `prevField`**

In `cmd/tui/setup.go`, `nextField` currently starts:

```go
func (m *setupModel) nextField() tea.Cmd {
	if m.focus < setupTextFieldCount {
		m.fields[m.focus].Blur()
	}
```

Replace the blur block in BOTH `nextField` and `prevField` with one that masks the token field when focus leaves it:

```go
	if m.focus < setupTextFieldCount {
		if m.focus == setupFieldPassword {
			m.fields[setupFieldPassword].EchoMode = textinput.EchoPassword
			m.revealGen++ // invalidate any pending reveal tick
		}
		m.fields[m.focus].Blur()
	}
```

(Apply the identical change to `prevField`'s opening blur block.)

- [ ] **Step 6: Build and vet**

Run: `go build -o bb . && go vet ./cmd/...`
Expected: compiles; no vet warnings.

- [ ] **Step 7: Manual verification (TUI)**

Launch the TUI against a throwaway config (no subcommand → TUI; opens setup when unconfigured):
`go run . --config ./tmp-secret-tui.yaml`

At the **API token** field:
- Typing shows cleartext; ~2s after you stop, it masks to `*`.
- Pasting shows the full value then masks after ~2s.
- Pressing Tab/Shift+Tab to leave the field masks it immediately.
- Returning to the field and typing reveals again.
- Save and confirm the captured token value is correct (the wizard builds a working client / writes `auth_type: apitoken`).

Clean up: `rm -f ./tmp-secret-tui.yaml ./bb`

- [ ] **Step 8: Commit**

```bash
git add cmd/tui/setup.go
git commit -m "feat(tui): reveal-then-mask the token field in setup

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Final verification

- [ ] **Full suite, vet, build**

Run: `go test ./... && go vet ./... && go build -o bb .`
Expected: all packages PASS (including the new `cmd/render` SecretLine tests and the unchanged `--describe` manifest snapshot); no vet warnings; binary builds. Then `rm -f ./bb`.

- [ ] **Cross-surface consistency check**

Confirm both `secretRevealDelay` constants (`cmd/secretinput.go` and `cmd/tui/setup.go`) are `2 * time.Second`.

---

## Self-Review (completed during planning)

- **Spec coverage:** Behavior spec (reveal-all/mask-on-idle, 2s, all `promptPassword` callers + TUI token field) → Tasks 2 & 3. §1 TUI (EchoMode toggle + tea.Tick + gen + mask-on-blur) → Task 3. §2 CLI (raw-mode reader, MakeRaw-fail fallback, non-TTY fallback preserved, Ctrl-C/backspace/Enter, ANSI-free redraw) → Task 2. §3 shared pure `render.SecretLine` → Task 1. §4 tests (SecretLine unit-tested; imperative paths manual) → Tasks 1 + manual steps. Out-of-scope items (no toggle, no doc-sync) respected.
- **Placeholder scan:** every code step has complete code; no TBD/TODO; manual-verification steps list concrete observations.
- **Type/name consistency:** `render.SecretLine(prompt, shown string, prevCells int) (string, int)` defined in Task 1 and called identically in Task 2 (`draw` closure) — note the closure is named `draw`, not `render`, to avoid shadowing the imported `render` package. `secretRevealDelay` is the same name in both `cmd` and `cmd/tui` (different packages, no collision). `maskPasswordMsg{gen}` defined and matched consistently in Task 3. `setupFieldPassword` is the existing field index constant.
- **Gotcha noted:** the CLI `draw` closure must not be named `render` (would shadow the package import). Explicitly named `draw` in Task 2 Step 1.
