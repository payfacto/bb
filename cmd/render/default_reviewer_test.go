package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestDefaultReviewerListString_empty(t *testing.T) {
	if got := render.DefaultReviewerListString(nil); got != "No default reviewers found.\n" {
		t.Errorf("unexpected empty output: %q", got)
	}
}

func TestDefaultReviewerListString_withSource(t *testing.T) {
	reviewers := []bitbucket.DefaultReviewer{
		{User: bitbucket.Actor{DisplayName: "Patrick Wolf", AccountID: "acc-1"}, ReviewerType: "project"},
		{User: bitbucket.Actor{DisplayName: "Davis Lee", AccountID: "acc-2"}},
	}
	out := render.DefaultReviewerListString(reviewers)
	for _, want := range []string{"NAME", "SOURCE", "Patrick Wolf", "acc-1", "project", "Davis Lee", "repository"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %q", want, out)
		}
	}
}
