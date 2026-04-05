# TUI Mode — Design Spec

**Date:** 2026-04-05  
**Scope:** Interactive terminal UI for browsing and acting on Bitbucket Cloud resources  
**Status:** Draft

---

## Context

`bb` currently operates as a traditional CLI — you type a command, get output, type another command. For human users who want to explore PRs, check pipelines, or review activity, this requires knowing the command tree and copy-pasting IDs between commands. A TUI mode provides an interactive, navigable interface where users can browse, drill down, and take actions without leaving the terminal.

The existing CLI commands and `--format json` output remain unchanged. The TUI is an additional mode, not a replacement.

---

## Goals

- Provide an interactive explorer for all Bitbucket resources accessible through `bb`
- Zero learning curve — arrow keys, Enter, Escape. Shortcuts for power users are self-documenting in a bottom bar.
- Reuse existing `pkg/bitbucket` client and `cmd/render` formatting — no duplicate API or rendering logic
- No breaking changes to existing CLI behavior

---

## Launch

Running `bb` with no subcommand enters TUI mode. All existing commands (`bb pr list`, `bb pipeline get`, etc.) continue to work exactly as before.

```go
// In cmd/root.go, set RunE on the root command:
// Cobra calls this only when no subcommand matches.
rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
    return tui.Run(client, cfg)
}
```

If stdout is not a TTY (piped), the root command prints usage help instead of launching the TUI — this prevents scripts from accidentally entering interactive mode.

---

## New Dependency

```
github.com/charmbracelet/bubbletea   — TUI framework (Elm architecture)
github.com/charmbracelet/bubbles     — Pre-built components (list, spinner, viewport, textinput)
```

Both are from the Charm ecosystem already used by the project (lipgloss, glamour).

---

## Navigation Model

**Drill-down with breadcrumbs.** Each view is pushed onto a navigation stack. Escape pops back. Breadcrumbs at the top always show the current path.

```
Home > Pull Requests > #62 > Comments
```

Three levels of views:

1. **Home menu** — top-level command groups (Pull Requests, Pipelines, Branches, etc.)
2. **List view** — entities in a group (PR list, branch list, pipeline list)
3. **Detail view** — single entity with info block + action menu

From a detail view, selecting an action (e.g., "Comments" on a PR) pushes another list or viewport onto the stack. The depth is unbounded but typically 3-4 levels.

---

## Home Menu

The main menu groups commands logically:

| Menu Item | Maps to | Description |
|-----------|---------|-------------|
| Pull Requests | `pr` | List, review, approve, merge PRs |
| Pipelines | `pipeline` | View builds, trigger, check steps |
| Branches | `branch` | List, create, delete branches |
| Commits | `commit` | Browse commit history |
| Tags | `tag` | List, create, delete tags |
| Issues | `issue` | Track and manage issues |
| Repositories | `repo` | List workspace repos |
| Deployments | `deployment` | View deployments |
| Settings | (group) | Webhooks, deploy keys, environments, restrictions |
| Members | `member` | Workspace members |

"Settings" is a sub-menu grouping less-used infrastructure commands, not a single command.

The header shows the current workspace and repo from config.

---

## List Views

All list views share a common structure:

```
Breadcrumb: Home > Pull Requests
──────────────────────────────────
Filter bar: [OPEN] | MERGED | DECLINED        (Tab to cycle, where applicable)
──────────────────────────────────
  ID    TITLE                          AUTHOR
  ───── ──────────────────────────── ────────
▸ #62   Add @Transactional to up...   Nivetha
  #61   Upgrade to spring boot 3     Nivetha
  #60   HikariCP tuning - token...   Liam
──────────────────────────────────
↑↓ navigate  enter open  tab filter  / search  esc back
```

**Pre-set API filters:** Lists that support server-side filtering show a filter bar at the top. Tab cycles through options and re-fetches from the API.

| List | Filter options |
|------|---------------|
| Pull Requests | OPEN, MERGED, DECLINED, SUPERSEDED |
| Pipelines | (none — API returns all) |
| Commits | Branch selector (prompted on entry) |
| Issues | (none) |

