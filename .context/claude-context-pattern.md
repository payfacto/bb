# Claude Code + `.context/` — Project Knowledge Convention

A reproducible pattern for giving Claude Code (and any other AI coding agent that respects `CLAUDE.md`) durable, curated, version-controlled project knowledge. Drop this whole layout into a fresh repo and the next agent that opens the project gets a coherent starting state in seconds.

The pattern is language-agnostic — adapt the standards/specs to whatever the project uses.

---

## TL;DR

```
repo/
├── CLAUDE.md                       ← project memory, auto-loaded by Claude Code
└── .context/                       ← curated, committed knowledge base
    ├── INDEX.md                    ← table of contents (imported by CLAUDE.md)
    ├── HANDOFF.md                  ← long-running session log (curated, committed)
    ├── TECHSTACK.md                ← stack reference
    ├── DESIGN.md                   ← design system / brand reference (UI projects)
    ├── RELEASE.md                  ← release runbook
    ├── CODESTYLE.md                ← code style / lint crib sheet (code projects; per-language, Java so far)
    ├── specs/                      ← design specs (YYYY-MM-DD-<topic>.md)
    ├── plans/                      ← TDD implementation plans
    ├── reference/                  ← external docs / vendor APIs / snapshots
    │   └── <topic>/
    └── tools/                      ← diagnostic helpers, gated on env vars
```

The three load-bearing ideas:

1. **`CLAUDE.md` at the repo root** is auto-loaded by Claude Code on every session and contains the project's stable description plus `@`-imports that pull in the rest of `.context/` only as needed.
2. **`.context/INDEX.md`** is a single navigation document the agent reads first to learn what's available without front-loading every file's content into its context window.
3. **`.context/HANDOFF.md`** is the long-running session log — the next agent reads the most recent block and the standing "Outstanding backlog" at the top to pick up where the last session left off.

---

## CLAUDE.md (root)

Claude Code reads `CLAUDE.md` from the working directory automatically at session start. Keep it short — under 200 lines — and use `@`-imports for everything else.

**Template:**

```markdown
@.context/INDEX.md

# <Project Name>

<One-paragraph description: what it is, who uses it, what makes it
non-obvious.>

## Tech stack

<One sentence. Full details live in TECHSTACK.md, which INDEX.md links.>

Always use the `<framework-skill>` skill when writing or reviewing
<language> code that uses <framework>.

## First-time setup after clone

```bash
# Concrete commands a new developer (or agent) runs to bring the repo
# up. Aim for copy-pasteable.
```

## Project layout

```
<tree of the 5-15 most important paths with one-line annotations>
```

## <Project-specific domain note>

<Any non-obvious convention specific to this codebase. E.g. for
Orchestra: how "agent-enabled" issues are detected via a Jira label;
how the Confluence per-project mapping works. Keep it brief; defer to
.context/specs/ for details.>

## Language

<Language>. Always use the `plugin-clean-code:<lang>` skill when
writing or reviewing <Language> code. For a project with a house style
beyond linter defaults, capture it in `.context/CODESTYLE.md` and
`@`-import it here so the rules load every session. (CODESTYLE.md is
per-language; a Java version exists so far.)
```

**The `@` syntax** — Claude Code recursively expands `@path/to/file.md` at session start. It's how a small `CLAUDE.md` pulls in the full `.context/INDEX.md` content, which in turn references (via prose links) the rest of `.context/`. The agent doesn't auto-expand every link in `INDEX.md` — only `@`-prefixed paths get loaded.

**Rule of thumb:** `@`-import only the things the agent should see on **every** session. Everything else is reachable via INDEX.md links, which the agent reads only when it needs them.

---

## .context/INDEX.md

This is the navigation hub. Every file in `.context/` gets exactly one one-line entry. The agent reads `INDEX.md` (because `CLAUDE.md` `@`-imports it) and learns the shape of the knowledge base without paying the token cost of loading every file.

**Template:**

```markdown
# <Project Name> — Context Index

## Root Files

- [HANDOFF.md](HANDOFF.md) — Long-running session log: outstanding
  backlog, condensed session summaries, decisions, open questions.
- [@TECHSTACK.md](TECHSTACK.md) — Tech stack reference (auto-imported).
- [@DESIGN.md](DESIGN.md) — Design system / brand reference
  (auto-imported, UI projects only).
- [@RELEASE.md](RELEASE.md) — Release runbook (auto-imported when
  build/release work is common).
- [@CODESTYLE.md](CODESTYLE.md) - Code style / lint crib sheet
  (auto-imported on code-heavy projects). Per-language; only a Java
  version exists so far.

## Subfolders

### `reference/`

- [reference/<topic>.md](reference/<topic>.md) — <one-line summary>.

### `specs/`

- [specs/2026-05-14-<topic>.md](specs/2026-05-14-<topic>.md) — <design spec summary>.

### `plans/`

- [plans/2026-05-14-plan-1.1-<topic>.md](plans/...) — <TDD plan summary>.

### `tools/`

- [tools/<helper>.go](tools/<helper>.go) — <one-line: what it does, how to invoke>.
```

