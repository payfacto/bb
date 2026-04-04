# bb — Bitbucket Cloud CLI Design

**Date:** 2026-04-04  
**Status:** Approved  
**Working directory:** `/Users/jay/code/cli-bitbucket-cloud`

---

## Overview

`bb` is a Go CLI for Bitbucket Cloud REST API 2.0, covering the same feature set as the existing Python skill at `~/.claude/skills/bitbucket-cloud/scripts/bitbucket_cloud_api.py`. It is designed to be used directly by AI agents and developers from the terminal.

---

## Project Structure

```
cli-bitbucket-cloud/
├── main.go
├── cmd/
│   ├── root.go        # global flags, config loading, --format flag
│   ├── setup.go       # bb setup / bb config interactive wizard
│   ├── pr.go          # bb pr {list,get,create,diff,approve,merge,decline}
│   ├── comment.go     # bb pr comment {list,add,reply}
│   └── task.go        # bb pr task {list,complete,reopen}
├── pkg/
│   └── bitbucket/
│       ├── client.go  # HTTP transport, Basic auth, request/response helpers
│       ├── pr.go      # PRs() resource with typed methods
│       ├── comment.go # Comments() resource
│       └── task.go    # Tasks() resource
├── internal/
│   └── config/
│       └── config.go  # load ~/.bbcloud.yaml, merge env vars + flags
├── go.mod
└── go.sum
```

---

## Configuration & Auth

**Config file:** `~/.bbcloud.yaml`

```yaml
workspace: payfactopay
repo: whosoncall
username: jay@example.com
token: your-app-password
```

**Precedence (lowest → highest):**
1. Config file (`~/.bbcloud.yaml` or `--config` override)
2. Environment variables: `BITBUCKET_USER`, `BITBUCKET_TOKEN`
3. CLI flags: `--workspace`, `--repo`, `--username`, `--token`

**Global flags (all optional when config file is present):**
```
--workspace   Bitbucket workspace slug
--repo        Repository slug
--config      Path to config file (default: ~/.bbcloud.yaml)
--format      Output format: json (default) or text
```

### bb setup

Interactive wizard that creates or updates `~/.bbcloud.yaml`.

- Prompts for workspace, default repo, username, token (masked input via `golang.org/x/term`)
- Displays current config values as defaults when re-running (update mode)
- Writes the YAML file and prints the path on completion
- Accessible as both `bb setup` and `bb config` (alias)

---

## Typed Client (`pkg/bitbucket/`)

`Client` holds base URL, auth credentials, and an `http.Client`. Resource methods are accessed via scoped structs:

```go
c := bitbucket.New(cfg)

// Pull Requests
c.PRs(workspace, repo).List(ctx, state)          // returns []PR, error
c.PRs(workspace, repo).Get(ctx, prID)            // returns PR, error
c.PRs(workspace, repo).Create(ctx, input)        // returns PR, error
c.PRs(workspace, repo).Diff(ctx, prID)           // returns string, error (raw patch text — not JSON decoded)
c.PRs(workspace, repo).Approve(ctx, prID)        // returns error
c.PRs(workspace, repo).Merge(ctx, prID, strategy) // returns error
c.PRs(workspace, repo).Decline(ctx, prID)        // returns error

// Comments
c.Comments(workspace, repo, prID).List(ctx)                    // returns []Comment, error
c.Comments(workspace, repo, prID).Add(ctx, input)              // returns Comment, error
c.Comments(workspace, repo, prID).Reply(ctx, parentID, text)   // returns Comment, error

// Tasks
c.Tasks(workspace, repo, prID).List(ctx)                       // returns []Task, error
c.Tasks(workspace, repo, prID).SetState(ctx, taskID, resolved) // returns error
```

All methods return typed structs and `error`. The Cobra layer marshals results to JSON or text.

---

## Command Structure

```
bb setup                                   # interactive config wizard
bb config                                  # alias for bb setup

bb pr list   [--state OPEN|MERGED|DECLINED|SUPERSEDED]
bb pr get    --pr-id <n>
bb pr create --title <s> --from-branch <branch> --to-branch <branch> [--description <s>] [--close-source-branch]
bb pr diff   --pr-id <n>
bb pr approve --pr-id <n>
bb pr merge   --pr-id <n> [--strategy merge_commit|squash|fast_forward]
bb pr decline --pr-id <n>

bb pr comment list  --pr-id <n>
bb pr comment add   --pr-id <n> --text <s> [--file <path> --line <n>]
bb pr comment reply --pr-id <n> --comment-id <n> --text <s>

bb pr task list     --pr-id <n>
bb pr task complete --pr-id <n> --task-id <n> [--task-id <n> ...]
bb pr task reopen   --pr-id <n> --task-id <n> [--task-id <n> ...]
```

`--task-id` is a repeatable flag (not comma-separated).  
`--repo` and `--workspace` are global flags, defaulting to config file values.

---

## Output

- **Default (`--format json`):** Raw JSON from the API, or a JSON array for list commands. Suitable for AI agent consumption and piping to `jq`.
- **`--format text`:** Human-readable list/table output matching the style of the Python script.
- **Errors:** Written to stderr with a descriptive message and non-zero exit code.
  - Missing config: `Error: no credentials found — run 'bb setup' to configure`
  - HTTP errors: `Error: HTTP 404: repository not found`

---

## Error Handling

- HTTP errors: stderr message with status code and API body, exit 1
- Missing workspace/repo: stderr with hint to run `bb setup`, exit 1
- Invalid flags: Cobra built-in usage error (stderr + usage, exit 2)
- Network errors: stderr with reason, exit 1

---

## Testing

- `pkg/bitbucket/` client tested with `net/http/httptest.Server` — no real API calls
- One test file per resource: `pr_test.go`, `comment_test.go`, `task_test.go`
- `cmd/` layer not unit tested — it is thin glue; coverage comes from client tests
- No mocking frameworks — stdlib only

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | Command structure and flag parsing |
| `gopkg.in/yaml.v3` | Config file read/write |
| `golang.org/x/term` | Masked password input in `bb setup` |

All other functionality uses the Go standard library.

---

## Out of Scope

- `bb config get <key>` / `bb config set <key> <value>` subcommands
- Pagination beyond the 50/100 item defaults already used by the Python script
- Branch, pipeline, or repository management commands
- OAuth / Bearer token auth (App Passwords + Basic auth only)
