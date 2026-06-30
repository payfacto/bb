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

- [specs/2026-06-30-search-namespace-design.md](specs/2026-06-30-search-namespace-design.md) - Design for the `bb search` namespace: native `code` search plus BBQL-backed `repos` and repo-scoped `prs`.
- [specs/2026-06-15-gcf-output-format-design.md](specs/2026-06-15-gcf-output-format-design.md) — Design for GCF output format: add `gcf`, make it the default, persist preferred format.
- [specs/2026-05-29-api-token-auth-design.md](specs/2026-05-29-api-token-auth-design.md) — Design for Bitbucket API token authentication.
- [specs/2026-05-29-secret-input-reveal-design.md](specs/2026-05-29-secret-input-reveal-design.md) — Design for timed reveal-then-mask secret input.

### `plans/`

- [plans/2026-06-15-gcf-output-format.md](plans/2026-06-15-gcf-output-format.md) — TDD plan (9 tasks) for the GCF output format feature.
- [plans/2026-05-29-api-token-auth.md](plans/2026-05-29-api-token-auth.md) — TDD implementation plan for API token auth.
- [plans/2026-05-29-secret-input-reveal.md](plans/2026-05-29-secret-input-reveal.md) — TDD implementation plan for reveal-then-mask secret input.

### `reference/`

- _(empty — add vendor/API references and external doc snapshots, one subfolder per topic)_

### `tools/`

- _(empty — add diagnostic helpers, gated on env vars; annotate each with how to invoke)_
