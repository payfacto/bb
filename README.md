![bb](assets/banner.png)

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-blue)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/payfacto/bb)](https://goreportcard.com/report/github.com/payfacto/bb)

# bb — Bitbucket Cloud CLI

A Go CLI for the Bitbucket Cloud REST API v2.0. Designed AI agent consumption, but humans can use it too!

> **Disclaimer:** `bb` is an unofficial tool and is not affiliated with or endorsed by Atlassian or Bitbucket.

## Install

```bash
# From source
go install github.com/payfacto/bb@latest

# Build locally
go build -o bb .
```

## Quick Start

```bash
bb setup                    # interactive config wizard
bb                          # launch interactive TUI
bb pr list                  # list open PRs as JSON
bb pr list -f text          # human-readable output
bb pr get -p 42             # get a specific PR as JSON
```

## Interactive TUI

Running `bb` with no arguments launches a full-screen interactive terminal UI.

```
bb — Bitbucket Cloud CLI
Workspace: myworkspace  Repo: myrepo
──────────────────────────────────────────────────
▸ Pull Requests    List, review, approve, merge PRs
  Pipelines        View builds, trigger, check steps
  Branches         List and manage branches
  Commits          Browse commit history
  ...
──────────────────────────────────────────────────
↑/k up  ↓/j down  enter select  / search  q quit
```

**Navigation:** Arrow keys or `j`/`k` to move, `Enter` to drill in, `Esc` to go back. Breadcrumbs track your location (`Home > Pull Requests > #42 > Comments`).

**Key features:**
- Browse PRs, pipelines, branches, commits, tags, issues, repos, deployments, and settings
- Drill into a PR to view comments, activity, statuses, diff, and tasks
- Approve PRs instantly (`a`), merge/decline with confirmation (`m`)
- Tab-cycle through filters (OPEN/MERGED/DECLINED) on PR lists
- Type-ahead search with `/` to filter any list
- Press `r` to refresh, `q` to quit
- Reconfigure credentials from the "Setup" menu item without leaving the TUI

**First run:** If no config exists, the TUI automatically shows a setup wizard — no need to run `bb setup` first.

The TUI requires a terminal — piped or scripted usage falls back to the standard CLI.

## Authentication

`bb` supports two authentication methods.

### Option A — OAuth 2.0 (recommended)

1. In Bitbucket, go to **Personal Settings → OAuth consumers → Add consumer**
2. Set the callback URL to `http://localhost` and grant the required scopes (Repositories: Read, Pull requests: Read/Write)
3. Run:

   ```bash
   bb auth login
   ```

   Enter your Consumer Key and Consumer Secret when prompted. Your browser will open for authorization.

4. Tokens are stored securely in the OS keyring (macOS Keychain, Windows Credential Manager, Linux libsecret).

Additional auth commands:

```bash
bb auth status    # show current auth state and token source
bb auth logout    # remove stored credentials
bb auth token     # print raw token (useful for scripts)
```

### Option B — App Password

```bash
bb setup
```

Prompts for workspace, username, and a [Bitbucket App Password](https://support.atlassian.com/bitbucket-cloud/docs/app-passwords/). The app password is stored in the OS keyring (not in `~/.bbcloud.yaml`).

### CI/CD environments

For environments without an OS keyring (headless Linux, CI):

```bash
export BITBUCKET_USER=myusername
export BITBUCKET_TOKEN=myapppassword
bb pr list --workspace myws --repo myrepo
```

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--workspace SLUG` | `-w` | Bitbucket workspace slug |
| `--repo SLUG` | `-r` | Repository slug |
| `--format json\|text` | `-f` | Output format (default: `json`) |
| `--config PATH` | | Path to config file (default: `~/.bbcloud.yaml`) |

### Common Shorthands

| Short | Long | Used by |
|-------|------|---------|
| `-p` | `--pr-id` | All PR, comment, and task commands |
| `-s` | `--state` | `pr list` |
| `-b` | `--branch` | `commit list`, `pipeline trigger` |
| `-u` | `--pipeline-uuid` | `pipeline get/stop/steps/log` |
| `-c` | `--comment-id` | `comment get`, `comment reply` |
| `-t` | `--text` | `comment add`, `comment reply` |
| `-T` | `--title` | `pr create`, `issue create` |
| `-d` | `--description` | `pr create`, `issue create` |
| `-n` | `--name` | `branch create/delete`, `tag create/delete` |
| `-i` | `--id` | `issue get`, `deploy-key delete`, `restriction delete` |
| `-k` | `--kind` | `issue create` |
| `-x` | `--hash` | `commit get` |

## Commands

### Pull Requests

```
bb pr list [-s OPEN|MERGED|DECLINED|SUPERSEDED]
bb pr get -p ID
bb pr create --title "..." --from-branch BRANCH --to-branch BRANCH [-d "..."] [--close-source-branch]
bb pr diff -p ID
bb pr approve -p ID
bb pr merge -p ID [--strategy merge_commit|squash|fast_forward]
bb pr decline -p ID
bb pr activity -p ID
bb pr statuses -p ID
```

### PR Comments

```
bb pr comment list -p ID
bb pr comment get -p ID -c COMMENT_ID
bb pr comment add -p ID -t "..."
bb pr comment reply -p ID -c COMMENT_ID -t "..."
```

### PR Tasks

```
bb pr task list -p ID
bb pr task complete -p ID --task-id ID
bb pr task reopen -p ID --task-id ID
```

### Pipelines

```
bb pipeline list
bb pipeline get -u UUID
bb pipeline trigger -b BRANCH
bb pipeline stop -u UUID
bb pipeline steps -u UUID
bb pipeline log -u UUID --step-uuid UUID
```

### Branches

```
bb branch list
bb branch create -n BRANCH --from COMMIT_OR_BRANCH
bb branch delete -n BRANCH
```

### Tags

```
bb tag list
bb tag create -n TAG --from COMMIT
bb tag delete -n TAG
```

### Commits

```
bb commit list -b BRANCH
bb commit get -x HASH
bb file get --ref REF --path PATH
```

### Repositories

```
bb repo list
```

### Issues

```
bb issue list
bb issue get -i ID
bb issue create -T "..." [-d "..."] [-k bug|enhancement|proposal|task] [--priority trivial|minor|major|critical|blocker]
```

### Deployments & Environments

```
bb deployment list [--env-uuid UUID]
bb env list
bb env get --env-uuid UUID
```

### Members & Users

```
bb member list
bb user me
bb user get --account-id ID
```

### Webhooks

```
bb webhook list
bb webhook create --url URL --events EVENT,EVENT [--description "..."] [--active]
bb webhook delete --webhook-id ID
```

### Deploy Keys

```
bb deploy-key list
bb deploy-key create --key "ssh-rsa ..." --label "..."
bb deploy-key delete --key-id ID
```

### Branch Restrictions

```
bb restriction list
bb restriction create --kind KIND --pattern PATTERN [--branch-match-kind glob|branching_model]
bb restriction delete --restriction-id ID
```

### Downloads

```
bb download list
bb download file --filename NAME --dest PATH
bb download upload --file PATH
```

## Output Modes

| Mode | How to use | When |
|------|-----------|------|
| **TUI** | `bb` (no args) | Interactive exploration in a terminal |
| **JSON** | `bb pr list` (default) | Scripts, agents, piping to `jq` |
| **Text** | `bb pr list -f text` | Human-readable CLI output with color |

`bb pr diff` and `bb pipeline log` always output plain text regardless of `--format`.

Errors are written to stderr with a non-zero exit code.

## For AI Agents

See [`llms.txt`](llms.txt) for a compact machine-readable reference.

Key notes:
- All list commands return JSON arrays; single-resource commands return a JSON object.
- IDs are integers for PRs, tasks, and issues. UUIDs (with `{}` braces) for pipelines, steps, and environments.
- `workspace` and `repo` can be omitted from flags if set in `~/.bbcloud.yaml`.
- The CLI exits non-zero on API errors and prints the error to stderr.

## Development

```bash
go build -o bb .       # build
go test ./...          # run all tests
go test ./pkg/bitbucket/  # client tests only
```

Tests use `net/http/httptest` — no external API calls required.