**Type-ahead search:** Pressing `/` opens a text input that filters the currently loaded list client-side. Matches against all visible columns. Escape clears the filter.

**Loading:** A spinner with context message ("Loading pull requests...") shows while the API call is in flight.

**Empty state:** Shows the same friendly messages as CLI text output ("No pull requests found.").

---

## Detail Views

Detail views have two sections:

**Top: Info block** — reuses existing `render.*String()` functions. For PRs, this is the label/value block (ID, Title, State, Author, Branch, URL). For pipelines, the build detail. Rendered in a viewport (scrollable if content is long).

**Bottom: Action menu** — a navigable list of contextual actions. These vary by entity type:

### PR Detail Actions

| Action | Behavior | Shortcut |
|--------|----------|----------|
| Comments | Push comment list | `c` |
| Activity | Push activity timeline (viewport) | |
| Statuses | Push status list | |
| Diff | Push diff viewport | `d` |
| Tasks | Push task list | |
| Approve | Approve PR (immediate, no confirm) | `a` |
| Merge | Merge PR (confirmation dialog) | `m` |
| Decline | Decline PR (confirmation dialog) | |

### Pipeline Detail Actions

| Action | Behavior | Shortcut |
|--------|----------|----------|
| Steps | Push steps list | |
| Log | Push log viewport (select step first) | |
| Stop | Stop pipeline (confirmation dialog) | |

### Issue Detail Actions

| Action | Behavior |
|--------|----------|
| (view only) | Issues are read-only in TUI — use CLI to create/edit |

### Commit Detail Actions

| Action | Behavior |
|--------|----------|
| View file | Push file content viewport (prompted for path) |

### Branch/Tag Actions

| Action | Behavior | Shortcut |
|--------|----------|----------|
| Delete | Delete branch/tag (confirmation dialog) | |

### Settings Sub-Items

Each settings item (Webhooks, Deploy Keys, Environments, Restrictions, Downloads) opens a list view. Detail views for these are read-only — create/delete via CLI.

---

## Confirmation Dialog

Destructive actions (merge, decline, delete) show an inline confirmation:

```
┌─────────────────────────────────────┐
│  Merge PR #62 into main?           │
│                                     │
│  Strategy: merge_commit             │
│                                     │
│  [Y]es    [N]o                      │
└─────────────────────────────────────┘
```

The dialog is a modal overlay. `y` confirms, `n` or Escape cancels. The confirmation message includes relevant context (PR number, target branch, strategy).

Safe actions (approve, trigger pipeline) execute immediately with an inline success message that auto-dismisses after 2 seconds.

---

## Keyboard Shortcuts

### Global (always available)

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Select / open |
| `Esc` | Go back one level |
| `/` | Open type-ahead filter |
| `r` | Refresh current view |
| `q` | Quit TUI |
| `?` | Show help overlay |

### Contextual (shown in bottom bar when available)

| Key | Context | Action |
|-----|---------|--------|
| `Tab` | List with filters | Cycle filter |
| `a` | PR detail | Approve |
| `m` | PR detail | Merge |
| `c` | PR detail | Open comments |
| `d` | PR detail | Open diff |

The bottom bar always shows which shortcuts are available in the current context. No hidden keys.

---

## Architecture

### New package: `cmd/tui/`

```
cmd/tui/
  app.go          — top-level Bubbletea program, navigation stack, breadcrumbs
  nav.go          — navigation stack (push/pop), breadcrumb string builder
  keys.go         — key binding definitions per context
  menu.go         — home menu model
  list.go         — generic list model (parameterized by entity type)
  detail.go       — generic detail model (info block + action menu)
  viewport.go     — scrollable text view (diffs, activity, logs)
  filter.go       — filter bar (pre-set + type-ahead)
  confirm.go      — confirmation dialog modal
  styles.go       — TUI-specific styles (extends cmd/render styles)
  sections.go     — section definitions (maps menu items to fetch/render/action configs)
```

### Generic List Model

The list model is parameterized, not duplicated per entity:

```go
type ListConfig[T any] struct {
    Title      string
    FetchFunc  func(ctx context.Context) ([]T, error)
    RenderFunc func(items []T) string       // reuses render.*String()
    OnSelect   func(item T) tea.Cmd         // what happens when user presses Enter
    Filters    []FilterOption               // pre-set API filters (optional)
    Shortcuts  []KeyBinding                 // contextual shortcuts
}
```

