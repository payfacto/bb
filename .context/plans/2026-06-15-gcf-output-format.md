# GCF Output Format Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add GCF (Graph Compact Format) as a third output format for `bb`, make it the built-in default with a `BB_FORMAT=json` escape hatch, and let users persist a preferred format in config + the setup wizard.

**Architecture:** Approach C from the spec — consolidate all format concerns (allowed set, precedence resolution, non-TTY guard, value rendering, error rendering) into one new `cmd/output.go`. `printOutput` keeps its name (67 call sites) but delegates to the new `renderValue`. `cmd/errors.go` and `cmd/root.go` call into `cmd/output.go`. The `cmd/render` package stays the `text` backend, unchanged.

**Tech Stack:** Go 1.26, Cobra, `github.com/blackwell-systems/gcf-go` v1.2.0 (`EncodeGeneric(any) string`), stdlib `encoding/json`, `golang.org/x/term` (TTY detection), stdlib `testing`.

---

## Verified facts (from live inspection of gcf-go v1.2.0)

- Public function: `func EncodeGeneric(data any) string`. Confirmed via `go doc`.
- `EncodeGeneric` uses **Go struct field names, sorted alphabetically**, and **ignores `json:` tags**. Example: a `[]PR{{ID,Title,State,Author}}` encodes as:
  ```
  GCF profile=generic
  ## [2]{Author,ID,State,Title}
  alice|1|OPEN|Add login
  bob|2|MERGED|Fix bug
  ```
- A single struct encodes as `key=value` lines under a `GCF profile=generic` header.
- A nested struct (e.g. `{"error": CLIError{...}}`) encodes with an indented `## Error` section.
- `go get github.com/blackwell-systems/gcf-go@v1.2.0` bumps the `go` directive (gcf-go requires go >= 1.26.1). This is expected and accepted; CI reads the Go version from `go.mod` (`setup-go` with go-version-file), so no workflow edit is required, but `TECHSTACK.md` must be updated.

## Spec refinements locked in here (update the spec to match)

- **Non-TTY guard:** coerce `text` -> `gcf` when stdout is not a TTY **only if `--format` was not set on this invocation** (`cmd.Flags().Changed("format") == false`). A persistent `text` preference (config/env) is coerced when piped; an explicit per-command `--format text` is honored.
- **Default + allowed set live in `cmd/output.go`**, not `config`. `config.Load` does **not** default `Format` (empty string means "unset"), so the precedence chain can detect whether the user set it.

## File structure

- Create: `cmd/output.go` — `allowedFormats`, `formatDefault`, `validateFormat`, `formatInputs`, `resolveFormatFrom`, `resolveFormat`, `renderValue`, `renderError`.
- Create: `cmd/output_test.go` — tests for all of the above.
- Modify: `cmd/root.go` — `--format` default `gcf` + usage; replace the lines 74-79 non-TTY guard with a `resolveFormat` call; `printOutput` delegates to `renderValue`.
- Modify: `cmd/errors.go` — `emitError` delegates to `renderError`; keep the JSON path identical.
- Modify: `internal/config/config.go` — add `Format string` field (no default in `Load`).
- Modify: `internal/config/config_auth_test.go` — `Format` round-trip test.
- Modify: `cmd/tui/setup.go` — add a `setupFieldFormat` picker slot beside the theme selector.
- Modify: `cmd/testdata/manifest.golden.json` — regenerate (`go test ./cmd/ -update`).
- Modify: `README.md`, `llms.txt`, `CLAUDE.md`, `.context/TECHSTACK.md` — docs sync.
- Modify: `go.mod`, `go.sum` — add gcf-go.

---

### Task 1: Add gcf-go dependency and lock in its real behavior with a characterization test

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `cmd/output.go` (minimal — just the import anchor + characterization is in test)
- Test: `cmd/output_test.go`

- [ ] **Step 1: Add the dependency**

Run:
```bash
GOFLAGS=-mod=mod go get github.com/blackwell-systems/gcf-go@v1.2.0
```
Expected: `go: added github.com/blackwell-systems/gcf-go v1.2.0` and the `go` directive in `go.mod` bumped to `1.26.1` (accepted).

