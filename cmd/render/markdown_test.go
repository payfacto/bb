package render_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
)

// stripANSI removes ANSI escape sequences so tests can check plain text content.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiEscape.ReplaceAllString(s, "") }

func TestRenderMarkdown_headings(t *testing.T) {
	out := stripANSI(render.RenderMarkdown("## Hello"))
	if !strings.Contains(out, "Hello") {
		t.Errorf("expected heading text in output, got: %q", out)
	}
}

func TestRenderMarkdown_plaintext(t *testing.T) {
	out := stripANSI(render.RenderMarkdown("just plain text"))
	if !strings.Contains(out, "just plain text") {
		t.Errorf("expected plain text preserved, got: %q", out)
	}
}

func TestRenderMarkdown_empty(t *testing.T) {
	out := render.RenderMarkdown("")
	// Should not panic; empty or whitespace-only output is fine
	_ = out
}
