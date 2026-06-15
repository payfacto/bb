# Design: API token authentication for `bb`

**Date:** 2026-05-29
**Status:** Approved (pending spec review)
**Author:** brainstorming session

## Background

Atlassian is deprecating Bitbucket Cloud **app passwords** in favour of **API
tokens**. Timeline:

- **2025-06-09** — announcement; app passwords keep working.
- **2025-09-09** — no new app passwords can be created.
- **2026-06-09** — app passwords **stop working entirely**.

Today is **2026-05-29**, so the hard cutover is **11 days away**. `bb` currently
authenticates with app passwords (HTTP Basic auth) or OAuth 2.0 (Bearer). After
2026-06-09 the app-password path breaks for every user.

Reference:
- <https://www.atlassian.com/blog/bitbucket/bitbucket-cloud-transitions-to-api-tokens-enhancing-security-with-app-password-deprecation>
- <https://support.atlassian.com/bitbucket-cloud/docs/create-an-api-token/>
- <https://support.atlassian.com/bitbucket-cloud/docs/using-api-tokens/>

## Key technical fact

Bitbucket Cloud API tokens authenticate via **HTTP Basic auth** — mechanically
identical to app passwords. The only differences from app passwords:

- The **password** is an API token (created *with scopes*), not an app password.
- The Basic-auth **username** field must be the user's **Atlassian account
  email** (e.g. `jmadore@payfacto.com`), *not* the Bitbucket username.

In `bb`, `Config.Username` is consumed in exactly three places: the Basic-auth
identity (`req.SetBasicAuth(c.username, c.token)` in
`pkg/bitbucket/client.go`), the OS-keyring key (`auth.GetToken/SetToken/
DeleteToken`), and display in `bb auth status`. It is **never** compared against
a Bitbucket username for "my PRs"/approve-style logic, and is never used to
build API URLs. Therefore the email can occupy the existing `Username` field
with no new field and no semantic conflict. (`bb setup` already labels the field
"Username (email)".)

## Decisions (from brainstorming)

1. **Scope:** full first-class API token support (new `auth_type`, default
   setup flow, deprecation messaging, docs) — not a minimal patch.
2. **Identity model:** store the **Atlassian account email** in the Basic-auth
   username field (the documented primary approach), not the static
   `x-bitbucket-api-token-auth` literal.
3. **Entry point:** `bb setup` defaults to the API-token flow. Non-interactive
   use is served by the existing env vars / flags (no new flags).
4. **Deprecation nudging:** warn in `bb auth status` and `bb setup` only — no
   per-command stderr warnings.

## Chosen approach

**Treat API token as a labeled Basic-auth variant.** Add a third `auth_type`
value (`apitoken`); reuse `Username` for the email; leave the HTTP client
completely untouched (it already produces `Basic base64(email:token)` for any
non-OAuth config). The work is config semantics, setup UX (CLI + TUI),
deprecation messaging, tests, and docs.

Rejected alternatives:
- **Explicit `email` config field + identity resolver** — adds a field,
  migration logic, and client branching for no functional gain, since nothing
  needs the Bitbucket username separately. More risk before the deadline.
- **Programmatic token minting** — not possible; API tokens are created by hand
  in the Atlassian account UI and shown exactly once.

## Detailed design

### 1. Config semantics (`internal/config`)

- `auth_type` now takes one of: `""`/`apppassword` (legacy), `apitoken`,
  `oauth`.
- Add helper:
  ```go
  // IsLegacyAppPassword reports whether the config is using the deprecated
  // app-password auth (no explicit auth_type, or "apppassword") while a token
  // is present. Used to surface the 2026-06-09 deprecation notice.
  func (cfg *Config) IsLegacyAppPassword() bool {
      if cfg.HasOAuth() || cfg.AuthType == "apitoken" {
          return false
      }
      return cfg.Token != ""
  }
  ```
- `HasOAuth()` stays as-is. The client decision remains binary: OAuth → Bearer,
  everything else → Basic.
- No change to `Load`, `Apply`, `Save`, or precedence rules.

### 2. HTTP client (`pkg/bitbucket`) — no change

