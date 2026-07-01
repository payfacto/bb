# Design: `bb search` namespace

Status: approved (brainstorm) - ready for implementation plan
Date: 2026-06-30
Author: brainstormed with Claude

## Summary

`bb` has no way to search across a workspace today. The only "search" in the
codebase is the TUI's in-memory list filter (`cmd/tui/list.go`), which filters
already-loaded items and never hits an API. This spec adds a top-level `search`
command namespace that mirrors `gh search` within the limits of the Bitbucket
Cloud REST API.

## Background and constraints

The framing "grep across all repos" overstates what Bitbucket Cloud offers.
The relevant API facts that shape this design:

- Code search is a real, native endpoint:
  `GET /2.0/workspaces/{workspace}/search/code?search_query=...` with
  `page`/`pagelen` pagination. It returns per-result `file.path`,
  `content_matches` (matched lines with highlighted segments), `path_matches`,
  and `content_match_count`.
- Code search is **token/word indexed, not regex**. It searches the
  **default branch only**, skips large/binary/generated files, is
  workspace-scoped, and requires the workspace to have code search enabled.
  `git grep`-style regex patterns do not translate.
- There is **no workspace-wide PR search** and **no commit-message search**
  API. PR listing is per-repo (`/repositories/{ws}/{repo}/pullrequests`) and
  accepts BBQL `q=` filtering. Repository listing
  (`/repositories/{workspace}`) accepts BBQL `q=` filtering (bb already uses
  this for `--project`).

Because of those constraints, the namespace ships three subcommands:

| Subcommand        | Backing API                                        | Nature                          |
|-------------------|----------------------------------------------------|---------------------------------|
| `search code`     | native `/workspaces/{ws}/search/code`              | true server-side search         |
| `search repos`    | `/repositories/{ws}` with BBQL `name~`/`description~` | approximate (name/description) |
| `search prs`      | `/repositories/{ws}/{repo}/pullrequests` with BBQL `title~`/`description~` | repo-scoped only |

`search commits` and `search issues` are intentionally excluded: Bitbucket has
no commit-message search API, and issues are a per-repo tracker most workspaces
do not enable. They can be revisited if the API grows.

## Command surface

```
bb search                          # parent; prints help, like `bb pr`
bb search code  <query> [flags]    # native workspace-wide code search
bb search repos <query> [flags]    # repo name/description match (BBQL)
bb search prs   <query> [flags]    # repo-scoped PR title/description match (BBQL)
```

Shapes follow `gh search code/repos/prs`. Deliberate divergences, documented in
each command's long `--help`:

- `search prs` is repo-scoped: it resolves a repo via `--repo` or config and
  errors if none is set. gh's equivalent is global; Bitbucket has no global PR
  search. The help text points users at `bb pr list` for richer per-repo
  filtering (state, branch, dates).
- `search code` long help states that it searches the indexed default branch
  only, is token-based rather than regex, and is workspace-wide, so users do
  not expect `git grep` semantics.

### Flags

- All three subcommands: `--limit N` (default `100`; `0` means unbounded), plus
  the global `--workspace` / `--repo` / `--format` flags.
- `search code`: positional `<query>` is passed to `search_query` **verbatim**,
  so any modifier the API supports keeps working (e.g.
  `bb search code 'ext:go parseConfig'`). Convenience flags that fold gh-style
  options into `search_query` modifiers are added only for modifiers verified
  against the live API during implementation (candidates: `--ext`, `--repo`,
  `--project`). Unverified gh flags (e.g. `--language`) are not shipped unless a
  real Bitbucket modifier backs them.
- `search prs`: `<query>` becomes `(title ~ "q" OR description ~ "q")`. No
  re-implementation of `pr list` filters.
- `search repos`: `<query>` becomes `(name ~ "q" OR description ~ "q")`.

## Client layer (`pkg/bitbucket`)

New file `pkg/bitbucket/search.go`:

```go
// SearchResource provides workspace-scoped search operations.
func (c *Client) Search(workspace string) *SearchResource

func (s *SearchResource) Code(ctx context.Context, opts CodeSearchOptions) ([]CodeSearchResult, error)
func (s *SearchResource) Repos(ctx context.Context, term string, limit int) ([]Repo, error)
```

