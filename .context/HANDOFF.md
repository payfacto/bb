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
- DONE (2026-07-01): Backlog #5 rich `pipeline trigger` - `--tag`/`--commit`
  (+ existing `-b`), `--custom NAME`, repeatable `--var K=V`. Client `Trigger`
  now takes `TriggerOptions`; builds ref/commit target + custom selector +
  variables body. Plan: `.context/plans/2026-07-01-rich-pipeline-trigger.md`.
  **`--env-uuid` deliberately DROPPED**: verified (Atlassian blog, API ref,
  elpy1/bbtrigger, community) that deployment environment is NOT a trigger-body
  field - it is bound to a YAML step, so you target it via `--custom NAME`.
  `--secure-var` omitted (unsecured `--var` only; avoid argv/shell-history
  secret exposure). **Live write NOT tested** (safe validation paths verified:
  no-ref/mutual-exclusion/bad-var all error pre-request; help lists flags).
  Live trigger needs user sign-off (starts a real pipeline).
- Backlog remaining: #6 `pr update`, #7 `pr create` auto-detect, #8 env CRUD,
  #9 `deployment get`/`pipeline-var update`, #10 `pr list` null-return, #11 misc
  ops - see the audit doc, untouched.
- `gcf-go` dependency sign-off still open from the GCF session.

