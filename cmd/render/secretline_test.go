package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
)

func TestSecretLineGrowing(t *testing.T) {
	// prevCells 0 → no padding, no backspaces.
	out, cells := render.SecretLine("Token: ", "ab", 0)
	if out != "\rToken: ab" {
		t.Errorf("out = %q, want %q", out, "\rToken: ab")
	}
	if want := len([]rune("Token: ab")); cells != want {
		t.Errorf("cells = %d, want %d", cells, want)
	}
}

func TestSecretLineShrinkingPadsAndBackspaces(t *testing.T) {
	// Previous line "Token: ab" = 9 cells; now shown shrinks to "a" → 8 cells.
	// One leftover char must be erased: one space then one backspace.
	out, cells := render.SecretLine("Token: ", "a", 9)
	want := "\rToken: a" + " " + "\b"
	if out != want {
		t.Errorf("out = %q, want %q", out, want)
	}
	if cells != 8 {
		t.Errorf("cells = %d, want 8", cells)
	}
}

func TestSecretLineEqualLengthNoPad(t *testing.T) {
	// "P: " (3) + "xy" (2) = 5 cells, prev 5 → no pad/backspace.
	out, cells := render.SecretLine("P: ", "xy", 5)
	if strings.Contains(out, " \b") {
		t.Errorf("expected no pad/backspace, got %q", out)
	}
	if out != "\rP: xy" {
		t.Errorf("out = %q, want %q", out, "\rP: xy")
	}
	if cells != 5 {
		t.Errorf("cells = %d, want 5", cells)
	}
}

func TestSecretLineEmptyShownErasesPrevious(t *testing.T) {
	// "Token: " = 7 cells, prev 9 → pad 2 spaces + 2 backspaces.
	out, cells := render.SecretLine("Token: ", "", 9)
	want := "\rToken: " + strings.Repeat(" ", 2) + strings.Repeat("\b", 2)
	if out != want {
		t.Errorf("out = %q, want %q", out, want)
	}
	if cells != 7 {
		t.Errorf("cells = %d, want 7", cells)
	}
}
