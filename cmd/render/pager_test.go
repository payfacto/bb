package render_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
)

// TestMaybePage_nonTTY verifies that MaybePage writes content directly to
// the provided writer when not in a TTY (always the case in test runs).
func TestMaybePage_nonTTY(t *testing.T) {
	content := strings.Repeat("line\n", 5)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	render.MaybePage(content)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)

	if buf.String() != content {
		t.Errorf("expected content printed directly, got: %q", buf.String())
	}
}