- [ ] **Step 2: Write the failing characterization test**

Create `cmd/output_test.go`:
```go
package cmd

import (
	"strings"
	"testing"

	gcf "github.com/blackwell-systems/gcf-go"
)

// Locks in gcf-go's documented behavior: field names come from Go struct
// fields (alphabetical), NOT json tags; slices become pipe-separated tables.
func TestGCFEncodeGeneric_sliceBecomesTable(t *testing.T) {
	type row struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	out := gcf.EncodeGeneric([]row{{ID: 1, Title: "x"}})
	if !strings.Contains(out, "GCF profile=generic") {
		t.Fatalf("missing GCF header: %q", out)
	}
	if !strings.Contains(out, "{ID,Title}") {
		t.Errorf("expected Go field names ID,Title (not json tags); got: %q", out)
	}
	if !strings.Contains(out, "1|x") {
		t.Errorf("expected pipe-separated row 1|x; got: %q", out)
	}
}
```

- [ ] **Step 3: Run test to verify it passes (characterization)**

Run: `go test ./cmd/ -run TestGCFEncodeGeneric_sliceBecomesTable -v`
Expected: PASS. (This test documents real behavior; if it fails, gcf-go changed and the plan's assumptions need revisiting.)

- [ ] **Step 4: Verify the module still builds**

Run: `go build ./...`
Expected: no output, exit 0.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum cmd/output_test.go
git commit -m "build: add gcf-go dependency and characterization test"
```

---

### Task 2: Add the `Format` field to config

**Files:**
- Modify: `internal/config/config.go:51` (struct field block)
- Test: `internal/config/config_auth_test.go`

- [ ] **Step 1: Write the failing round-trip test**

Append to `internal/config/config_auth_test.go`:
```go
func TestFormatRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	if err := (&Config{Workspace: "w", Username: "u", Format: "gcf"}).Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != "gcf" {
		t.Errorf("Format = %q, want %q", got.Format, "gcf")
	}
}

func TestFormatUnsetIsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	if err := (&Config{Workspace: "w", Username: "u"}).Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != "" {
		t.Errorf("unset Format should stay empty (precedence depends on it); got %q", got.Format)
	}
}
```

If `filepath` is not already imported in this test file, add it to the import block.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestFormat -v`
Expected: FAIL — `got.Format undefined (type *Config has no field Format)`.

- [ ] **Step 3: Add the field**

In `internal/config/config.go`, add to the `Config` struct after the `Theme` field (line 51):
```go
	Theme             string `yaml:"theme,omitempty"`
	Format            string `yaml:"format,omitempty"`
```
Do NOT add a default for `Format` in `Load` (unlike `Theme`); an empty value must remain empty so the format precedence chain in `cmd/output.go` can tell "unset" from "set to gcf".

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestFormat -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_auth_test.go
git commit -m "feat(config): add persisted output format field"
```

---

### Task 3: `validateFormat` and the allowed-format set

**Files:**
- Modify: `cmd/output.go`
- Test: `cmd/output_test.go`

- [ ] **Step 1: Write the failing test**

Append to `cmd/output_test.go`:
```go
import "errors" // add to existing import block

