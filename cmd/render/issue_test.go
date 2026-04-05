package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestIssueListString_empty(t *testing.T) {
	out := render.IssueListString(nil)
	if !strings.Contains(out, "No issues found.") {
		t.Errorf("got: %q", out)
	}
}

func TestIssueListString_row(t *testing.T) {
	issues := []bitbucket.Issue{{ID: 5, State: "open", Kind: "bug", Title: "Login broken on Safari"}}
	out := render.IssueListString(issues)
	if !strings.Contains(out, "#5") {
		t.Errorf("expected ID, got: %q", out)
	}
	if !strings.Contains(out, "open") {
		t.Errorf("expected state, got: %q", out)
	}
	if !strings.Contains(out, "bug") {
		t.Errorf("expected kind, got: %q", out)
	}
	if !strings.Contains(out, "Login broken") {
		t.Errorf("expected title, got: %q", out)
	}
}

func TestIssueDetailString_fields(t *testing.T) {
	issue := bitbucket.Issue{
		ID:       5,
		Title:    "Login broken on Safari",
		State:    "open",
		Kind:     "bug",
		Priority: "major",
		Reporter: bitbucket.Actor{DisplayName: "alice"},
		Content:  bitbucket.Content{Raw: "## Steps\n\n1. Open Safari\n2. Login fails"},
	}
	out := render.IssueDetailString(issue)
	for _, want := range []string{"#5", "Login broken", "open", "bug", "major", "alice", "Steps"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}

func TestIssueDetailString_noContent(t *testing.T) {
	issue := bitbucket.Issue{
		ID:       1,
		Title:    "T",
		State:    "open",
		Kind:     "bug",
		Reporter: bitbucket.Actor{DisplayName: "x"},
	}
	out := render.IssueDetailString(issue)
	if strings.Count(out, "────") > 1 {
		t.Errorf("expected no extra separator, got:\n%s", out)
	}
}
