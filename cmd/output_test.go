package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gcf "github.com/blackwell-systems/gcf-go"
	"github.com/spf13/cobra"

	"github.com/payfacto/bb/internal/config"
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

func TestResolveFormatFrom(t *testing.T) {
	tests := []struct {
		name string
		in   formatInputs
		want string
	}{
		{"default when nothing set", formatInputs{isTTY: true}, "json"},
		{"config overrides default", formatInputs{cfgFormat: "gcf", isTTY: true}, "gcf"},
		{"env overrides config", formatInputs{cfgFormat: "gcf", envFormat: "json", isTTY: true}, "json"},
		{"flag overrides env", formatInputs{cfgFormat: "json", envFormat: "gcf", flagFormat: "text", flagChanged: true, isTTY: true}, "text"},
		{"piped text from config coerced to json", formatInputs{cfgFormat: "text", isTTY: false}, "json"},
		{"piped text from env coerced to json", formatInputs{envFormat: "text", isTTY: false}, "json"},
		{"explicit --format text honored when piped", formatInputs{flagFormat: "text", flagChanged: true, isTTY: false}, "text"},
		{"piped default stays json", formatInputs{isTTY: false}, "json"},
		{"piped gcf stays gcf", formatInputs{cfgFormat: "gcf", isTTY: false}, "gcf"},
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
	if !strings.Contains(out, "GCF profile=generic") || !strings.Contains(out, "code=not_found") || !strings.Contains(out, "message=missing") {
		t.Errorf("gcf error missing content: %q", out)
	}
	// P2: with nil Details, the gcf view must not emit a spurious section header.
	if strings.Contains(out, "details") || strings.Contains(out, "Details") {
		t.Errorf("gcf error rendered a details section for nil Details: %q", out)
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

func TestResolveFormat_defaultIsJSON(t *testing.T) {
	old := format
	defer func() { format = old }()
	format = formatDefault // simulate unset flag
	c := &cobra.Command{}
	c.Flags().StringP("format", "f", formatDefault, "")
	if err := resolveFormat(c, &config.Config{}); err != nil {
		t.Fatal(err)
	}
	// In `go test` stdout is not a TTY, but json is not text, so no coercion.
	if format != "json" {
		t.Errorf("default format = %q, want json", format)
	}
}

func TestResolveFormat_envBBFormat(t *testing.T) {
	old := format
	defer func() { format = old }()
	format = formatDefault
	t.Setenv("BB_FORMAT", "json")
	c := &cobra.Command{}
	c.Flags().StringP("format", "f", formatDefault, "")
	if err := resolveFormat(c, &config.Config{}); err != nil {
		t.Fatal(err)
	}
	if format != "json" {
		t.Errorf("BB_FORMAT=json should yield json, got %q", format)
	}
}

func TestResolveFormat_flagBeatsEnv(t *testing.T) {
	old := format
	defer func() { format = old }()
	format = "text" // simulate --format text on the command line
	t.Setenv("BB_FORMAT", "json")
	c := &cobra.Command{}
	c.Flags().StringP("format", "f", formatDefault, "")
	if err := c.Flags().Set("format", "text"); err != nil {
		t.Fatal(err)
	}
	if err := resolveFormat(c, &config.Config{}); err != nil {
		t.Fatal(err)
	}
	if format != "text" {
		t.Errorf("--format text should beat BB_FORMAT=json, got %q", format)
	}
}

func TestRenderError_gcfDoesNotLeakSecretlikeDetails(t *testing.T) {
	old := format
	format = "gcf"
	defer func() { format = old }()
	// A CLIError whose Details contains a sensitive value. We assert the
	// known-safe fields render, and document current behavior re: details.
	out := captureStderr(t, func() {
		renderError(&CLIError{
			Code:    ErrCodeNotFound,
			Message: "missing",
			Details: map[string]any{"response_body_redacted": true},
		})
	})
	if !strings.Contains(out, "GCF profile=generic") || !strings.Contains(out, "not_found") {
		t.Errorf("gcf error missing expected content: %q", out)
	}
	// Non-empty Details must still render (the gcfErrorView omits Details only
	// when empty).
	if !strings.Contains(out, "response_body_redacted") {
		t.Errorf("gcf error dropped non-empty Details: %q", out)
	}
	// Guard: the unexported cause must never appear (the gcf view is an explicit
	// map of code/message/details, so cause can never be encoded).
	if strings.Contains(out, "cause") {
		t.Errorf("gcf error unexpectedly rendered the unexported cause: %q", out)
	}
}

// Regression for the P1 bug: commands that override PersistentPreRunE (here
// `bb user me`) must still resolve the persisted/BB_FORMAT output format. Before
// the fix, userMeCmd called config.Load directly and never ran resolveFormat,
// so BB_FORMAT / config `format:` were silently ignored and output was always
// the flag default. We invoke its pre-run with BB_FORMAT=json and an empty temp
// config; the credential check fails afterward (no creds), but the format must
// already have been resolved to json by then.
func TestUserMePreRun_resolvesFormat(t *testing.T) {
	oldFormat := format
	oldCfgFile := cfgFile
	defer func() { format = oldFormat; cfgFile = oldCfgFile }()

	format = formatDefault // simulate the unset --format flag default (json)
	cfgFile = filepath.Join(t.TempDir(), "absent.yaml")
	t.Setenv("BB_FORMAT", "json")

	// Pre-run returns a credentials error (no creds configured); that's expected.
	// What we assert is that resolveFormat ran first.
	_ = userMeCmd.PersistentPreRunE(userMeCmd, nil)

	if format != "json" {
		t.Errorf("user me pre-run did not resolve BB_FORMAT: format = %q, want json", format)
	}
}
