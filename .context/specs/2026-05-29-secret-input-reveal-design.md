# Design: timed reveal-then-mask for secret input

**Date:** 2026-05-29
**Status:** Approved (pending spec review)
**Author:** brainstorming session

## Problem

When entering a secret (API token, app password, OAuth consumer secret), `bb`
gives poor or no feedback that input landed:

- **CLI** (`cmd/setup.go` `promptPassword`, used by `bb setup` and
  `bb auth login`): uses `term.ReadPassword`, which echoes **nothing** — you
  cannot tell whether a paste or keystroke registered.
- **TUI** (`cmd/tui/setup.go`): the token field masks each character as `*`
  (bubbles `textinput`, `EchoPassword`). You see `*`s but cannot verify the
  actual pasted/typed value.

The user wants: while typing or pasting, show the real input; a few seconds
after the last keystroke, mask the whole field to `*`.

## Decisions (from brainstorming)

1. **Scope:** both the TUI token field *and* the CLI `promptPassword` prompts.
2. **Reveal style:** reveal the entire field while active; after an idle delay,
   mask the entire field to `*`. (Not "last character only", not paste-only.)
3. **Idle delay:** **2 seconds** after the last keystroke/paste, matching the
   existing `statusClearDelay` in the TUI.
4. **No manual toggle** — timed auto-mask only. Typing a character re-reveals.
5. **Approach:** native idiom per surface (Approach 1): Bubble Tea `tea.Tick`
   for the TUI; a self-contained raw-mode reader for the CLI. Shared,
   unit-tested pure rendering helper.

## Behavior specification (both surfaces)

- On each keystroke or paste into a secret field, the field displays in
  cleartext and the 2-second idle timer is (re)started.
- 2 seconds after the last input, the entire field re-masks to `*`.
- Any subsequent keystroke re-reveals cleartext and restarts the timer.
- Leaving the field (TUI: blur/focus change; CLI: submit) masks immediately.
- Applies to every `promptPassword` caller (so the OAuth consumer-secret prompt
  in `bb auth login` is covered too) and the TUI setup wizard's token field.

## Detailed design

### 1. TUI (`cmd/tui/setup.go`)

The token field already uses `textinput` with `EchoMode = EchoPassword` and
`EchoCharacter = '*'`.

Add to the package/model:

- `const secretRevealDelay = 2 * time.Second` (local to the `tui` package).
- A field on `setupModel`: `revealGen int` — a monotonically increasing
  "generation" counter used to ignore stale mask ticks.
- A message type: `type maskPasswordMsg struct{ gen int }`.

`Update` changes:

- After routing a message to the focused field, if the focused field is the
  password/token field **and** the message is a `tea.KeyMsg`:
  - set `m.fields[setupFieldPassword].EchoMode = textinput.EchoNormal`
  - `m.revealGen++`
  - capture `gen := m.revealGen`
  - return `m, tea.Batch(cmd, tea.Tick(secretRevealDelay, func(time.Time) tea.Msg { return maskPasswordMsg{gen: gen} }))`
    where `cmd` is the textinput's own update command.
- Add a top-level case:
  ```go
  case maskPasswordMsg:
      if msg.gen == m.revealGen {
          m.fields[setupFieldPassword].EchoMode = textinput.EchoPassword
      }
      return m, nil
  ```
  The `gen` guard means only the most recent tick re-masks; earlier ticks
  (superseded by later keystrokes) are ignored.

Mask on blur: in `nextField` and `prevField`, when the focus is leaving the
password field, set `m.fields[setupFieldPassword].EchoMode =
textinput.EchoPassword` and `m.revealGen++` (so any pending tick is moot).

Reveal trigger: reveal on any `tea.KeyMsg` routed to the focused password field.
Revealing on navigation keys is harmless (the timer simply re-masks 2s later);
this keeps the logic simple and avoids enumerating key types.

`time` must be imported in `setup.go` (it is used for the `tea.Tick`/delay).

### 2. CLI (`cmd/setup.go` + new `cmd/secretinput.go`)

`promptPassword` keeps its signature `func(label, current string) string` and
its **non-TTY `bufio` fallback path unchanged**. Only the
`term.IsTerminal(...)` branch changes: instead of `term.ReadPassword`, call a
new helper:

```go
// readSecretRevealing reads a secret from in, echoing it in cleartext while
// the user types and re-masking to '*' after secretRevealDelay of idle time.
// in must be a terminal (caller checks term.IsTerminal). Returns the entered
// secret. On Ctrl-C it restores the terminal and exits with code 130.
func readSecretRevealing(in *os.File, prompt string) string
```

New file `cmd/secretinput.go`:

