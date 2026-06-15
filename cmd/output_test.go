package cmd

import (
	"errors"
	"io"
	"os"
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