- `Code` builds `GET /workspaces/{ws}/search/code?search_query=...&pagelen=...`
  and decodes into a typed `CodeSearchResult` (file path, repository slug,
  `content_match_count`, and matched line snippets). Exact JSON field names are
  pinned against the live response while writing tests.
- `CodeSearchOptions` carries the raw query plus any verified modifier flags and
  the limit.
- PR text search reuses the existing `PRResource` instead of a parallel path: a
  `Query string` field is added to `PRListOptions`, folded into BBQL alongside
  the existing State/SourceBranch/Since/Until filters. `search prs` calls
  `client.PRs(ws, repo).List(ctx, PRListOptions{Query: term, ...})`.
- Repo text search lives on `SearchResource.Repos` (or reuses the existing
  repositories endpoint via BBQL); it does not duplicate `RepoResource.List`
  beyond constructing the `q=` filter.

### Pagination with a limit

`fetchAllPages` always drains every page. A new sibling helper handles the
capped case:

```go
// fetchPagesLimit pages until `limit` items are collected, or all pages when
// limit <= 0. Reuses the same "next" link following as fetchAllPages.
func fetchPagesLimit[T any](ctx context.Context, c *Client, path string, q url.Values, limit int) ([]T, error)
```

This is the only change to shared pagination logic. `pagelen` is set so a single
page covers a default limit where possible to minimise round-trips.

## Output

- `json` (default) and `gcf` emit the typed structs unchanged - agent- and
  pipe-friendly, consistent with existing commands.
- `text` renderer (in `cmd/render/`) is grep-style:
  - code: `repo/path/to/file.go:42: <matched line>`
  - repos: `slug  -  description`
  - prs: `#42  title  (OPEN)`

## Errors

Errors flow through the existing `APIError -> CLIError` mapping in
`cmd/errors.go`. A workspace without code search enabled (or a missing
resource) maps to a clean `not_found` / `validation_failed` message rather than
a raw HTTP 404. `search prs` with no resolvable repo returns
`validation_failed` with guidance to set `--repo` or config.

## Agent manifest (`--describe`)

- Register `search code`, `search repos`, `search prs` as `read`-class leaves in
  `commandRegistry` (`cmd/manifest_registry.go`).
- Add `CodeSearchResult` (and any new output type) to `typeRegistry`.
- Regenerate `cmd/testdata/manifest.golden.json` via `go test ./cmd/ -update`.
- `TestEveryLeafIsRegistered` and `TestManifestSnapshot` enforce both.

## Testing

TDD, stdlib `net/http/httptest` only, in `pkg/bitbucket/search_test.go`:

- `Code`: asserts `search_query` and `pagelen` construction, `--limit` stop
  behaviour, "next" pagination following, decode of `content_matches`, and
  `APIError` mapping on 4xx.
- `Repos`: asserts BBQL `q=` construction and limit behaviour.
- PR text search: asserts the new `PRListOptions.Query` path folds into BBQL
  correctly without disturbing existing filters.

`cmd/` stays untested per repo convention (thin Cobra wiring).

## Documentation sync (same commit as implementation)

Required by CLAUDE.md whenever commands/flags change:

1. `README.md` - add the `bb search code/repos/prs` block to the Commands
   reference and any narrative examples.
2. `llms.txt` - add the condensed command reference with flag shapes.
3. `CLAUDE.md` - add `search` to the command hierarchy tree and a
   `client.Search(workspace).Code(...)` example to the Client-pattern block.

## Out of scope / YAGNI

- `search commits`, `search issues` (no backing API).
- Global (cross-repo) PR search (no backing API).
- Regex code search (Bitbucket does not support it).
- TUI integration for search (separate follow-up if desired).

## Open item resolved during implementation

The exact set of supported `search_query` modifiers (`ext:`, `repo:`,
`project:`, possibly others) determines which `search code` convenience flags
ship. Raw-query passthrough guarantees the feature works regardless of which
flags are added.
