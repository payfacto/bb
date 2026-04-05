package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestCommitListString_empty(t *testing.T) {
	out := render.CommitListString(nil)
	if !strings.Contains(out, "No commits found.") {
		t.Errorf("expected empty message, got: %q", out)
	}
}

func TestCommitListString_row(t *testing.T) {
	commits := []bitbucket.Commit{
		{
			Hash:    "abc123def456789012345678901234567890abcd",
			Date:    "2026-04-01T10:00:00Z",
			Message: "feat: add something\n\nLonger description",
			Author:  bitbucket.CommitAuthor{Raw: "Alice <alice@example.com>"},
		},
	}
	out := render.CommitListString(commits)
	if !strings.Contains(out, "abc123de") {
		t.Errorf("expected short hash, got: %q", out)
	}
	if !strings.Contains(out, "2026-04-01") {
		t.Errorf("expected date, got: %q", out)
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected author, got: %q", out)
	}
	if !strings.Contains(out, "feat: add something") {
		t.Errorf("expected first line of message, got: %q", out)
	}
	if strings.Contains(out, "Longer description") {
		t.Errorf("expected multi-line message truncated, got: %q", out)
	}
}

func TestCommitDetailString_fields(t *testing.T) {
	c := bitbucket.Commit{
		Hash:    "abc123def456789012345678901234567890abcd",
		Date:    "2026-04-01T10:00:00Z",
		Message: "feat: add something",
		Author:  bitbucket.CommitAuthor{Raw: "Alice <alice@example.com>"},
		Parents: []bitbucket.CommitParent{{Hash: "parent123456789012345678901234567890abcd"}},
	}
	out := render.CommitDetailString(c)
	for _, want := range []string{"abc123de", "2026-04-01", "Alice", "feat: add something", "parent12"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}