### Where to resume
- Branch `feat/pipeline-ux`, clean tree, all green (374 tests), nothing pushed.
  Pipeline UX spec (#1-#3) + graceful-Ctrl-C + backlog #5 all implemented.
  NEXT: optional live trigger test (user sign-off) -> user review -> PR against
  `main` -> (on sign-off) `v0.10.0` tag (all additive; minor bump). Pre-existing
  CRLF gofmt noise on `main.go` / `cmd/render/markdown.go` is unrelated (do not
  "fix" - it flips line endings).

## Session - 2026-07-01 12:23 (Ctrl-C + backlog #5 shipped; next: backlog #6-#11)

### State at handoff
- Branch `feat/pipeline-ux`, HEAD `37d6e5c`, clean tree, **374 tests green**,
  `-race`/`vet`/`gofmt` clean, **nothing pushed**.
- Shipped since the AFK block above: graceful Ctrl-C for `watch` (exit 130) and
  backlog **#5 rich `pipeline trigger`** (`--tag`/`--commit`/`--custom`/`--var`).
  Both went through plan -> TDD -> code-review-expert -> clean-code:go. Details
  are in the "Deferred / follow-ups" bullets above and the two plan docs.
- Established working pattern this thread (reuse it): per item, `superpowers:
  writing-plans` -> TDD (tests in `pkg/bitbucket/`; pure `cmd` helpers get table
  tests; Cobra wiring untested) -> `code-review-expert` (fix all, judgement) ->
  `clean-code:go` -> docs sync (README + llms.txt + CLAUDE.md) -> commit. Never
  push without asking. Regenerate the manifest golden (`go test ./cmd/ -update`)
  whenever a leaf/flag changes; the snapshot strips schemas + examples.

### Next session: backlog #6-#11 (source of truth: the audit doc)
Read [reference/2026-07-01-pipeline-deploy-enhancement-audit.md](reference/2026-07-01-pipeline-deploy-enhancement-audit.md)
for the per-item endpoints/shapes. Summary + suggested order (small/independent first):
- **#7 `pr create` auto-detect** `--repo`/`--source-branch` from git (effort S).
- **#6 `bb pr update <id> --title/--description`** (effort S; `PUT` on the PR).
- **#9 `deployment get`, `deployment list --env-uuid`, `pipeline-var update`** (M).
- **#8 env CRUD + env-var mgmt** (M) - also fixes the CLAUDE.md `env get` doc-drift
  noted in the audit; larger surface, do after the S items.
- **#10 investigate `bb pr list` intermittent null returns** (investigation) -
  audit flags possible RTK proxy interference; reproduce with RTK disabled first.
- **#11 misc ops** (L): pipeline schedules / enable-disable / test-reports /
  commit statuses / `pr open` - lowest demand, split into sub-slices.

### Gotchas / decisions to carry
- **Write ops need explicit sign-off before any live test** (as with #5's trigger
  and the still-untested live trigger). Unit-assert the request body; do not fire
  real mutations autonomously.
- For any API-shape uncertainty, verify against docs/real tools BEFORE building
  (the #5 `--env-uuid` lesson: the audit can be speculative - `--env-uuid` was
  dropped because deployment env is not a trigger-body field).
- Client method signature changes must update ALL call sites incl. `cmd/tui/`
  (the #5 Trigger change touched `cmd/tui/sections.go`).
- Still open (not #6-#11): optional live `watch`-manual-gate confirmation on a
  real paused pipeline; optional live `trigger` smoke test; `gcf-go` sign-off;
  the eventual PR -> `v0.10.0`.

### Suggested skills next session
- `superpowers:writing-plans`, then TDD, `code-review-expert`, `clean-code:go`.
- `handoff` to append the next block.

## Session - 2026-07-02 11:19 (Backlog #7 + #6 shipped; v0.10.0 review-hardening + clean-code; all on local main)

### Shipped this session (all merged linearly to local `main`, NOT pushed)
Followed the established loop per item (brainstorm -> writing-plans -> subagent-driven
TDD -> per-task review -> final whole-branch review -> merge --ff-only to main).
- **Backlog #7 - `bb pr create` auto-detect.** Infers `--workspace`/`--repo` from the
  git `origin` remote (bitbucket.org only) and `--from-branch` from the current branch
  when unset; config/flags win; notes to stderr. New `internal/git` package
  (`ParseBitbucketRemote` pure + `OriginURL`/`CurrentBranch` exec wrappers, 3s timeout);
  pure resolvers `inferWorkspaceRepo`/`inferFromBranch` in `cmd/pr_infer.go`. Commits
  be853d5, a968e74, 8dd69d7, c67640a. Plan: [plans/2026-07-01-pr-create-autodetect.md](plans/2026-07-01-pr-create-autodetect.md).
- **Backlog #6 - `bb pr update -p ID [-T][-d|--description-file]`** (+ stdin JSON). Client
  `PRResource.Update` fetch-then-merge (GET current -> overlay -> PUT), mirroring
  `AddReviewer`. Commits 7f59334, 12f515c. Plan: [plans/2026-07-02-pr-update.md](plans/2026-07-02-pr-update.md).
- **Review "fix all" hardening wave** (from a /code-review-expert pass over the full
  v0.10.0 delta: 1 P1 + 8 P2 + ~12 P3). Two reviewed fix waves + cleanup:
  b8e5681 (pr+git), 7b2a220 (pipeline), c8e4a04 (em-dash comment), then clean-code:go
  pass 8b2bdf1. Highlights: `pipeline watch --branch` no longer false-404s (`Latest`
  now paginates newest-first to first branch match + `ErrNoPipelines` sentinel mapped
  in cmd/errors.go); `Watch` uses one `time.NewTicker` + a `context.WithTimeout` child
  so `--timeout` bounds in-flight requests, with correct `context.Canceled` (exit 130)
  vs `DeadlineExceeded` (exit 3) split in both `Watch` and `watchErr`;
  `ParseBitbucketRemote` rejects >2 path segments and handles `ssh://host:port/`;
  `UpdatePRInput` now `*string` (nil=keep, non-nil=set incl. "" to clear description);
  `pr update` trims/rejects whitespace-only title; pr-create inference notes print
  only after `workspaceAndRepo()` succeeds; `GetByBuildNumber` guards build<=0;
  pipeline state strings hoisted to consts (`StateInProgress` exported for cmd);
  `pipelineWebURL` uses `url.PathEscape`; trigger success echoes the resolved ref.

### Current state / where to resume
- On `main` @ **8b2bdf1**, clean tree, **418 tests pass**, -race/vet/gofmt clean.
- `main` is **37 commits ahead of `origin/main` (fc285ff = v0.9.0), UNPUSHED.** Everything
  additive -> clean **v0.10.0** minor bump. NEXT (needs user sign-off): release v0.10.0
  (push main + tag `v0.10.0` -> GoReleaser/GitHub Release/Homebrew), OR continue backlog.

### Deferred / open (decided NOT to fix - non-regressions on just-reviewed code)
- `watchCtxResult` (pkg/bitbucket/pipeline.go): an extremely narrow simultaneity window
  where a real API error arriving exactly at the child-ctx deadline is classified as
  timeout (exit 3) instead of surfaced. Reviewer called it defensible/not a regression.
  Optional hardening: only treat the request-error branch as timeout when the returned
  error `errors.Is(context.DeadlineExceeded)`.
- `runningStep` cmd literal now uses `bitbucket.StateInProgress` (fixed). Two nice-to-haves
  left: `prCreateCmd` mutates `cfg.Workspace`/`cfg.Repo` to feed `workspaceAndRepo()`
  (G36, pre-existing pattern across all commands); `classifyPipelineState` 3-return
  signature could be a named struct (idiomatic as-is).
- Still open from earlier sessions: live smoke tests needing sign-off (`pipeline trigger`,
  `watch` manual-gate on a real paused pipeline); `gcf-go` dependency sign-off.

### Remaining backlog (source: [reference/2026-07-01-pipeline-deploy-enhancement-audit.md](reference/2026-07-01-pipeline-deploy-enhancement-audit.md))
- #9 `deployment get`, `deployment list --env-uuid`, `pipeline-var update` (M).
- #8 env CRUD + env-var mgmt (M; also fixes CLAUDE.md `env get` doc-drift).
- #10 investigate `bb pr list` intermittent null returns (reproduce with RTK disabled first).
- #11 misc ops (L): pipeline schedules / enable-disable / test-reports / commit statuses / `pr open`.

### Gotchas / conventions
- Never push/tag without explicit user sign-off (global rule). origin is GitHub
  (github.com/payfacto/bb); releases fire on `v*` tags. Local `main` now leads origin/main.
- Feature branches this session were merged `--ff-only` (linear history) and deleted.
- Start each backlog item on a fresh branch off `main`.
- Manifest golden is schema-stripped: pointer/type changes to a stdin struct may produce
  NO golden diff (correct, not a papered-over failure). Regen with `go test ./cmd/ -update`
  only when a leaf/flag/example actually changes.
- SDD ledger + scratch under `.superpowers/sdd/` (git-ignored); review findings for this
  session are in the session scratchpad.

### Suggested skills next session
- Release: `go-release` runbook ([GO-RELEASE-PATTERNS.md](GO-RELEASE-PATTERNS.md)) if cutting v0.10.0.
- Backlog items: `superpowers:writing-plans` -> `subagent-driven-development` -> `code-review-expert` -> `clean-code:go`.

## Session - 2026-07-02 18:57 (Backlog #9/#8/#10/#11 shipped; v0.10.0 RELEASED)

### Headline
Cleared the remaining audit backlog (#9, #8, #10, #11) and **released v0.10.0**. Pushed
`main` + tag `v0.10.0` to origin (GitHub); release run `28626505058` succeeded. VERIFIED:
GitHub Release https://github.com/payfacto/bb/releases/tag/v0.10.0 published with 6 platform
archives + checksums.txt; Homebrew tap `Formula/bb.rb` bumped to `version "0.10.0"`.
`origin/main` == local `main` == release commit `d441871`; clean tree; nothing pending push.

### Shipped this session (all merged --ff-only to main, then released)
Each item ran the loop: writing-plans -> subagent-driven TDD -> per-task review -> whole-branch
code-review-expert (fix-all) -> clean-code:go -> merge. Plans in `.context/plans/`.
- **#9 round-out-coverage** (plan `2026-07-02-backlog-9-coverage.md`): `deployment get`,
  `deployment list --env-uuid/--sort`, `pipeline-var get`, `pipeline-var update` (stdin).
  Commits c628764..9a5311b.
- **#8 env CRUD + env-var** (plan `2026-07-02-backlog-8-env.md`): `env get/create/delete`,
  new `env-var list/create/update/delete` group (new `cmd/env_var.go` +
  `pkg/bitbucket/environment_variable.go`, reuses `PipelineVariable`). Commits 0074e9a..58f9f24.
- **#10 pr list null fix** (bugfix): `fetchAllPages`/`fetchPagesLimit` started with a nil
  accumulator -> empty result marshaled to JSON `null`. Now `all := []T{}` -> `[]`. Fixes ALL
  paginated list cmds. Commit a52d70d.
- **#11 (scoped)** (plan `2026-07-02-backlog-11-misc.md`): `pr open` (reuses PRs.Get +
  Links.HTML.Href + pkg/browser) + `commit statuses` (new CommitStatus type +
  CommitResource.Statuses). Commits ffdd877..d441871.

### Key decisions / API facts learned (durable)
- **Deployments endpoint silently IGNORES `q=` filters** (live-verified) -> `deployment list
  --env-uuid` filters client-side over the first page (pagelen=25). `sort` only accepts limited
  attributes (`state.name`/`-state.name` work; `created_on`/`last_update_time`/`name` -> HTTP 400).
- **`env update` DEFERRED** (user decision): uses undocumented `POST .../environments/{uuid}/changes/`
  with no authoritative body schema. Not built. API research: `.superpowers/sdd/env-api-research.md`.
- **#11 scoped to pr open + commit statuses** (user decision): test-reports, pipeline enable/disable,
  and schedules DEFERRED (effort L, finicky/undocumented write bodies).
- **JSON is the built-in default** output format (per CLAUDE.md + user); the old GCF-default handoff
  note was stale. No breaking-change release note needed.
- **`gcf-go` v1.2.0 re-audited clean** and shipped: no net/os-exec/syscall/unsafe/init; `os` only in
  its unused `cmd/gcf` CLI; only transitive dep is yaml.v3 (already in tree); checksummed.
- Braced UUIDs auto-escape in Go's HTTP path, but #8 uses explicit `url.PathEscape` (matches
  pipeline.go idiom); client tests assert `r.URL.EscapedPath()` against `%7B...%7D`.
- `context.Background()` is the codebase-wide RunE convention (86 uses, 0 cmd.Context) - a reviewer
  false-positive to that effect was rejected.

### Live smoke tests (all pass; throwaway resources cleaned up)
Reads: pr list (OPEN -> [] not null; MERGED populated), deployment list/get, pipeline-var list/get,
env list/get, env-var list, commit statuses, pr open, + text renderers. Writes: full CRUD cycles for
pipeline-var, env, and env-var. Two confirmed **Bitbucket behaviors (not bb bugs)**: env-var list has
brief create->list eventual-consistency lag; env delete is async (renames env `<name>_<ts>` then
removes within ~30s, returns 204 immediately).

### Deferred / open (backlog now effectively exhausted)
- `env update` (undocumented `/changes/` endpoint).
- #11 leftovers: pipeline test-reports, enable/disable, schedules CRUD.
- **Cosmetic follow-up:** `.goreleaser.yaml` brew `description` has a non-ASCII dash (renders as
  `Bitbucket Cloud CLI <char> manage...` in the generated Formula) - change to a hyphen per the
  ASCII-only rule next time that file is touched.
- From earlier sessions (still optional): live confirmation of the `pipeline watch` manual-gate
  classifier against a real paused pipeline; live `pipeline trigger` smoke.

### State / conventions
- On `main` @ `d441871` = `origin/main`; v0.10.0 tagged and released; clean tree; no background procs.
- origin is GitHub (`github.com/payfacto/bb`), single remote; releases fire on `v*` tag push. The
  GO-RELEASE-PATTERNS.md Bitbucket-mirror section is a generic template, NOT how bb is wired.
- SDD ledger + per-task reports under `.superpowers/sdd/` (git-ignored). 436 tests, -race/vet/gofmt clean.
