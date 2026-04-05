package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func ep(branch string) bitbucket.Endpoint {
	var e bitbucket.Endpoint
	e.Branch.Name = branch
	return e
}

func TestPRListString_empty(t *testing.T) {
	out := render.PRListString(nil)
	if !strings.Contains(out, "No pull requests found.") {
		t.Errorf("expected empty message, got: %q", out)
	}
}

func TestPRListString_columns(t *testing.T) {
	prs := []bitbucket.PR{
		{
			ID:          42,
			Title:       "Add OAuth support",
			State:       "OPEN",
			Author:      bitbucket.Actor{DisplayName: "alice"},
			Source:      ep("feat/oauth"),
			Destination: ep("main"),
		},
	}
	out := render.PRListString(prs)
	if !strings.Contains(out, "#42") {
		t.Errorf("expected PR ID in output, got: %q", out)
	}
	if !strings.Contains(out, "Add OAuth support") {
		t.Errorf("expected title in output, got: %q", out)
	}
	if !strings.Contains(out, "OPEN") {
		t.Errorf("expected state in output, got: %q", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("expected author in output, got: %q", out)
	}
	if !strings.Contains(out, "feat/oauth") {
		t.Errorf("expected source branch in output, got: %q", out)
	}
	if !strings.Contains(out, "main") {
		t.Errorf("expected destination branch in output, got: %q", out)
	}
}

func TestPRListString_titleTruncated(t *testing.T) {
	longTitle := strings.Repeat("x", 60)
	prs := []bitbucket.PR{
		{ID: 1, Title: longTitle, State: "OPEN", Author: bitbucket.Actor{DisplayName: "bob"}, Source: ep("a"), Destination: ep("b")},
	}
	out := render.PRListString(prs)
	if strings.Contains(out, longTitle) {
		t.Errorf("expected title to be truncated, but full title appeared in: %q", out)
	}
	if !strings.Contains(out, "…") {
		t.Errorf("expected ellipsis in truncated title, got: %q", out)
	}
}

func TestPRDetailString_fields(t *testing.T) {
	var pr bitbucket.PR
	pr.ID = 42
	pr.Title = "Add OAuth support"
	pr.State = "OPEN"
	pr.Author = bitbucket.Actor{DisplayName: "alice"}
	pr.Source = ep("feat/oauth")
	pr.Destination = ep("main")
	pr.Links.HTML.Href = "https://bitbucket.org/org/repo/pull-requests/42"
	pr.Description = ""

	out := render.PRDetailString(pr)
	checks := []string{"#42", "Add OAuth support", "OPEN", "alice", "feat/oauth", "main", "https://bitbucket.org"}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in PRDetail output, got:\n%s", want, out)
		}
	}
}

func TestPRDetailString_withDescription(t *testing.T) {
	var pr bitbucket.PR
	pr.ID = 1
	pr.Title = "Test PR"
	pr.State = "OPEN"
	pr.Author = bitbucket.Actor{DisplayName: "bob"}
	pr.Source = ep("branch")
	pr.Destination = ep("main")
	pr.Description = "## Summary\n\nSome changes here."

	out := render.PRDetailString(pr)
	if !strings.Contains(out, "Summary") {
		t.Errorf("expected markdown heading text in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Some changes here") {
		t.Errorf("expected description body in output, got:\n%s", out)
	}
}

func TestPRDetailString_noDescription(t *testing.T) {
	var pr bitbucket.PR
	pr.ID = 1
	pr.Title = "T"
	pr.State = "OPEN"
	pr.Author = bitbucket.Actor{DisplayName: "x"}
	pr.Source = ep("a")
	pr.Destination = ep("b")

	out := render.PRDetailString(pr)
	// Should not contain a stray separator or crash with empty description
	if strings.Count(out, "────") > 1 {
		t.Errorf("expected at most one separator for empty description, got:\n%s", out)
	}
}