**Convention notes:**
- The `@` prefix in `[@TECHSTACK.md](TECHSTACK.md)` is a **convention used inside INDEX.md** to signal "this file is also `@`-imported by CLAUDE.md". The agent doesn't parse it — it's a hint to humans (and to future-you).
- Entries should be one line each, ~150 chars max. INDEX.md gets loaded on every session; bloat costs tokens.
- Reorganize semantically by topic, not chronologically. Specs/plans are date-prefixed but stay grouped together.

---

## Standing documents

### `.context/TECHSTACK.md`

Stack reference. Sections cover language + runtime, frameworks, data layer, APIs, secrets, build/dependency mgmt, testing, CI/CD, infra/deployment, frontend stack, dev-experience tooling. Bullets cite version numbers — that's the whole point. The agent answers "what library does this project use for X?" by reading this file instead of grepping the lockfile.

### `.context/DESIGN.md` (UI projects)

Brand colors, typography, spacing scale, component vocabulary, do's and don'ts. The agent uses this to keep new UI work consistent with the existing design language. Skip for non-UI projects.

### `.context/RELEASE.md`

The release runbook. Pre-flight checks, tag-and-push flow, troubleshooting table. Saves the agent from having to read CI configs to answer "how do we ship a release?"

### `.context/CODESTYLE.md` (code projects; per-language)

A language-specific coding-standard crib sheet: the rules that bite in practice (formatter settings, lint/Checkstyle rules, package layout, naming, layering conventions) written out in human-readable form for the times no IDE is in the loop - Claude Code edits, ad-hoc patches, code review. `@`-import it in CLAUDE.md on code-heavy projects so it loads every session, and have the agent read it before any code edit.

**So far only a Java version exists** (PayFacto IntelliJ formatter + gecko-plugin-codequality Checkstyle rules). The same file shape works for any language - add a TypeScript/Go/Python equivalent the same way. For a polyglot repo, split per language (`CODESTYLE-java.md`, `CODESTYLE-go.md`, ...) and `@`-import each. Keep it grounded in the repo's *actual* conventions (real class names, real package tree), not a generic style guide - a borrowed CODESTYLE from another repo will carry stale examples and wrong CI claims.

---

## `.context/HANDOFF.md` — the session log

This is the most-used file in the system. Every meaningful work session ends with a new block appended at the bottom. The next agent reads the latest block (and the "Outstanding backlog" at the top) to pick up cleanly.

**File structure:**

```markdown
# <Project> — Handoff

## Goal

<2-3 sentence project mission statement. Doesn't change session to session.>

## Stack

<One paragraph stack summary. Same content as TECHSTACK.md compressed
to one paragraph for in-context use.>

---

## Outstanding backlog

Items carried across multiple sessions. Most-recent-session details
remain in full further down.

**Ship and verify**
- **<item>** — <why it matters>. (Carried since YYYY-MM-DD.)

**Code hygiene / DX**
- ...

**Phase N polish / nice-to-haves**
- ...

---

## Session history — condensed

One-paragraph summaries of historical sessions. Full detail for the
latest session is below.

**Session NN (YYYY-MM-DD).** <One paragraph: what shipped, key
decisions, important file paths.>

**Session NN+1 (YYYY-MM-DD).** ...

---

## Session — YYYY-MM-DD HH:MM (Optional title)

### Purpose

<Why this session existed.>

### What was done / What shipped

<Bullets. Be specific. Include commit hashes when known. Link to
files with the `[name](path)` shape and line numbers via `path#L42`
when useful.>

### Files changed

<Bulleted list grouped by area (Frontend / Backend / Tests / Docs).
One line per file with a short note on what changed.>

### Decisions

<Bullets on non-obvious choices. Lead with the decision, follow with
the *why* — so future-you can judge edge cases.>

### Open questions / Blockers

<Bullets. What's waiting on someone else, what's deferred, what's
still unclear.>

### Running state

- Branch: <name>, tree <clean/dirty>.
- Background processes / dev servers / worktrees.
- Anything actively running that the next session needs to know about.

### Inferred next steps

<Bullets. What the next agent should pick up. Reference the
"Outstanding backlog" entries that map to them.>

### Suggested skills for next session

