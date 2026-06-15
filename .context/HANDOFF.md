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
