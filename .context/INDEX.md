# bb — Context Index

Navigation hub for the curated `.context/` knowledge base. `CLAUDE.md` `@`-imports
this file every session; the entries below are read on demand. `@`-prefixed files
are also auto-imported by `CLAUDE.md`.

## Root Files

- [HANDOFF.md](HANDOFF.md) — Long-running session log: outstanding backlog,
  condensed session history, decisions, open questions. Read this first to pick up
  where the last session left off.
- [@TECHSTACK.md](TECHSTACK.md) — Tech stack reference (Go 1.25, Cobra, Bubble Tea,
  Charm libs, go-keyring, GoReleaser). Versioned bullets derived from `go.mod` and CI.
- [GO-RELEASE-PATTERNS.md](GO-RELEASE-PATTERNS.md) — Release runbook: Bitbucket→GitHub
  mirror, GoReleaser tag-and-push flow, Homebrew tap automation, troubleshooting.

## Subfolders

### `specs/`

- [specs/2026-07-01-pipeline-ux-design.md](specs/2026-07-01-pipeline-ux-design.md) - Design for the pipeline UX slice: build-number addressing (`-n`), repo UUID in `repo get`, and `bb pipeline watch` (poll-to-terminal, manual-gate detect + deep-link). First slice of the pipeline/deploy roadmap.
- [specs/2026-06-30-search-namespace-design.md](specs/2026-06-30-search-namespace-design.md) - Design for the `bb search` namespace: native `code` search plus BBQL-backed `repos` and repo-scoped `prs`.
- [specs/2026-06-15-gcf-output-format-design.md](specs/2026-06-15-gcf-output-format-design.md) — Design for GCF output format: add `gcf`, make it the default, persist preferred format.
- [specs/2026-05-29-api-token-auth-design.md](specs/2026-05-29-api-token-auth-design.md) — Design for Bitbucket API token authentication.
- [specs/2026-05-29-secret-input-reveal-design.md](specs/2026-05-29-secret-input-reveal-design.md) — Design for timed reveal-then-mask secret input.

### `plans/`

- [plans/2026-07-01-pr-create-autodetect.md](plans/2026-07-01-pr-create-autodetect.md) - TDD plan (3 tasks) for backlog #7: `bb pr create` auto-detect of `--workspace`/`--repo` (git origin, bitbucket.org only) and `--from-branch` (current branch); notes to stderr.
- [plans/2026-07-01-rich-pipeline-trigger.md](plans/2026-07-01-rich-pipeline-trigger.md) - TDD plan (3 tasks) for backlog #5: rich `pipeline trigger` (--tag/--commit, --custom NAME, --var K=V). `--env-uuid` dropped (verified not a real trigger-body field).
- [plans/2026-07-01-pipeline-ux-slices-2-3.md](plans/2026-07-01-pipeline-ux-slices-2-3.md) - TDD plan (5 tasks) for slices #2 (repo uuid in `repo get`) and #3 (`bb pipeline watch`: poll-to-terminal, exit codes, manual-gate detection) of the pipeline UX spec.
- [plans/2026-07-01-pipeline-build-number-addressing.md](plans/2026-07-01-pipeline-build-number-addressing.md) - TDD plan (3 tasks) for slice #1 of the pipeline UX spec: `-n/--build-number` addressing on `pipeline get/stop/steps/log`.
- [plans/2026-06-30-search-namespace.md](plans/2026-06-30-search-namespace.md) - TDD plan (6 tasks) for the `bb search` namespace (code/repos/prs).
- [plans/2026-06-15-gcf-output-format.md](plans/2026-06-15-gcf-output-format.md) — TDD plan (9 tasks) for the GCF output format feature.
- [plans/2026-05-29-api-token-auth.md](plans/2026-05-29-api-token-auth.md) — TDD implementation plan for API token auth.
- [plans/2026-05-29-secret-input-reveal.md](plans/2026-05-29-secret-input-reveal.md) — TDD implementation plan for reveal-then-mask secret input.

### `reference/`

- [reference/2026-07-01-pipeline-deploy-enhancement-audit.md](reference/2026-07-01-pipeline-deploy-enhancement-audit.md) - bb enhancement audit and roadmap (pipeline/deploy API gaps + session-log friction). First slice: build-number addressing, repo UUID, `pipeline watch`, manual step trigger. Backlog: rich trigger, env CRUD/env-vars, pr update/auto-detect, deployment get, and more.

### `tools/`

- _(empty — add diagnostic helpers, gated on env vars; annotate each with how to invoke)_