func TestValidateFormat(t *testing.T) {
	for _, f := range []string{"gcf", "json", "text"} {
		if err := validateFormat(f); err != nil {
			t.Errorf("validateFormat(%q) = %v, want nil", f, err)
		}
	}
	err := validateFormat("yaml")
	if err == nil {
		t.Fatal("validateFormat(\"yaml\") = nil, want error")
	}
	var cliErr *CLIError
	if !errors.As(err, &cliErr) || cliErr.Code != ErrCodeValidationFailed {
		t.Errorf("want *CLIError with code %q, got %v", ErrCodeValidationFailed, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestValidateFormat -v`
Expected: FAIL — `undefined: validateFormat`.

- [ ] **Step 3: Write the implementation**

Create/extend `cmd/output.go`:
```go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	gcf "github.com/blackwell-systems/gcf-go"
)

// formatDefault is the built-in output format when nothing is configured.
const formatDefault = "gcf"

// allowedFormats is the single source of truth for valid --format values.
// Order matters: it is the order shown in error messages and the wizard.
var allowedFormats = []string{"gcf", "json", "text"}

// validateFormat returns a validation_failed CLIError if f is not allowed.
func validateFormat(f string) error {
	for _, a := range allowedFormats {
		if f == a {
			return nil
		}
	}
	return newCLIError(ErrCodeValidationFailed,
		fmt.Sprintf("invalid format %q (allowed: %s)", f, strings.Join(allowedFormats, ", ")),
		nil)
}
```

(The `gcf`, `json`, `os` imports are used by later tasks in this same file; if the compiler complains about unused imports at this step, add them in Task 5 instead and keep only `fmt`/`strings` here. Prefer adding all imports now and completing Task 5 in the same session.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestValidateFormat -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/output.go cmd/output_test.go
git commit -m "feat(cmd): add format validation and allowed-format set"
```

---

### Task 4: `resolveFormat` precedence and the non-TTY guard

**Files:**
- Modify: `cmd/output.go`
- Test: `cmd/output_test.go`

- [ ] **Step 1: Write the failing test**

Append to `cmd/output_test.go`:
```go
func TestResolveFormatFrom(t *testing.T) {
	tests := []struct {
		name string
		in   formatInputs
		want string
	}{
		{"default when nothing set", formatInputs{isTTY: true}, "gcf"},
		{"config overrides default", formatInputs{cfgFormat: "json", isTTY: true}, "json"},
		{"env overrides config", formatInputs{cfgFormat: "json", envFormat: "gcf", isTTY: true}, "gcf"},
		{"flag overrides env", formatInputs{cfgFormat: "json", envFormat: "gcf", flagFormat: "text", flagChanged: true, isTTY: true}, "text"},
		{"piped text from config coerced to gcf", formatInputs{cfgFormat: "text", isTTY: false}, "gcf"},
		{"piped text from env coerced to gcf", formatInputs{envFormat: "text", isTTY: false}, "gcf"},
		{"explicit --format text honored when piped", formatInputs{flagFormat: "text", flagChanged: true, isTTY: false}, "text"},
		{"piped gcf stays gcf", formatInputs{isTTY: false}, "gcf"},
		{"piped json stays json", formatInputs{cfgFormat: "json", isTTY: false}, "json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveFormatFrom(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveFormatFrom(%+v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveFormatFrom_invalid(t *testing.T) {
	_, err := resolveFormatFrom(formatInputs{cfgFormat: "yaml", isTTY: true})
	if err == nil {
		t.Fatal("want error for invalid config format, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestResolveFormatFrom -v`
Expected: FAIL — `undefined: formatInputs` / `undefined: resolveFormatFrom`.

- [ ] **Step 3: Write the implementation**

Add to `cmd/output.go`:
```go
// formatInputs captures everything resolveFormat needs, so the precedence
// logic and non-TTY guard can be unit-tested without a real terminal or cobra.
type formatInputs struct {
	cfgFormat   string // ~/.bbcloud.yaml "format:" ("" = unset)
	envFormat   string // BB_FORMAT ("" = unset)
	flagFormat  string // --format value
	flagChanged bool   // was --format set on THIS invocation
	isTTY       bool   // stdout is a terminal
}

// resolveFormatFrom applies precedence (default < config < env < --format) and
// the non-TTY text guard: when stdout is not a terminal and the resolved format
// is the human "text" format, coerce to gcf UNLESS --format was set on this
// invocation (a per-command explicit choice is always honored).
func resolveFormatFrom(in formatInputs) (string, error) {
	f := formatDefault
	if in.cfgFormat != "" {
		f = in.cfgFormat
	}
	if in.envFormat != "" {
		f = in.envFormat
	}
	if in.flagChanged {
		f = in.flagFormat
	}
	if err := validateFormat(f); err != nil {
		return "", err
	}
	if !in.isTTY && f == "text" && !in.flagChanged {
		f = "gcf"
	}
	return f, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/ -run TestResolveFormatFrom -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add cmd/output.go cmd/output_test.go
git commit -m "feat(cmd): add format precedence resolution with non-TTY guard"
```

---

### Task 5: `renderValue` (gcf/json/text dispatch) and repoint `printOutput`

**Files:**
- Modify: `cmd/output.go`
- Modify: `cmd/root.go:157-166` (the `printOutput` body)
- Test: `cmd/output_test.go`

- [ ] **Step 1: Write the failing test**

Append to `cmd/output_test.go`:
```go
func TestRenderValue_gcf(t *testing.T) {
	old := format
	format = "gcf"
	defer func() { format = old }()

	out := captureStdout(t, func() {
		_ = renderValue([]struct {
			ID    int
			Title string
		}{{ID: 7, Title: "hi"}}, func() { t.Fatal("textFn must not run for gcf") })
	})
	if !strings.Contains(out, "GCF profile=generic") || !strings.Contains(out, "7|hi") {
		t.Errorf("gcf output missing expected content: %q", out)
	}
}

func TestRenderValue_textCallsTextFn(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()

	called := false
	_ = renderValue(struct{}{}, func() { called = true })
	if !called {
		t.Error("textFn was not called for format=text")
	}
}

func TestRenderValue_jsonDefault(t *testing.T) {
	old := format
	format = "json"
	defer func() { format = old }()

	out := captureStdout(t, func() {
		_ = renderValue(map[string]int{"n": 1}, func() { t.Fatal("textFn must not run for json") })
	})
	if !strings.Contains(out, "\"n\": 1") {
		t.Errorf("json output missing expected content: %q", out)
	}
}
```

Add this helper to `cmd/output_test.go` (and `"io"`, `"os"` to its imports):
```go
// captureStdout redirects os.Stdout for the duration of fn and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	b, _ := io.ReadAll(r)
	return string(b)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestRenderValue -v`
Expected: FAIL — `undefined: renderValue`.

- [ ] **Step 3: Write `renderValue` and repoint `printOutput`**

Add to `cmd/output.go`:
```go
// renderValue writes v to stdout in the active format. For "text" it delegates
// to textFn (the per-command human renderer backed by cmd/render).
func renderValue(v any, textFn func()) error {
	switch format {
	case "text":
		textFn()
		return nil
	case "gcf":
		fmt.Fprint(os.Stdout, gcf.EncodeGeneric(v))
		return nil
	default: // json
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
}
```

In `cmd/root.go`, replace the existing `printOutput` (lines 157-166) with:
```go
// printOutput renders v in the active output format (see cmd/output.go).
// textFn supplies the human-readable rendering used by --format text.
func printOutput(v any, textFn func()) error {
	return renderValue(v, textFn)
}
```
Remove the now-unused `encoding/json` import from `cmd/root.go` ONLY if no other use remains (search the file first: `grep -n "json\." cmd/root.go`). If `json` is still used elsewhere in root.go, leave the import.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -run 'TestRenderValue|TestValidateFormat|TestResolveFormatFrom|TestGCFEncodeGeneric' -v`
Then: `go build ./...`
Expected: all PASS; build clean.

- [ ] **Step 5: Commit**

```bash
git add cmd/output.go cmd/output_test.go cmd/root.go
git commit -m "feat(cmd): add renderValue dispatch and delegate printOutput"
```

---

### Task 6: Format-aware error rendering

**Files:**
- Modify: `cmd/output.go` (add `renderError`)
- Modify: `cmd/errors.go:163-174` (`emitError` delegates)
- Test: `cmd/output_test.go`

- [ ] **Step 1: Write the failing test**

Append to `cmd/output_test.go` (add `"io"`/`"os"` if not already imported, and a `captureStderr` helper mirroring `captureStdout` but for `os.Stderr`):
```go
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old
	b, _ := io.ReadAll(r)
	return string(b)
}

func TestRenderError_jsonUnchanged(t *testing.T) {
	old := format
	format = "json"
	defer func() { format = old }()
	out := captureStderr(t, func() {
		renderError(&CLIError{Code: ErrCodeNotFound, Message: "missing"})
	})
	if !strings.Contains(out, `"error"`) || !strings.Contains(out, `"code":"not_found"`) {
		t.Errorf("json error envelope changed: %q", out)
	}
}

func TestRenderError_gcf(t *testing.T) {
	old := format
	format = "gcf"
	defer func() { format = old }()
	out := captureStderr(t, func() {
		renderError(&CLIError{Code: ErrCodeNotFound, Message: "missing"})
	})
	if !strings.Contains(out, "GCF profile=generic") || !strings.Contains(out, "not_found") {
		t.Errorf("gcf error missing content: %q", out)
	}
}

func TestRenderError_text(t *testing.T) {
	old := format
	format = "text"
	defer func() { format = old }()
	out := captureStderr(t, func() {
		renderError(&CLIError{Code: ErrCodeNotFound, Message: "missing"})
	})
	if !strings.Contains(out, "error: not_found: missing") {
		t.Errorf("text error missing content: %q", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -run TestRenderError -v`
Expected: FAIL — `undefined: renderError`.

- [ ] **Step 3: Implement `renderError` and delegate `emitError`**

Add to `cmd/output.go`:
```go
// renderError writes a CLIError to stderr in the active format. The JSON path
// preserves the exact historical envelope so JSON consumers see no change.
func renderError(e *CLIError) {
	switch format {
	case "text":
		fmt.Fprintf(os.Stderr, "error: %s: %s\n", e.Code, e.Message)
	case "gcf":
		fmt.Fprint(os.Stderr, gcf.EncodeGeneric(map[string]any{"error": e}))
	default: // json
		emitErrorJSON(e)
	}
}
```

In `cmd/errors.go`, rename the existing `emitError` body to `emitErrorJSON` and make `emitError` delegate:
```go
// emitError writes the error to stderr in the active output format.
func emitError(e *CLIError) {
	renderError(e)
}

// emitErrorJSON writes a single JSON object to stderr in the shape
//
//	{"error": {"code": "...", "message": "...", "details": {...}}}
//
// followed by a newline. stdout is left untouched so successful payloads on
// stdout and error payloads on stderr never interleave.
func emitErrorJSON(e *CLIError) {
	payload := map[string]any{"error": e}
	b, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error":{"code":%q,"message":%q}}`+"\n",
			ErrCodeInternalError, e.Message)
		return
	}
	fmt.Fprintln(os.Stderr, string(b))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -run TestRenderError -v`
Then: `go build ./...`
Expected: all PASS; build clean.

- [ ] **Step 5: Commit**

```bash
git add cmd/output.go cmd/errors.go cmd/output_test.go
git commit -m "feat(cmd): render errors in the active output format"
```

---

### Task 7: Wire format resolution into the root command and flip the default

**Files:**
- Modify: `cmd/root.go:128` (flag default + usage)
- Modify: `cmd/root.go:67-107` (PersistentPreRunE — replace lines 74-79 guard)
- Modify: `cmd/output.go` (add the cobra-aware `resolveFormat` wrapper)

- [ ] **Step 1: Add the cobra-aware wrapper to `cmd/output.go`**

```go
import (
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"github.com/payfacto/bb/internal/config"
)

// resolveFormat reads the effective format from cobra flags, BB_FORMAT, and
// cfg, applies the non-TTY guard, and stores the result in the package-level
// `format` var used by renderValue/renderError. Call once in PersistentPreRunE
// AFTER config is loaded.
func resolveFormat(cmd *cobra.Command, cfg *config.Config) error {
	resolved, err := resolveFormatFrom(formatInputs{
		cfgFormat:   cfg.Format,
		envFormat:   os.Getenv("BB_FORMAT"),
		flagFormat:  format,
		flagChanged: cmd.Flags().Changed("format"),
		isTTY:       term.IsTerminal(int(os.Stdout.Fd())),
	})
	if err != nil {
		return err
	}
	format = resolved
	return nil
}
```
(Merge these imports into the existing `cmd/output.go` import block.)

- [ ] **Step 2: Change the flag default and usage in `cmd/root.go:128`**

Replace:
```go
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "json", "output format: json or text")
```
with:
```go
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", formatDefault, "output format: gcf, json, or text")
```

- [ ] **Step 3: Replace the old non-TTY guard in PersistentPreRunE**

In `cmd/root.go`, delete lines 74-79 (the `if !cmd.Flags().Changed("format") && !term.IsTerminal(...) { format = "json" }` block) and instead call `resolveFormat` right after the config is loaded. The block becomes:
```go
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return err
		}
		if err := resolveFormat(cmd, cfg); err != nil {
			return err
		}
		cfg.Apply(workspace, repo, username, token)
		if err := cfg.Validate(); err != nil {
			return err
		}
```
If `golang.org/x/term` is now unused in `root.go` (it was only used by the deleted guard), remove its import from `root.go` (it is still imported by `output.go`). Verify with `grep -n "term\." cmd/root.go`.

- [ ] **Step 4: Write an integration-ish test for the wired default**

Append to `cmd/output_test.go`:
```go
func TestResolveFormat_defaultIsGCF(t *testing.T) {
	old := format
	defer func() { format = old }()
	format = formatDefault // simulate unset flag
	c := &cobra.Command{}
	c.Flags().StringP("format", "f", formatDefault, "")
	if err := resolveFormat(c, &config.Config{}); err != nil {
		t.Fatal(err)
	}
	// In `go test` stdout is not a TTY, but gcf is not text, so no coercion.
	if format != "gcf" {
		t.Errorf("default format = %q, want gcf", format)
	}
}
```
Add `"github.com/spf13/cobra"` and `"github.com/payfacto/bb/internal/config"` to the test imports.

- [ ] **Step 5: Run tests and build**

Run: `go test ./cmd/... ./internal/... -v`
Then: `go vet ./...` and `go build -o bb .`
Expected: all PASS; vet clean; binary builds.

- [ ] **Step 6: Manual smoke check**

Run:
```bash
./bb pr list --workspace x --repo y 2>&1 | head -1   # may auth-fail, but should emit GCF or a GCF/json error
BB_FORMAT=json ./bb --help >/dev/null && echo "json env accepted"
./bb pr list -f yaml 2>&1 | head -1                   # expect validation_failed about invalid format
```
Expected: invalid `-f yaml` produces a `validation_failed` error; `BB_FORMAT=json` is accepted.

- [ ] **Step 7: Commit**

```bash
git add cmd/root.go cmd/output.go cmd/output_test.go
git commit -m "feat(cmd): default to gcf and resolve format via precedence chain"
```

---

### Task 8: Add the format picker to the setup wizard

**Files:**
- Modify: `cmd/tui/setup.go`

This mirrors the existing theme selector. The theme slot uses `setupFieldTheme`, `themeIdx`, `themeNames`, and left/right cycling (see `setup.go:21,106-114,278-320`).

- [ ] **Step 1: Add a format field slot**

In the `const` block (around `setup.go:17-27`), insert `setupFieldFormat` before `setupFieldCount`:
```go
	setupFieldWorkspace = iota
	setupFieldRepo
	setupFieldUsername
	setupFieldPassword
	setupFieldTheme
	setupFieldFormat
	setupFieldCount
)
```
`setupTextFieldCount` must remain `= setupFieldTheme` (only the 4 text inputs are textinputs; theme and format are pickers).

- [ ] **Step 2: Add format state to the model and init**

Add `formatIdx int` to the setup model struct (beside `themeIdx`). In the constructor, initialize it from `existing.Format` (default to the index of `gcf` when empty):
```go
	fIdx := 0 // gcf
	for i, name := range setupFormatNames {
		if name == existing.Format {
			fIdx = i
		}
	}
	// ...assign m.formatIdx = fIdx in the returned model
