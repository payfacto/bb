package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
)

func TestRenderMarkdown_headings(t *testing.T) {
	out := render.RenderMarkdown("## Hello")
	if !strings.Contains(out, "Hello") {
		t.Errorf("expected heading text in output, got: %q", out)
	}
}

func TestRenderMarkdown_plaintext(t *testing.T) {
	out := render.RenderMarkdown("just plain text")
	if !strings.Contains(out, "just plain text") {
		t.Errorf("expected plain text preserved, got: %q", out)
	}
}

func TestRenderMarkdown_empty(t *testing.T) {
	out := render.RenderMarkdown("")
	// Should not panic; empty or whitespace-only output is fine
	_ = out
}
