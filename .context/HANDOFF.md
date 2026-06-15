# bb — Handoff

## Goal

`bb` is a Cobra-based CLI plus Bubble Tea TUI wrapping the Bitbucket Cloud REST API
v2.0. It gives developers and agents a fast, scriptable (JSON-default, `--format text`)
interface to PRs, pipelines, branches, repos, and more — with an interactive TUI when
run with no subcommand.

## Stack

Go 1.25, single static binary (`CGO_ENABLED=0`). Cobra for commands, Bubble Tea +
Bubbles + Lipgloss + Glamour for the TUI, `gopkg.in/yaml.v3` for `~/.bbcloud.yaml`
config, `go-keyring` for OS credential storage, `golang.org/x/term` for masked input.
No app database; remote state is the Bitbucket REST API. Tests use stdlib
`testing`/`httptest`. Released via GoReleaser on `v*` tags (GitHub Actions) to GitHub
Releases + a Homebrew tap. Full detail in [TECHSTACK.md](TECHSTACK.md).

---

## Outstanding backlog

Items carried across sessions. Most-recent-session detail is below.

**Ship and verify**
- **Push `main` to `origin`** — local `main` has run ahead of the remote in prior
  sessions; confirm origin is current. (Carried since 2026-04-16.)

**Code hygiene / DX**
- **Extend `lipgloss/table` to other list views** — Pipelines, Branches, Commits could
  use a table renderer like `prTableRenderer` (e.g. `#`, result badge, branch, duration,
  date). (Carried since 2026-04-16.)
- **Stale design-doc reference** — `CLAUDE.md` cited
  `docs/superpowers/specs/2026-04-04-bb-cli-design.md`, which is not present in the repo.
  Updated the pointer to `.context/specs/` during the 2026-06-15 setup; locate or
  recreate the original design doc if still needed.

**Phase N polish / nice-to-haves**
- **Theme selector Save vs Preview** — the Setup theme selector applies live and reverts
  on Escape; a distinct "Save" step would clarify the UX.
- **Two-column main menu** — the home menu has many items; a `JoinHorizontal` grid would
  use wide terminals better.

---

## Session history — condensed

**Session 2026-04-16 (TUI theming + layout).** Added an 8-palette theme system
(`cmd/tui/themes.go`: Catppuccin default, Tokyo Night, Dracula, Gruvbox, Nord, Rosé Pine,
One Dark, Facto brand) with runtime `applyTheme()` reassigning package-level lipgloss
style vars; added a live-preview theme selector to the Setup page persisting `Theme` to
`~/.bbcloud.yaml`. Added a `TableRenderer` hook to `ListConfig` and a `prTableRenderer`
(colour-coded state, strikethrough for merged/declined), `lipgloss.Place()` empty states,
terminal-width tracking, help-bar overflow truncation, and a side-by-side detail layout
(≥60 cols). Facto light variant uses `#7a5c2e` gold for WCAG AA.

---

## Session — 2026-06-15 (Bootstrap .context/ knowledge convention)

### Purpose

Set up the repo per `claude-context-pattern.md`: give the project a durable, curated,
version-controlled `.context/` knowledge base discoverable by any agent on session start.

### What was done

- Created [INDEX.md](INDEX.md) — the navigation hub, `@`-imported by `CLAUDE.md`.
- Created this curated [HANDOFF.md](HANDOFF.md) as the single committed handoff. The
  old root `HANDOFF.md` was deleted and its `.gitignore` rule removed (user), so there is
  no ephemeral root copy; the `/handoff` skill should write here.
- Relocated specs and plans out of the **gitignored** `docs/superpowers/` into tracked
  `.context/specs/` and `.context/plans/` so they are committed and reviewable; removed
  the now-empty `docs/` tree.
- Added empty `.context/reference/` and `.context/tools/` (each with `.gitkeep`).
- Prepended `@.context/INDEX.md` to `CLAUDE.md` and updated its design-reference pointer
  from the missing `docs/...` path to `.context/specs/`.

### Files changed

- Docs: `.context/INDEX.md` (new), `.context/HANDOFF.md` (new),
  `.context/reference/.gitkeep` (new), `.context/tools/.gitkeep` (new).
- Moved: `docs/superpowers/specs/*` → `.context/specs/`,
  `docs/superpowers/plans/*` → `.context/plans/`.
- Edited: `CLAUDE.md` (added `@`-import header, fixed design-reference path).

### Decisions

- **Relocate (not copy) specs/plans into `.context/`** — `docs/` was gitignored, leaving
  specs/plans uncommitted; relocating tracks a single source of truth, matching the
  pattern's "curated, committed" thesis. (User-confirmed.)