```
Add the package-level slice near the theme names:
```go
var setupFormatNames = []string{"gcf", "json", "text"}
```

- [ ] **Step 3: Handle left/right when the format slot is focused**

In the key handler (mirroring `setup.go:106-114`), add branches:
```go
		case "left":
			if m.focus == setupFieldTheme {
				// existing theme code...
			}
			if m.focus == setupFieldFormat {
				m.formatIdx = (m.formatIdx - 1 + len(setupFormatNames)) % len(setupFormatNames)
			}
		case "right":
			if m.focus == setupFieldTheme {
				// existing theme code...
			}
			if m.focus == setupFieldFormat {
				m.formatIdx = (m.formatIdx + 1) % len(setupFormatNames)
			}
```

- [ ] **Step 4: Persist the chosen format on save**

In the save path (around `setup.go:191-218`), add:
```go
	chosenFormat := setupFormatNames[m.formatIdx]
	// ... include Format: chosenFormat in the Config literal that gets saved
```

- [ ] **Step 5: Render the format row and its help binding**

After the theme selector row (around `setup.go:278-320`), add a parallel "Format" row showing `setupFormatNames[m.formatIdx]`, highlighted when `m.focus == setupFieldFormat`, and a `←/→ change format` help binding when focused. Match the theme row's styles exactly.

- [ ] **Step 6: Build and manual-verify**

Run: `go build -o bb . && go test ./cmd/tui/...`
Then manually: `./bb setup`, Tab to the Format row, confirm ←/→ cycles gcf/json/text and that saving writes `format:` to `~/.bbcloud.yaml`.
Expected: build clean, tests pass, format persists.

- [ ] **Step 7: Commit**

```bash
git add cmd/tui/setup.go
git commit -m "feat(tui): add output format picker to setup wizard"
```

---

### Task 9: Regenerate the manifest golden and sync docs

**Files:**
- Modify: `cmd/testdata/manifest.golden.json` (regenerated)
- Modify: `README.md`, `llms.txt`, `CLAUDE.md`, `.context/TECHSTACK.md`

- [ ] **Step 1: Regenerate the manifest golden**

The `--format` flag's default (`json`->`gcf`) and usage string changed, which alters every command entry in the manifest. Regenerate:
```bash
go test ./cmd/ -update
```
Then verify the snapshot guard passes:
```bash
go test ./cmd/ -run TestManifestSnapshot -v
```
Expected: PASS. Inspect the diff (`git diff cmd/testdata/manifest.golden.json`) and confirm only `format` default/usage strings changed.

- [ ] **Step 2: Update README.md**

In the output/usage section, change the format description to list `gcf, json, text`, state that **gcf is the default**, document `BB_FORMAT`, and add a migration note:
> `bb` defaults to GCF (Graph Compact Format), a compact AI-native format. Set `BB_FORMAT=json` (or `format: json` in `~/.bbcloud.yaml`, or `--format json`) to get JSON. Output format precedence: config < `BB_FORMAT` < `--format`.

- [ ] **Step 3: Update llms.txt**

Update the format reference to `gcf | json | text`, note the `gcf` default and the `BB_FORMAT` env var, mirroring README.

- [ ] **Step 4: Update CLAUDE.md**

In the `### Output` section, update the description: default is now `gcf` via `renderValue` in `cmd/output.go`; formats are `gcf | json | text`; precedence config < `BB_FORMAT` < `--format`; the non-TTY guard now coerces `text`->`gcf` (not `json`) unless `--format` is set; errors render in the active format via `renderError`.

