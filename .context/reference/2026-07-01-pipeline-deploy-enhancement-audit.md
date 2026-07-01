# bb Enhancement Audit and Roadmap - Pipelines, Deploy, PR UX

Date: 2026-07-01
Source: two parallel research passes (Bitbucket REST API gap audit + Claude session-log
mining across ~1076 transcripts, 94 with bb usage). Raw reports were generated in the
session scratchpad; this file is the durable consolidation.

Purpose: capture every identified bb enhancement opportunity so nothing is lost, mark
which are being built first, and record the API endpoints each maps to.

---

## Status legend

- [SLICE 1] = committed to the first build slice (this cycle)
- [BACKLOG] = validated opportunity, not yet scheduled
- [DONE] = already shipped

---

## Root cause of the "no UUIDs / can't do some pipeline-deploy commands" complaint

Three compounding issues, all confirmed by both the API audit and real session transcripts:

1. No build-number addressing. `pipeline get/stop/steps/log` require a braced `{uuid}`.
   Agents see `build_number` (e.g. `#123`) in `pipeline list` output and naturally try to
   use it; it fails. Bitbucket exposes `GET pipelines/{build_number}` (plain integer).
2. `bb repo get` never surfaces the repository `uuid` - the `Repo` struct in
   `pkg/bitbucket/types.go` has no `UUID` field, so the API value is dropped. Blocks
   OIDC / Terraform trust configs. No CLI workaround exists today.
3. Missing pipeline/deploy actions: resume a `trigger: manual` gate step, rich pipeline
   trigger (custom pipelines, per-run variables, tag/commit, deployment-env target),
   environment CRUD and env-var management, `deployment get`, `pipeline-var update`.

Doc-drift bug: `CLAUDE.md` command hierarchy lists `env get`, but it is not implemented.

---

## First slice (this cycle)

### [SLICE 1] #1 - Build-number addressing for pipeline get/stop/steps/log  (effort S)

- Gap: every pipeline command after `list` requires the opaque UUID.
- Endpoint: `GET repositories/{ws}/{repo}/pipelines/{build_number}` (returns the full
  Pipeline object, same shape as the UUID endpoint). VERIFY the integer path form against
  the live API before relying on it.
- Shape: add `--build-number` / `-n INT` to `pipeline get`, `stop`, `steps`, `log`.
  Client gains `GetByBuildNumber(ctx, int) (Pipeline, error)`; stop/steps/log resolve the
  build number to a UUID first, then chain. Exactly one of `--pipeline-uuid` /
  `--build-number` required (error if both).

### [SLICE 1] #2 - Expose repository UUID in `bb repo get`  (effort S)

- Gap: `Repo` struct omits `uuid`; blocking for OIDC/Terraform, no workaround.
- Fix: add `UUID string \`json:"uuid"\`` to the `Repo` struct (and render it in the text
  view). Bitbucket already returns it. Check `Workspace` output surfaces its UUID too
  (the struct has the field; confirm `workspace list` is not otherwise broken/deprecated).

### [SLICE 1] #3 - `bb pipeline watch <build-or-branch>` composite  (effort M)

- Highest-frequency friction: the `list -> steps -> log` UUID-chained loop was run 6+ times
  across two projects, every pipeline session.
- Behavior: resolve build number (or latest for a branch) to UUID internally; poll step
  statuses on an interval; with `--tail-log`, stream the active step's log; exit 0 on
  pipeline success, non-zero on failure (enables `bb pipeline watch 5 && deploy`).
- Depends on #1 (build-number resolution).

### [SLICE 1] #4 - `bb pipeline step trigger` (resume a manual gate step)  (effort M)

- Gap: `trigger: manual` deploy steps cannot be resumed from the CLI; forces the Bitbucket
  UI. Confirmed dead end in two projects.
- Endpoint (VERIFY exact path): `POST repositories/{ws}/{repo}/pipelines/{uuid}/steps/{step_uuid}/triggerExecution`.
- Shape: `bb pipeline step trigger --pipeline-uuid UUID --step-uuid UUID` (and, once #1
  lands, `--build-number`).

---

## Backlog (do not forget)

### [BACKLOG] #5 - Rich `pipeline trigger`  (effort M)

- Gap: trigger is branch-only; cannot run custom/named pipelines, pass per-run variables,
  target a tag/commit, or target a deployment environment.
- Endpoint: `POST pipelines/` with richer body (`selector.type=custom` + `pattern`;
  `variables[]`; `ref_type=tag`/commit hash; `target.deployment_environment.uuid`).
- Shape: `bb pipeline trigger [--branch|--tag|--commit] [--custom NAME] [--var K=V ...] [--env-uuid UUID]`.
  `--custom` and the ref-mode flags are mutually exclusive as appropriate.

### [BACKLOG] #6 - `bb pr update <id> --title/--description`  (effort S)