- **Kept `.context/GO-RELEASE-PATTERNS.md` as the release runbook** rather than renaming
  to `RELEASE.md` — avoids churn; INDEX.md labels it as the release runbook.
- **Left the repo's `.claude/` gitignore policy unchanged** — the pattern suggests
  committing `.claude/`, but this repo deliberately ignores it; out of scope for setup.

### Running state

- Branch: `main`, tree dirty (this setup is uncommitted).
- No background processes.

### Inferred next steps

- Commit the `.context/` setup (the user has not yet been asked to commit/push).
- Work through the Outstanding backlog above.

### Suggested skills for next session

- `handoff` — to append the next session block here.
- `clean-code:go` / `techstack-review-summarizer` — when editing Go or refreshing TECHSTACK.md.

## Session — 2026-06-15 08:15 (Ship GCF output format)

### What shipped

Full brainstorm → spec → plan → subagent-driven TDD → review → merge cycle for
the **GCF (Graph Compact Format) output format** feature. All on local `main`
(merge commits `8f02881` feature, `f948634` review fixes), plus a clean-code pass
(`8ffa6fa`). Not pushed.

- New `cmd/output.go`: `validateFormat`, pure `resolveFormatFrom` (precedence +
  non-TTY guard), `resolveFormat`, `renderValue`, `renderError`, `gcfErrorView`.
- `gcf` is now the **default** output format (was `json`) via `gcf-go` v1.2.0.
  Precedence: built-in `gcf` < `~/.bbcloud.yaml` `format:` < `BB_FORMAT` env <
  `--format` flag. Non-TTY guard coerces `text`->`gcf` unless `--format` set.
- Persisted format: `config.Format` field + `bb setup` wizard picker.
- Errors render in the active format (`renderError`); JSON envelope byte-identical.
- Spec: `.context/specs/2026-06-15-gcf-output-format-design.md`; plan:
  `.context/plans/2026-06-15-gcf-output-format.md`. Docs synced (README, llms.txt,
  CLAUDE.md, TECHSTACK.md). 296 tests pass, vet/gofmt clean.

### Decisions

- **Aggressive gcf default with a JSON escape hatch** (`BB_FORMAT=json`) — agents
  get token savings by default; existing automation pins JSON with one env var.
  This is a documented **breaking change** for anyone piping `bb` expecting JSON.
- **Format vocabulary lives in `internal/config`** (`FormatGCF/JSON/Text`,
  `OutputFormats`) — single source of truth shared by `cmd` and `tui` (cmd imports
  tui, so tui can't import cmd; config is the shared dependency). Done in the
  clean-code pass to kill a G5 duplication between `cmd.allowedFormats` and
  `tui.setupFormatNames`.
- **`loadConfig(cmd)` helper in `cmd/root.go`** — Cobra runs only the nearest
  `PersistentPreRunE` (no parent chaining), so commands that override it (e.g.
  `bb user me`) must use `loadConfig` to guarantee `resolveFormat` runs. This was
  a post-merge code-review P1 fix: persisted format / `BB_FORMAT` were silently
  ignored for `bb user me` before it.
- **GCF errors encode a `{code, message, details?}` map view** (`gcfErrorView`),
  not the `*CLIError` struct — avoids a spurious `## details` section on nil
  details and guarantees the unexported `cause` can never leak.

### Open questions / risks

- **Supply-chain sign-off on `gcf-go`** (third-party, outside payfacto org) before
  pushing. Audited safe: no `net`/`os-exec`/`syscall`/`unsafe`, no `init()`,
  checksummed in `go.sum`, used only to encode already-fetched output. A
  deliberate dependency sign-off is still recommended (`skill-vetter`).
- **Known cosmetic follow-up (not fixed):** GCF error ordering puts `details`
  between `code` and `message` (gcf sorts map keys alphabetically). Acceptable;
  noted only.

### Running state

- Branch `main`, ahead of `origin/main` by ~17 commits, **not pushed**.
- Working tree: `.context/.claudeignore` was cleaned up (deduped, made generic;
  dropped a non-generic `architecture.html` entry) and is **untracked/uncommitted**.
  `.context/claude-context-pattern.md` also untracked (user-added).
- No background processes.

### Inferred next steps

- **Push `main`** to origin (awaiting user go-ahead per their no-push-without-asking rule).
- **Sign off on the `gcf-go` dependency** before it goes remote.
- Decide whether to commit the cleaned `.context/.claudeignore`.
- Add a release note for the GCF-default **breaking change** when cutting the next tag.

### Suggested skills for next session

- `skill-vetter` — for the `gcf-go` dependency sign-off.
