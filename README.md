![bb](assets/banner.png)

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-blue)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/payfacto/bb)](https://goreportcard.com/report/github.com/payfacto/bb)

# bb — Bitbucket Cloud CLI & TUI

A Go CLI for the Bitbucket Cloud REST API v2.0. Designed AI agent consumption, but humans can use it too. Now with a fun TUI interface!

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
bb — Bitbucket Cloud TUI
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
- From the repo detail view, press `Enter` on **Clone SSH** or **Clone HTTPS** to run `git clone` directly — the TUI suspends, git runs with full terminal output, then the TUI resumes. Press `t` to toggle between clone mode and copy-to-clipboard mode. Default is clone mode; set `clone_action: copy` in `~/.bbcloud.yaml` to default to copy.

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

### Option B — API Token *(replaces App Passwords)*

> **Note:** Bitbucket App Passwords are being phased out by Atlassian. Generate an **API token** (or a scoped token) instead.
> See [Bitbucket API tokens](https://support.atlassian.com/bitbucket-cloud/docs/api-tokens/) for instructions.

```bash
bb setup
```

Prompts for workspace, username, and a Bitbucket API token. The token is stored in the OS keyring (not in `~/.bbcloud.yaml`).

### CI/CD environments

For environments without an OS keyring (headless Linux, CI):

```bash
export BITBUCKET_USER=myusername
export BITBUCKET_TOKEN=myapitoken
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
| `-i` | `--id` | `issue get/close/reopen`, `deploy-key delete`, `restriction delete` |
| `-k` | `--kind` | `issue create` |
| `-k` | `--key` | `pipeline-var create` |
| `-v` | `--value` | `pipeline-var create` |
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
bb pr add-reviewer -p ID --account-id ACCOUNT_ID
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
bb repo get SLUG
bb repo update SLUG [--description "..."] [--default-branch BRANCH]
bb repo delete SLUG
bb repo create SLUG [--name "..."] [--description "..."] [--private] [--project KEY]
bb repo fork SLUG [--name "..."] [--workspace SLUG]
```

### Issues

```
bb issue list
bb issue get -i ID
bb issue create -T "..." [-d "..."] [-k bug|enhancement|proposal|task] [--priority trivial|minor|major|critical|blocker]
bb issue close -i ID
bb issue reopen -i ID
```

### Pipeline Variables

```
bb pipeline-var list
bb pipeline-var create -k KEY -v VALUE [--secured]
bb pipeline-var delete --uuid UUID
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
go build -o bb .          # build (version reports as "dev")
go test ./...             # run all tests
go test ./pkg/bitbucket/  # client tests only
```

Tests use `net/http/httptest` — no external API calls required.

## For Maintainers

### Version info

`bb` embeds a version string via Go's `-ldflags -X`. The variable lives at
`github.com/payfacto/bb/cmd.Version` and defaults to `"dev"` for plain
`go build` invocations.

Check the compiled version:

```bash
bb --version
# bb version v1.2.3
```

The version is also shown in the TUI home header (`bb — Bitbucket Cloud TUI v1.2.3`).

### Building with a version stamp

The `Makefile` derives the version from the nearest git tag:

```bash
make build        # -> ./bb, version from `git describe --tags --always --dirty`
make install      # installs to $GOPATH/bin with the same version
make test         # go test ./...
make clean        # removes bb / bb.exe
```

Override the version explicitly if needed:

```bash
make build VERSION=v1.2.3
```

Equivalent raw `go build` command (what `make build` runs under the hood):

```bash
go build -ldflags "-X 'github.com/payfacto/bb/cmd.Version=v1.2.3'" -o bb .
```

### Cutting a release (step-by-step)

Releases are fully automated by GitHub Actions
(`.github/workflows/release.yml`) + [GoReleaser](https://goreleaser.com/)
(`.goreleaser.yaml`). Pushing a tag that starts with `v` triggers a build for
Linux/macOS/Windows (amd64 + arm64), uploads the archives to a GitHub Release,
and bumps the [Homebrew tap](https://github.com/payfacto/homebrew-tap).

Dummy-proof checklist:

1. **Make sure `main` is clean and green.**
   ```bash
   git checkout main
   git pull
   go test ./...
   ```

2. **Pick the next version.** We use [Semantic Versioning](https://semver.org/):
   - `vMAJOR.MINOR.PATCH` — e.g. `v1.2.3`
   - Breaking changes → bump MAJOR
   - New features → bump MINOR
   - Bug fixes only → bump PATCH
   - **The `v` prefix is required** (GoReleaser and the release workflow both
     key off tags matching `v*`).

   See existing tags for reference:
   ```bash
   git tag --list --sort=-v:refname | head
   ```

3. **Create an annotated tag.** Annotated tags (`-a`) carry a message and
   author and are what GoReleaser expects:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   ```

4. **Push the tag** (this is what fires the workflow):
   ```bash
   git push origin v1.2.3
   ```
   You can also push all tags at once with `git push --tags`, but pushing the
   specific tag is safer.

5. **Watch the build.** Go to the repo's **Actions** tab on GitHub. The
   `Release` workflow should run `go test`, then `goreleaser release --clean`.
   When it's green, a new entry appears under **Releases** with platform
   archives and a `checksums.txt`.

6. **Verify the published version:**
   ```bash
   # via Homebrew (macOS / Linux)
   brew update && brew upgrade bb
   bb --version

   # or download an archive from the GitHub Release page and run it
   ```

### Fixing a broken tag

If something went wrong and the release needs redoing for the *same* version
(rare — prefer bumping PATCH instead):

```bash
git tag -d v1.2.3                      # delete locally
git push --delete origin v1.2.3        # delete on remote
# also delete the GitHub Release via the UI, then re-tag and re-push
```

### CI secrets

The release workflow needs `HOMEBREW_TAP_TOKEN` configured as a repo secret —
a fine-grained PAT with `contents:write` on `payfacto/homebrew-tap`. Without
it the Homebrew publish step fails but the GitHub Release itself still
succeeds.

### Version-stamp wiring (how it plumbs through the code)

- `cmd/root.go` declares `var Version = "dev"` and assigns it to
  `rootCmd.Version` so `bb --version` works.
- `cmd.Execute()` passes `Version` into `tui.Run(client, cfg, version)`.
- `cmd/tui/run.go` stores it in a package-level `version` var.
- `cmd/tui/menu.go` renders it in the home header using `subtitleStyle`.
- `.goreleaser.yaml` injects the tag value via
  `-X 'github.com/payfacto/bb/cmd.Version=v{{.Version}}'` at release time.

If you rename the variable or move it to another package, update **both** the
`Makefile` and `.goreleaser.yaml` to match.