- `<skill-name>` — when X.
```

**Rules:**
- Date format: `YYYY-MM-DD HH:MM` (24-hour, local time). Convert relative dates ("Thursday") to absolute when writing.
- Newest session at the bottom. As sessions accumulate, oldest ones get compressed into the "Session history — condensed" section so the file stays under ~700 lines.
- Outstanding backlog at the top is the single place to look for "what's still on the list." Clear entries when work ships; add `(Carried since YYYY-MM-DD)` when an item survives a session without progress.
- Avoid summarizing what's already in `git log` — the handoff is for non-obvious context, decisions, deferred items, open questions.

---

## `.context/specs/` — design specs

Pre-implementation design docs. Date-prefixed filenames:

```
specs/YYYY-MM-DD-<short-feature-name>-design.md
```

Each spec covers: problem, goals, non-goals, design, alternatives considered, open questions, references. Specs are written **before** plans (which are written before code). The agent should produce a spec when starting non-trivial work; the human reviews it; only then does a plan or code follow.

---

## `.context/plans/` — TDD implementation plans

Numbered, task-by-task plans for executing a spec. Filenames:

```
plans/YYYY-MM-DD-plan-<#>.<sub>-<topic>.md
```

Format: numbered tasks, each task with explicit test → impl → verify steps. The agent works through tasks one at a time, reporting completion. Useful when handing implementation work to a subagent or another session — the plan is the contract.

---

## `.context/reference/` — external knowledge

Snapshots of external docs that don't change often: vendor API references, CLI reference manuals, historical-decision PDFs, articles you keep coming back to. Subfolder per topic:

```
reference/oauth/<provider>-oauth-notes.md
reference/jira/<workspace>-summary.md
reference/<cli-tool>/CLI-REFERENCE.md
```

The agent reads these when working on that area. INDEX.md lists them with one-line summaries so the agent knows what's available.

---

## `.context/tools/` — diagnostic helpers

Single-purpose programs / scripts that aren't part of the production build but help during debugging. Env-var gated when possible. Annotate each in INDEX.md with how to invoke. Examples from Orchestra:

- A `term-replay.html` page that replays captured terminal byte streams to reproduce rendering bugs.
- A `vtdump.go` single-file program that prints a human-readable trace of VT/ANSI sequences.

If a tool needs heavy environment setup, note it in INDEX.md.

---

## `.gitignore` additions for the pattern

```gitignore
# Ephemeral handoff writes from the superpowers /handoff skill.
# The curated, committed copy lives at .context/HANDOFF.md.
HANDOFF.md

# .claude/ is committed (skills, slash commands, settings.json define the
# agent contract for this repo). Only the per-contributor and per-session
# files are ignored.
.claude/settings.local.json
.claude/.current-tool

# If using the superpowers harness:
.superpowers/
```

---

## Bootstrap script — drop this pattern into a new repo

Save as `bootstrap-claude-context.sh` and run from the repo root:

```bash
#!/usr/bin/env bash
set -euo pipefail

PROJECT_NAME="${1:-MyProject}"
LANG="${2:-go}"   # or python, typescript, etc.

mkdir -p .context/{reference,specs,plans,tools}

# --- CLAUDE.md ---
cat > CLAUDE.md <<EOF
@.context/INDEX.md

# ${PROJECT_NAME}

<One-paragraph description goes here.>

## Tech stack

<One sentence. Full details in TECHSTACK.md.>

## First-time setup after clone

\`\`\`bash
# concrete commands
\`\`\`

## Project layout

\`\`\`
src/    — production code
tests/  — tests
\`\`\`

## Language

${LANG}. Always use the \`plugin-clean-code:${LANG}\` skill when
writing or reviewing ${LANG} code.
EOF

# --- INDEX.md ---
cat > .context/INDEX.md <<'EOF'
# Context Index

## Root Files

- [HANDOFF.md](HANDOFF.md) — Long-running session log.
- [@TECHSTACK.md](TECHSTACK.md) — Stack reference.

## Subfolders

### `reference/`
- (populate with vendor / API references)

### `specs/`
- (populate with YYYY-MM-DD-<topic>-design.md)

### `plans/`
- (populate with YYYY-MM-DD-plan-#-<topic>.md)

### `tools/`
- (populate with diagnostic helpers)
EOF

# --- HANDOFF.md ---
cat > .context/HANDOFF.md <<EOF
# ${PROJECT_NAME} — Handoff

## Goal

<2-3 sentence project mission.>

## Stack

<One-paragraph stack summary.>

---

## Outstanding backlog

**Ship and verify**
- _(none yet)_

**Code hygiene / DX**
- _(none yet)_

---

## Session history — condensed

_(populated as old sessions are compressed)_

---

## Session — $(date +%Y-%m-%d) (Bootstrap)

### Purpose

Bootstrap the .context/ knowledge convention.

### What shipped

- CLAUDE.md + .context/{INDEX,HANDOFF,TECHSTACK}.md
- .gitignore entries for ephemeral state.

### Running state

- Branch: main, tree dirty (this commit).
- No background processes.
EOF

# --- TECHSTACK.md (skeleton) ---
cat > .context/TECHSTACK.md <<EOF
# TECHSTACK — ${PROJECT_NAME}

## 1. Language and Runtime
- ${LANG} <version> — primary application language.

## 2. Core Frameworks and Libraries
- _(populate)_

## 3. Data and Persistence
- _(populate)_

## 4. API and Contract Tooling
- _(populate)_

## 5. Security and Secrets
- _(populate)_

## 6. Build and Dependency Management
- _(populate)_

## 7. Testing Stack
- _(populate)_

## 8. CI/CD and Delivery
- _(populate)_

## 9. Infrastructure and Deployment
- _(populate)_
EOF

# --- .gitignore additions ---
{
  echo ''
  echo '# Ephemeral handoff writes (curated copy lives at .context/HANDOFF.md)'
  echo 'HANDOFF.md'
  echo ''  
  echo '# .claude/ is committed (skills, slash commands, settings.json define the'
  echo '# agent contract for this repo). Only the per-contributor and per-session'
  echo '# files are ignored.'
  echo '.claude/settings.local.json'
  echo '.claude/.current-tool'
  echo ''  
  echo '.superpowers/'
} >> .gitignore

echo "Done. Edit CLAUDE.md and the .context/ files to fit ${PROJECT_NAME}."
```