Each section (PRs, Pipelines, etc.) provides a `ListConfig` — the list model handles rendering, navigation, filtering, and loading state generically.

### Generic Detail Model

Same pattern:

```go
type DetailConfig[T any] struct {
    Title      string
    FetchFunc  func(ctx context.Context) (T, error)
    RenderFunc func(item T) string          // reuses render.*DetailString()
    Actions    []Action                     // navigable action menu items
    Shortcuts  []KeyBinding                 // contextual shortcuts (a=approve, etc.)
}
```

### Section Definitions

`sections.go` is the single file that wires everything together — it defines the config for each menu item:

```go
func PRListSection(client *bitbucket.Client, ws, repo string) ListConfig[bitbucket.PR] {
    return ListConfig[bitbucket.PR]{
        Title:      "Pull Requests",
        FetchFunc:  func(ctx context.Context) ([]bitbucket.PR, error) { return client.PRs(ws, repo).List(ctx, "OPEN") },
        RenderFunc: render.PRListString,
        OnSelect:   func(pr bitbucket.PR) tea.Cmd { return pushPRDetail(pr.ID) },
        Filters:    []FilterOption{{Label: "OPEN"}, {Label: "MERGED"}, {Label: "DECLINED"}},
    }
}
```

This keeps the TUI framework code (list.go, detail.go) completely decoupled from Bitbucket-specific logic.

### Connection to Existing Code

- `cmd/root.go` detects no subcommand and calls `tui.Run(client, cfg)`
- TUI calls `pkg/bitbucket` client methods directly — same API layer as CLI
- TUI reuses `cmd/render/*String()` functions for formatting content
- Loading states use `bubbles/spinner`
- Lists use `bubbles/list` (or a thin wrapper)
- Scrollable content uses `bubbles/viewport`
- Text input (search) uses `bubbles/textinput`

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| API error (4xx/5xx) | Inline error at bottom of view in red, `r` to retry |
| Network timeout | Same as API error |
| Empty response | Friendly message ("No pull requests found.") |
| Auth failure (401) | Exit TUI with message: "Authentication expired — run `bb auth login` or `bb setup`" |

Errors never crash the TUI. The user can always Escape back or `r` to retry.

---

## Styling

Reuses existing lipgloss styles from `cmd/render/styles.go`:
- `IDStyle` for entity IDs
- `LabelStyle` for labels
- `BranchStyle` for branch names
- `StateBadge()` for state badges
- `SepStyle` for separator lines
- `DimStyle` for secondary text

TUI-specific additions in `cmd/tui/styles.go`:
- Breadcrumb style
- Selected item highlight (background + left border accent)
- Error message style (red background)
- Success message style (green, auto-dismiss)
- Confirmation dialog border
- Filter bar active/inactive styles
- Bottom shortcut bar style

---

## Out of Scope

- **Creating entities from TUI** — PRs, issues, branches, comments require multi-field forms. Use CLI commands.
- **Inline editing** — no editing descriptions or comments
- **Real-time updates** — no auto-refresh or websockets. Press `r` to refresh.
- **Mouse support** — keyboard only for v1
- **Custom themes** — uses the same Catppuccin-inspired palette as CLI text output
- **Configuration** — no TUI-specific config. Uses same `~/.bbcloud.yaml` as CLI.

---

## Verification

```bash
# Build and launch TUI
go build -o bb .
./bb                          # should enter TUI mode

# Verify CLI still works
./bb pr list -f text          # should work as before
./bb pr list                  # should output JSON as before

# TUI navigation test
# 1. Arrow down to "Pull Requests", Enter
# 2. See PR list with state filter
# 3. Tab to cycle OPEN → MERGED → DECLINED
# 4. Enter on a PR to see detail
# 5. Press 'c' to view comments
# 6. Escape back to detail, Escape to list, Escape to home
# 7. Press 'q' to quit

# Verify non-TTY still works (JSON output for pipes)
./bb pr list | jq .          # should still produce JSON
```