`setAuth` already sends Bearer when a bearer token is set and Basic otherwise.
API tokens use Basic auth with `username=email`, which the current code produces
verbatim. The client source is untouched; only a test assertion is added
(section 7).

### 3. `bb setup` CLI (`cmd/setup.go`) — default to API token

- Prompt order: Workspace → Default repo (optional) → **Atlassian account
  email** → **API token**.
- Relabel prompts: identity prompt → `Atlassian account email`; secret prompt →
  `API token`.
- Before the email prompt, print a one-line pointer:
  `Create an API token (with scopes) at https://id.atlassian.com/manage-profile/security/api-tokens`
- Set `AuthType = "apitoken"` whenever a token is entered (replacing the current
  `"apppassword"` assignment).
- Token stored via `auth.SetToken(email, token)` (existing keyring path);
  success message updated to "API token stored in OS keyring."
- The app-password entry path is **removed** from setup — there is no reason to
  capture a credential that stops working in 11 days. Existing stored app
  passwords keep working until 2026-06-09 through the unchanged Basic-auth flow;
  users migrate by re-running `bb setup` with an email + API token.

### 4. TUI setup (`cmd/tui/setup.go`)

Mirror the CLI changes so the no-subcommand TUI stays consistent:
- Identity field label/placeholder → `Atlassian account email`.
- Secret field label → `API token`.
- Write `AuthType = "apitoken"` on save.
- Update the review/summary labels (currently `"Username"`, `"App password"`).

### 5. Deprecation messaging (`cmd/auth.go` status + setup)

- `bb auth status` renders the auth method as:
  - `apitoken` → `API token`
  - `oauth` → `OAuth 2.0`
  - legacy (`IsLegacyAppPassword()`) →
    `app password (DEPRECATED — stops working 2026-06-09; run 'bb setup' to switch to an API token)`
- No per-command stderr warnings are added.

### 6. Non-interactive / CI path — documented, no code change

`BITBUCKET_USER=<atlassian-email>` + `BITBUCKET_TOKEN=<api-token>` (or the
`--username` / `--token` flags) already yield correct Basic auth. This is the
agent/CI path and is documented in README/llms.txt. No new flags or env vars.

### 7. Tests

- `internal/config`: table-driven test for `IsLegacyAppPassword()` across
  `auth_type` values `""`, `apppassword`, `apitoken`, `oauth`, each with and
  without a token.
- `pkg/bitbucket` (`client_auth_test.go`): add a case asserting that a config
  with `auth_type: apitoken`, an email username, and a token produces
  `Authorization: Basic base64(email:token)` and **no** Bearer header.
- `--describe` manifest: unaffected. No leaf commands are added or renamed, so
  `commandRegistry`, `typeRegistry`, and `cmd/testdata/manifest.golden.json` do
  not change. (`TestEveryLeafIsRegistered` / `TestManifestSnapshot` stay green.)

### 8. Documentation sync (required by CLAUDE.md)

Update in the same change:
- **README.md** — authentication section (two methods → three; app password
  marked deprecated with the 2026-06-09 date) and the `bb setup` walkthrough;
  document the env-var path for CI.
- **llms.txt** — the auth method reference line.
- **CLAUDE.md** — no command-hierarchy change, but update any narrative that
  describes the auth methods if present.

## Out of scope (YAGNI)

- `bb auth migrate` guided helper.
- A dedicated `email` config field.
- Per-command deprecation warnings.
- Any change to the HTTP client's auth logic.
- An `bb auth login --api-token` subcommand (setup is the single entry point;
  OAuth keeps its existing `bb auth login`).

## Migration story for users

1. Create an API token (with scopes) in Atlassian account settings.
2. Run `bb setup`; enter Atlassian email + API token.
3. `bb` writes `auth_type: apitoken` and stores the token in the OS keyring.
4. CI/agents set `BITBUCKET_USER=<email>` and `BITBUCKET_TOKEN=<api-token>`.

Existing OAuth users are unaffected. Existing app-password users keep working
until 2026-06-09 and see a deprecation notice in `bb auth status`.
