# Go Release Patterns — Payfacto

Reusable reference for Go projects that:
- Use **Bitbucket** as the source-of-truth repo
- Mirror to **GitHub** for CI builds and releases
- Distribute macOS/Linux binaries via the **Payfacto Homebrew tap**

Copy-paste and replace `<PROJECT>` with the actual binary/repo name.

---

## 1. Go Versioning with Tags and ldflags

### Version package (one file, committed once)

```go
// internal/version/version.go
package version

// Version is injected at build time via -ldflags.
// Defaults to "dev" for local builds.
var Version = "dev"
```

### Build-time injection

```bash
# Resolve version from nearest tag
VERSION=$(git describe --tags --always --dirty)

# Go build
go build -ldflags "-X '<module>/internal/version.Version=${VERSION}'" ./...

# Wails build
wails build -platform windows/amd64 \
  -ldflags "-X '<module>/internal/version.Version=${VERSION}'"
```

`git describe --tags --always --dirty` produces:
| State | Output |
|---|---|
| Exact tag | `v1.2.3` |
| After tag | `v1.2.3-4-gabcdef0` |
| Uncommitted changes | `v1.2.3-4-gabcdef0-dirty` |
| No tags yet | short SHA `abcdef0` |

### Tagging convention

```bash
git tag v1.2.3          # annotated or lightweight both work
git push origin v1.2.3  # push tag to Bitbucket; pipeline mirrors it to GitHub
```

- Always use **semver** (`vMAJOR.MINOR.PATCH`).
- Tag on `main` only.
- Never reuse a tag — delete and re-push only if the release hasn't been published yet.
- GitHub Actions triggers on `v*` — any tag starting with `v` fires a release build.

### Exposing version to users

#### CLI (manual switch)

```go
// main.go
import "github.com/<org>/<project>/internal/version"

switch os.Args[1] {
case "version", "--version", "-v":
    fmt.Println("<project> " + version.Version)
}
```

#### CLI (cobra)

```go
// cmd/root.go
var Version = "dev"   // injected via -ldflags

var rootCmd = &cobra.Command{
    Version: Version,   // cobra wires --version automatically
}
```

**Build command** — always use `.` not `./...` when passing `-o <file>`:

```bash
go build -ldflags "-X '<module>/internal/version.Version=${VERSION}'" -o build/bin/<PROJECT> .
```

> `./...` matches all packages; `-o <file>` with multiple packages fails.
> Use `.` to build only the `main` package at the repo root.

---

## 2. Bitbucket → GitHub Mirror Pipeline

Bitbucket is source of truth. GitHub hosts CI and releases only.

### One-time setup

1. **Generate SSH keypair** in Bitbucket:
   `Repository settings → Pipelines → SSH keys → Generate keys`

2. **Add the public key to GitHub** as a deploy key with write access:
   `GitHub repo → Settings → Deploy keys → Add deploy key → ✓ Allow write access`

3. **Add GitHub to known hosts** in Bitbucket:
   `Repository settings → Pipelines → SSH keys → Known hosts → Add host: github.com`

4. **Enable Pipelines** in Bitbucket:
   `Repository settings → Pipelines → Settings → Enable`

5. **Never commit directly to the GitHub mirror** — the pipeline force-pushes and will overwrite divergent state.

### `bitbucket-pipelines.yml`

```yaml
image: alpine/git:latest

clone:
  depth: full   # required for git describe --tags to work in GitHub Actions

definitions:
  steps:
    - step: &mirror-to-github
        name: Mirror to GitHub
        script:
          - git fetch origin
          - git remote add github git@github.com:payfacto/<PROJECT>.git
          - git push github refs/remotes/origin/main:refs/heads/main --force
          - git push github --tags --force

pipelines:
  branches:
    main:
      - step: *mirror-to-github
  tags:
    'v*':
      - step: *mirror-to-github
```

### Rules

- The pipeline runs on every push to `main` **and** on every `v*` tag push.
- Tags must be pushed to Bitbucket first — the pipeline propagates them to GitHub.
- `clone: depth: full` is mandatory so GitHub Actions can resolve `git describe --tags`.

---

## 3. GitHub Actions — Build and Release

### `.github/workflows/release.yml`