- [ ] **Step 5: Update .context/TECHSTACK.md**

Change the Go version line to the new `go.mod` directive (`1.26.1`), and add `github.com/blackwell-systems/gcf-go v1.2.0 — GCF output encoding` to the Core Frameworks and Libraries section.

- [ ] **Step 6: Full verification**

Run: `go test ./... && go vet ./... && go build -o bb .`
Expected: all PASS; clean build.

- [ ] **Step 7: Commit**

```bash
git add cmd/testdata/manifest.golden.json README.md llms.txt CLAUDE.md .context/TECHSTACK.md
git commit -m "docs: document gcf default + BB_FORMAT; regenerate manifest"
```

---

## Final verification checklist

- [ ] `go test ./...` passes.
- [ ] `go vet ./...` clean.
- [ ] `go build -o bb .` succeeds.
- [ ] `./bb pr list -f yaml` returns a `validation_failed` error.
- [ ] `BB_FORMAT=json ./bb ... | head` yields JSON; default (no env/flag) yields GCF.
- [ ] `./bb setup` shows and persists the Format picker.
- [ ] `git diff` on `manifest.golden.json` shows only format default/usage changes.
- [ ] README, llms.txt, CLAUDE.md, TECHSTACK.md all reflect the gcf default + BB_FORMAT.