- `const secretRevealDelay = 2 * time.Second` (local to the `cmd` package).
- Raw mode: `oldState, err := term.MakeRaw(int(in.Fd()))`; if `err != nil`,
  fall back to `term.ReadPassword` behavior (no reveal) so we never hard-fail;
  otherwise `defer term.Restore(int(in.Fd()), oldState)`.
- Input goroutine: a `bufio.NewReader(in)` loop calling `ReadRune()`, sending
  each rune on `runeCh chan rune`; on error it closes/signals via a `doneCh`.
- The prompt is **not** printed separately. It is emitted only through
  `render.SecretLine` (which prepends `\r` + prompt on every redraw). Do one
  initial render with an empty buffer before entering the loop so the prompt
  is visible before the first keystroke.
- Main loop maintains `buf []rune`, `prevCells int`, and an idle
  `*time.Timer` (created stopped, reset to `secretRevealDelay` on each rune):
  ```
  for {
    select {
    case r, ok := <-runeCh:
       if !ok { finalize-masked; return string(buf) }
       switch r {
       case '\r', '\n':            // Enter → submit
           render masked; write "\r\n"; return string(buf)
       case 0x03:                  // Ctrl-C
           term.Restore(...); fmt.Fprint(out, "\r\n"); os.Exit(130)
       case 0x7f, 0x08:            // Backspace / Delete
           if len(buf) > 0 { buf = buf[:len(buf)-1] }
       default:
           if r >= 0x20 { buf = append(buf, r) }  // ignore other control runes
       }
       render cleartext; idle.Reset(secretRevealDelay)
    case <-idle.C:
       render masked
    }
  }
  ```
- Rendering uses the shared pure helper (section 3); the helper's output is
  written to `out` (stderr or stdout — match what `term.ReadPassword`-era code
  wrote prompts to; prompts currently print via `fmt.Printf` to stdout, so use
  stdout for consistency).
- "Masked" rendering shows `strings.Repeat("*", len(buf))`; "cleartext" shows
  `string(buf)`.

Cross-platform note: rendering is **ANSI-free**. It uses only `\r`, spaces, and
`\b` (backspace) so it works on Windows `cmd.exe` without requiring VT
processing. (No `\x1b[K`.)

### 3. Shared pure rendering helper (`cmd/render`)

`cmd/render` already has unit tests (added in the API-token feature). Add a pure
function there so the redraw logic is testable:

```go
// SecretLine builds the terminal redraw string for a single-line secret prompt.
// prevCells is the rune-width of the previously rendered line (prompt+shown);
// SecretLine pads with spaces to erase any leftover characters when the new
// line is shorter, then emits backspaces to leave the cursor at the end of the
// visible text. It returns the string to write and the new cell count to pass
// as prevCells next time. It emits no ANSI escapes.
func SecretLine(prompt, shown string, prevCells int) (out string, cells int)
```

Implementation sketch:
```go
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

The CLI reader calls `render.SecretLine(prompt, shown, prevCells)` for both the
cleartext and masked states (where `shown` is the cleartext or the `*` string),
updating its `prevCells` from the returned value.

### 4. Tests

- `cmd/render/secretline_test.go` (new): table-driven tests for `SecretLine`:
  - growing input (no pad, no backspaces),
  - shrinking input (pad spaces + matching backspaces to erase leftovers),
  - equal length, empty `shown`, empty `prompt`,
  - returned `cells` equals `len([]rune(prompt+shown))`.
- The imperative raw-mode loop (`cmd/secretinput.go`) and the TUI `Update`
  changes are **manually verified** (build + run on Windows; spot-check
  macOS/Linux if available), consistent with the repo convention that `cmd/`
  wiring and `cmd/tui` are not unit-tested. No changes to `pkg/bitbucket`.

Manual verification checklist:
- CLI `bb setup`: type a token → see cleartext → wait 2s → masks to `*`; paste a
  token → see full value → masks after 2s; backspace erases correctly; Enter
  submits the correct value; piping stdin (non-TTY) still works via the
  unchanged fallback.
- TUI setup: same reveal/mask behavior on the token field; tabbing away masks
  immediately; the saved token value is correct.

## Out of scope (YAGNI)

- Manual reveal/hide toggle keybinding.
- "Last character only" reveal mode.
- Configurable delay (hardcoded 2s constant per surface).
- Documentation sync: no command or flag changes, so README/llms.txt/CLAUDE.md
  do not require updates under the repo's doc-sync rule.

## Logistics

These changes touch the same files as the unfinished `feat/api-token-auth`
branch (`cmd/setup.go`, `cmd/tui/setup.go`). Recommended sequencing: finish
`feat/api-token-auth` (merge or PR) first, then implement this on a fresh
branch off updated `main` (e.g. `feat/secret-input-reveal`) so the two features
are independently reviewable. To be confirmed before implementation.