- Gap: no PR edit command; agents decline+recreate PRs to change a description.
- Endpoint: `PUT repositories/{ws}/{repo}/pullrequests/{id}` (title, description,
  destination branch, reviewers). Support stdin JSON via the existing `stdinInputOr` pattern.

### [BACKLOG] #7 - `bb pr create` auto-detect `--repo` and `--source-branch`  (effort S)

- Gap: agents look up these flags every session and get burned by omitting `--repo`.
- Fix: infer `--repo` from the git `origin` remote when unset; default `--source-branch`
  to the current branch (`git rev-parse --abbrev-ref HEAD`). Print a note when inferring;
  no breaking change.

### [BACKLOG] #8 - Environment CRUD + env-var management (+ fix `env get` doc drift)  (effort M)

- Gap: environments are list-only; environment-scoped variables are unreachable.
- Endpoints: `GET/POST environments/`, `GET/PUT/DELETE environments/{uuid}`;
  `GET/POST deployments_config/environments/{uuid}/variables`,
  `GET/PUT/DELETE deployments_config/environments/{uuid}/variables/{var_uuid}`.
- Shape: `bb env get/create/update/delete`, `bb env-var list/create/update/delete --env-uuid UUID`.
  Fixes the CLAUDE.md `env get` doc-drift bug.

### [BACKLOG] #9 - Round out coverage  (effort S each)

- `bb deployment get --uuid UUID` (`GET deployments/{uuid}`).
- `bb deployment list --env-uuid UUID` (BBQL `q=environment.uuid="..."`) + sort.
- `bb pipeline-var update --uuid UUID --value NEW` (`PUT pipelines_config/variables/{uuid}`);
  also `pipeline-var get`.
- Resolve deployment/environment UUID references to names in output (deployment output
  currently shows env UUID only).

### [BACKLOG] #10 - Reliability: `bb pr list` intermittent null returns  (investigation)

- `bb pr list` silently returned null in multiple sessions (documented in the bb repo
  HANDOFF.md). Possibly RTK proxy interference or GCF/pagination interaction. Investigate;
  ensure a real empty result is distinguishable from a failure (exit code + stderr).

### [BACKLOG] #11 - Lower-demand ops surface  (effort L / lower priority)

- Pipeline schedules CRUD (`pipelines_config/schedules/`).
- Pipeline enable/disable (`GET/PUT pipelines_config`).
- Pipeline test reports (`.../steps/{uuid}/test_reports`).
- Commit/build statuses (`commit/{node}/statuses`).
- `bb pr open` (open a PR in the browser via the existing `pkg/browser` dependency).

---

## Already shipped

### [DONE] Code search - shipped in v0.9.0

The "no code-search command" finding surfaced from a stale HANDOFF note; `bb search code`
(plus `search repos` / `search prs`) shipped in v0.9.0.

---

## Endpoints to verify against the live API before implementing

1. `GET pipelines/{build_number}` accepting a plain integer in the UUID path position (#1, #3).
2. The exact manual-step-trigger path, expected `.../pipelines/{uuid}/steps/{step_uuid}/triggerExecution` (#4).

Both are near-certain from the API audit but the Atlassian docs truncated during fetch, so
confirm with a real call (or the OpenAPI spec) during design/TDD.

---

## Full capability gap table (reference)

| Capability | bb today | Bitbucket endpoint | Slice |
|---|---|---|---|
| pipeline list / get-by-uuid / trigger(branch) / stop / steps / log | yes | pipelines/* | - |
| pipeline get by build number | no | GET pipelines/{build_number} | 1 (#1) |
| pipeline watch (composite) | no | (client-side over steps/log) | 1 (#3) |
| pipeline step trigger (manual gate) | no | POST .../steps/{uuid}/triggerExecution | 1 (#4) |
| repo get exposes uuid | no | GET repositories/{ws}/{repo} (uuid field) | 1 (#2) |
| pipeline trigger: tag/commit/custom/vars/env | no | POST pipelines/ (richer body) | backlog (#5) |
| pipeline step get (single) | no | GET .../steps/{step_uuid} | backlog |
| pipeline test reports | no | GET .../steps/{uuid}/test_reports | backlog (#11) |
| pipeline-var get / update | no | GET/PUT pipelines_config/variables/{uuid} | backlog (#9) |
| pipeline enable/disable | no | GET/PUT pipelines_config | backlog (#11) |
| pipeline schedules | no | pipelines_config/schedules/* | backlog (#11) |
| deployment get / list-by-env | no | GET deployments/{uuid}; deployments/?q= | backlog (#9) |
| env get | no (doc drift) | GET environments/{uuid} | backlog (#8) |
| env create / update / delete | no | POST/PUT/DELETE environments/* | backlog (#8) |
| env-var list/create/update/delete | no | deployments_config/environments/{uuid}/variables* | backlog (#8) |
| pr update (edit) | no | PUT pullrequests/{id} | backlog (#6) |
| pr create auto-detect repo/branch | no | (client-side git inference) | backlog (#7) |
| commit/build statuses | no | commit/{node}/statuses* | backlog (#11) |
