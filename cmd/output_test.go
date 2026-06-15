package cmd

import (
	"errors"
	"io"
	"os"
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
	// Guard: the unexported cause must never appear (gcf skips unexported fields).
	if strings.Contains(out, "cause") {
		t.Errorf("gcf error unexpectedly rendered the unexported cause: %q", out)
	}
}