```yaml
name: Build

# Triggered by the Bitbucket -> GitHub mirror (bitbucket-pipelines.yml).
#
# - Push to main:   builds all platforms, uploads artifacts (14-day retention).
# - Push of v* tag: builds all platforms AND publishes a GitHub Release.

on:
  push:
    branches: [main]
    tags: ['v*']

permissions:
  contents: write   # required for gh release create/upload

jobs:
  build:
    strategy:
      fail-fast: false
      matrix:
        include:
          - os: ubuntu-latest
            platform: linux/amd64
            archive: <PROJECT>-linux-amd64.tar.gz
          - os: windows-latest
            platform: windows/amd64
            archive: <PROJECT>-windows-amd64.zip
          - os: macos-latest
            platform: darwin/arm64
            archive: <PROJECT>-darwin-arm64.zip

    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0   # full history so git describe --tags works

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'   # pin to current stable

      - name: Resolve version
        id: version
        shell: bash
        run: echo "version=$(git describe --tags --always --dirty)" >> "$GITHUB_OUTPUT"

      - name: Build
        shell: bash
        run: |
          go build \
            -ldflags "-X '<module>/internal/version.Version=${{ steps.version.outputs.version }}'" \
            -o build/bin/<PROJECT> \
            ./...

      - name: Package (Linux)
        if: matrix.os == 'ubuntu-latest'
        run: tar -czvf ${{ matrix.archive }} -C build/bin <PROJECT>

      - name: Package (Windows)
        if: matrix.os == 'windows-latest'
        shell: pwsh
        run: Compress-Archive -Path build/bin/<PROJECT>.exe -DestinationPath ${{ matrix.archive }}

      - name: Package (macOS)
        if: matrix.os == 'macos-latest'
        run: cd build/bin && zip -r ../../${{ matrix.archive }} <PROJECT>

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: ${{ matrix.archive }}
          path: ${{ matrix.archive }}
          retention-days: 14

      - name: Attach to GitHub Release
        if: startsWith(github.ref, 'refs/tags/v')
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        shell: bash
        run: |
          # Idempotent: first matrix job creates the release, others upload to it.
          gh release view "$GITHUB_REF_NAME" --repo "$GITHUB_REPOSITORY" >/dev/null 2>&1 \
            || gh release create "$GITHUB_REF_NAME" --repo "$GITHUB_REPOSITORY" \
                 --title "$GITHUB_REF_NAME" --generate-notes \
            || true
          gh release upload "$GITHUB_REF_NAME" "${{ matrix.archive }}" \
            --repo "$GITHUB_REPOSITORY" --clobber

  brew-tap:
    needs: build
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    steps:
      - name: Compute bare version (strip leading v)
        id: ver
        run: echo "version=${GITHUB_REF_NAME#v}" >> "$GITHUB_OUTPUT"

      - name: Download macOS arm64 archive
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          gh release download "$GITHUB_REF_NAME" \
            --pattern "<PROJECT>-darwin-arm64.zip" \
            --repo "$GITHUB_REPOSITORY"

      - name: Compute SHA256
        id: sha
        run: echo "arm64=$(sha256sum <PROJECT>-darwin-arm64.zip | awk '{print $1}')" >> "$GITHUB_OUTPUT"

      - name: Checkout homebrew-tap
        uses: actions/checkout@v4
        with:
          repository: payfacto/homebrew-tap
          token: ${{ secrets.HOMEBREW_TAP_TOKEN }}
          path: homebrew-tap

      - name: Update cask formula
        env:
          VERSION: ${{ steps.ver.outputs.version }}
          SHA256: ${{ steps.sha.outputs.arm64 }}
        run: |
          cat > homebrew-tap/Casks/<PROJECT>.rb << EOF
          cask "<PROJECT>" do
            version "${VERSION}"
            sha256 "${SHA256}"

            url "https://github.com/payfacto/<PROJECT>/releases/download/v#{version}/<PROJECT>-darwin-arm64.zip"
            name "<Display Name>"
            desc "<Short description>"
            homepage "https://github.com/payfacto/<PROJECT>"

            binary "<PROJECT>"   # or: app "<ProjectName>.app" for GUI apps
          end
          EOF

      - name: Commit and push
        run: |
          cd homebrew-tap
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add Casks/<PROJECT>.rb
          git diff --cached --quiet || git commit -m "chore: bump <PROJECT> to ${{ steps.ver.outputs.version }}"
          git push
```

---

## 4. Homebrew Tap — `payfacto/homebrew-tap`

Tap URL: `https://github.com/payfacto/homebrew-tap`

### One-time tap setup per new project

1. Create `Casks/<PROJECT>.rb` (or `Formula/<PROJECT>.rb` for CLI tools) in the tap repo with any placeholder content — the CI job overwrites it on first release.
2. Use `Casks/` for GUI `.app` bundles; `Formula/` for CLI binaries.

### One-time secret setup per release repo

1. Create a fine-grained PAT in GitHub with **`Contents: write`** scope on `payfacto/homebrew-tap`.
2. Add it as `HOMEBREW_TAP_TOKEN` in the release repo's secrets:
   `GitHub repo → Settings → Secrets and variables → Actions → New repository secret`

### End-user install

```bash
brew tap payfacto/tap
brew install --cask <PROJECT>   # GUI app
# or
brew install payfacto/tap/<PROJECT>   # CLI tool
```

### Formula vs Cask

| Type | File location | Use when |
|---|---|---|
| `Formula` | `Formula/<PROJECT>.rb` | CLI binary, no `.app` bundle |
| `Cask` | `Casks/<PROJECT>.rb` | macOS `.app` GUI application |

---

## 5. CI Secrets Checklist

| Secret | Repo | Value |
|---|---|---|
| `HOMEBREW_TAP_TOKEN` | Release repo (GitHub) | Fine-grained PAT, `Contents: write` on `payfacto/homebrew-tap` |
| *(none needed)* | Bitbucket | SSH keypair generated in Pipeline SSH keys settings |

---

## 6. Operational Rules

- **Source of truth is Bitbucket.** All feature branches and PRs live there.
- **Never push directly to the GitHub mirror.** The Bitbucket pipeline force-pushes and will overwrite it.
- **Tag in Bitbucket, not GitHub.** Push the tag to Bitbucket; the mirror pipeline delivers it to GitHub, which triggers the release build.
- **The `brew-tap` job is idempotent** — re-running a tag release pipeline is safe; it just re-computes the SHA256 and updates the formula.
- **`fetch-depth: 0` is mandatory** in `actions/checkout` so `git describe --tags` produces a real version string and not a bare SHA.