Run with:

```bash
bash bootstrap-claude-context.sh "MyProject" go
```

Then the agent's first task in the new repo is to populate `TECHSTACK.md` and `INDEX.md` from the actual codebase — `techstack-review-summarizer` and similar codebase-inspection skills handle this in one shot. For a code-heavy repo, also add a `.context/CODESTYLE.md` grounded in the repo's real conventions and `@`-import it from CLAUDE.md (a Java template exists today; mirror it for other languages).

---

## How an agent actually uses this in a session

1. Session opens → Claude Code reads `CLAUDE.md` → `@`-expands `.context/INDEX.md`.
2. Agent now knows: project description, stack at a glance, every file available in `.context/`.
3. User asks a question or assigns a task.
4. Agent decides which `.context/` files it needs:
   - Designing a feature? → read recent `specs/` for adjacent work, then write a new spec.
   - Hit a bug? → read the relevant `reference/` doc; check `HANDOFF.md` for "Known gotchas" or prior sessions that touched the same area.
   - Editing code? → read `CODESTYLE.md` (if present) before writing, so edits match house style and pass the linter.
   - About to ship? → read `RELEASE.md`.
5. Session ends → agent appends a new block to `.context/HANDOFF.md` and prunes any cleared items from "Outstanding backlog" at the top.
6. Commit the handoff update alongside any code changes (or as its own commit if cleaner).

---

## Anti-patterns (don't do these)

- **Putting everything in `CLAUDE.md`.** Bloats every session. Use `@`-imports and INDEX.md links instead.
- **`@`-importing every `.context/` file.** Same problem — only `@`-import the things the agent needs every session.
- **Writing a handoff block that summarizes `git log`.** The handoff is for the *non-obvious*: decisions, deferred items, surprises. The commit log is authoritative for "what changed."
- **Letting HANDOFF.md grow unbounded.** Once it's over ~700 lines, compress old sessions into one-paragraph entries in the "Session history — condensed" section.
- **Storing secrets, tokens, or PII anywhere in `.context/`.** It's all committed. Keep them in `.env`, OS keychains, or CI secrets.
- **Generating docs the agent doesn't read.** Every file in `.context/` should map to either INDEX.md (general reference) or be auto-imported in CLAUDE.md (always loaded). Orphan files are dead weight.

---

## What this pattern doesn't cover

- **User-level memory.** That's a per-developer Claude Code feature (`~/.claude/projects/<project>/memory/MEMORY.md`). It's separate from `.context/` and shouldn't be checked in — it's individual context (preferences, OS quirks, personal shortcuts).
- **Skills and plugins.** Those live under `~/.claude/` and Claude Code's plugin marketplace. The pattern here just *references* skill names in CLAUDE.md ("use the `plugin-clean-code:go` skill") — installing them is out of scope.
- **Subagents.** When an agent dispatches a subagent, the subagent reads `CLAUDE.md` too (so it inherits `.context/INDEX.md`). The agent's prompt to the subagent should mention specific `.context/` files when relevant.

---

## Why this works

- **Discoverable**: every piece of project knowledge has exactly one home, and INDEX.md is the map.
- **Bounded context window**: only `CLAUDE.md` + `@`-imported files load every session. Specifics load on demand.
- **Reproducible across sessions**: any agent (or human) opening the repo gets the same starting state.
- **Reviewable**: `.context/` is git-tracked, so changes are visible in PRs.
- **Composable with Claude Code defaults**: works with the `@`-import mechanism Claude Code already implements. No custom tooling required.
