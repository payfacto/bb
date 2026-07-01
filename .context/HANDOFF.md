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

## Session - 2026-06-30 21:59 (Search shipped + v0.9.0; pipeline UX spec ready)

### Shipped this session
- `bb search` namespace (code/repos/prs) built via brainstorm -> spec -> plan ->
  subagent-driven TDD. Merged (PR #3) with list-pagination (PR #2).
- BBQL input escaping hardening: `bbqlQuote` helper applied to all `q=` clauses
  (search.go, pr.go, repo.go). Commit `0593783`.
- Released **v0.9.0** (tag on `main` fc285ff): GitHub Release + Homebrew tap
  updated, GoReleaser workflow green. All additive, no breaking changes.
- Enhancement audit + roadmap saved to
  `.context/reference/2026-07-01-pipeline-deploy-enhancement-audit.md` (commit
  `2f7a615`) from two research passes (Bitbucket API gap audit + session-log
  friction mining across ~1076 transcripts). This is the durable backlog.

### Current state / where to resume
- On branch **`feat/pipeline-ux`** (HEAD `93bdd2d`), based on `origin/main` = v0.9.0.
- Spec **APPROVED** and committed: `.context/specs/2026-07-01-pipeline-ux-design.md`.
- NEXT STEP: invoke `superpowers:writing-plans` for that spec, then run
  `superpowers:subagent-driven-development` to build it (same pipeline as search),
  then PR against `main`, then (on user sign-off) a v0.10.0 tag release.

### Slice scope (first slice of the roadmap)
- #1 `-n/--build-number` on `pipeline get/stop/steps/log` (exactly one of `-u`/`-n`).
- #2 add `uuid` to the `Repo` struct so `bb repo get` surfaces it.
- #3 `bb pipeline watch` [-n|-u|-b|latest]: poll-to-terminal, ONE final JSON on
  non-TTY / live view on TTY, `--tail-log` to stderr, exit codes
  0 success / 1 failed / 2 blocked-manual-gate / 3 timeout, manual-gate deep-link.
- #4 (resume manual step) DROPPED: no public API (open FR BCLOUD-20050); its value
  folded into watch's blocked/deep-link handling.

### Verify during TDD (flagged in spec)
- Confirm `GET pipelines/{build_number}` accepts the plain integer path form.
- Confirm the Bitbucket pipeline/step state fields that identify a paused manual
  gate; add minimal struct fields for the classifier (keep it a pure, tested fn).

### Backlog (not this slice) - see the audit doc
- #5 rich `pipeline trigger` (--custom/--var/--tag/--commit/--env-uuid)
- #6 `bb pr update`; #7 `pr create` auto-detect --repo/--source-branch
- #8 env CRUD + env-var mgmt (also fixes CLAUDE.md `env get` doc-drift)
- #9 `deployment get`, `deployment list --env`, `pipeline-var update`
- #10 investigate `bb pr list` intermittent null returns (possible RTK interference)
- #11 pipeline schedules / enable-disable / test-reports / commit statuses / `pr open`

### Working conventions / gotchas
- Never push to git/bitbucket without explicit user confirmation (global rule).
- origin is GitHub (`github.com/payfacto/bb`); use `gh`. Releases fire on `v*` tags.
- Repo dating skew: docs written this session are dated 2026-07-01; system clock
  said 2026-06-30. Harmless.
- Local `main` pointer is stale (5497516); `origin/main` is the source of truth
  (fc285ff / v0.9.0). Re-point local main before the next release if needed.
- SDD scratch/ledger lives under `.superpowers/sdd/` (git-ignored).

### Suggested skills next session
- `superpowers:writing-plans` (immediate next step), then
  `superpowers:subagent-driven-development`, then `verify` / release flow.

## Session - 2026-07-01 (Pipeline UX slices #1-#3 built, AFK)

### Shipped this session (all on `feat/pipeline-ux`, NOT pushed)
Built the entire pipeline UX spec (`.context/specs/2026-07-01-pipeline-ux-design.md`)
across all three slices, each via plan -> TDD -> code-review-expert -> clean-code:go.
Commits `00ee04d`..HEAD (14 commits on top of `ded53ba`). Full suite green:
357 tests, `-race` clean, `go vet` clean, gofmt clean.

- **Slice #1 - build-number addressing.** `-n/--build-number` on `pipeline
  get/stop/steps/log` (exactly one of `-u`/`-n`, validated in RunE; dropped
  `MarkFlagRequired` on pipeline-uuid). New client `GetByBuildNumber`. Selector
  logic refactored into a `pipelineSelector` type (clean-code F1 fix). Plan:
  `.context/plans/2026-07-01-pipeline-build-number-addressing.md`.
  **CONFIRMED LIVE:** `bb pipeline get -n 59` returns the pipeline (plain-integer
  path `GET pipelines/59` works).
- **Slice #2 - repo uuid.** Added `Repo.UUID` (`json:"uuid"`); rendered in
  `repo get`. The API already returned it; the struct had dropped it.
- **Slice #3 - `bb pipeline watch`.** `[-n|-u|-b|latest] [--tail-log]
  [--interval 5] [--timeout 0]`. Client `Latest(ctx, branch)` + `Watch(ctx,
  uuid, WatchOptions)` + pure `classifyPipelineState`. Single-JSON contract on
  stdout; progress (TTY-gated) and `--tail-log` (always when requested) to
  stderr. Exit codes 0/1/2/3 via a new package-level `exitCode` honored by
  `cmd.Execute`. Manual-gate detection: pipeline `IN_PROGRESS` + stage
  `PAUSED`/`HALTED` => `blocked` with a `manual_gate` {step, resume URL}.
  Plan: `.context/plans/2026-07-01-pipeline-ux-slices-2-3.md`.
  **VERIFIED LIVE:** `watch -n 59` -> success/exit 0; no-selector -> latest;
  `-n` + `-b` -> validation_failed/exit 1.

### Key decisions / judgement calls
- **Manual-gate state model** taken from Atlassian docs (IN_PROGRESS+PAUSED =
  manual/workflow gate; HALTED = system gate) - both classified `blocked`. The
  classifier is pure + unit-tested, but the exact PAUSED/HALTED strings are NOT
  yet confirmed against a live *paused* pipeline. **Confirm before relying on the
  `blocked` exit code (2) in automation.**
- **`Latest` branch filter is client-side** over the first `-created_on` page
  (25). A branch whose newest pipeline is older than a full page of others could
  be missed. Acceptable for "latest"; revisit if it bites.
- **TTY "live view" simplified** to a per-poll stderr status line (not a
  repainting TUI) - keeps stdout clean + testable.
- **Manifest snapshot strips `example` strings**, so example edits are not
  snapshot-locked (they still improve live `--describe`).
- **Exit codes** use a package-level `exitCode` var read by `Execute` (watch
  prints a normal result + sets a non-zero code; it is not an error path).

### Deferred / follow-ups
- DONE (2026-07-01): Graceful SIGINT for `watch`. `watch` now uses
  `signal.NotifyContext(ctx, os.Interrupt)`; Ctrl-C cancels the poll loop and
  exits 130 (`exitInterrupted`) with a "watch canceled" stderr note, no error
  envelope. Client cancellation locked by `TestPipelines_Watch_ContextCancel`.
- Live confirmation of the manual-gate classifier against a real paused pipeline.
- Backlog #5+ (rich `pipeline trigger`, `pr update`, env CRUD, `deployment get`,
  `pr list` null-return investigation) - see the audit doc, untouched.
- `gcf-go` dependency sign-off still open from the GCF session.

### Where to resume
- Branch `feat/pipeline-ux`, clean tree, all green, nothing pushed. Spec fully
  implemented. NEXT: user review -> PR against `main` -> (on sign-off) `v0.10.0`
  tag (all additive; minor bump). Pre-existing CRLF gofmt noise on `main.go` /
  `cmd/render/markdown.go` is unrelated (do not "fix" - it flips line endings).
