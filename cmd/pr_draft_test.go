package cmd

import (
	"errors"
	"slices"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestFilterPRsByDraft(t *testing.T) {
	prs := []bitbucket.PR{{ID: 1}, {ID: 2, Draft: true}, {ID: 3}}
	ids := func(list []bitbucket.PR) []int {
		var out []int
		for _, pr := range list {
			out = append(out, pr.ID)
		}
		return out
	}
	tests := []struct {
		name   string
		filter draftFilter
		want   []int
	}{
		{"any keeps all", draftAny, []int{1, 2, 3}},
		{"only drafts", draftOnly, []int{2}},
		{"exclude drafts", draftExclude, []int{1, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ids(filterPRsByDraft(prs, tt.filter)); !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureMergeable(t *testing.T) {
	if err := ensureMergeable(bitbucket.PR{ID: 5}); err != nil {
		t.Fatalf("non-draft PR: unexpected error %v", err)
	}
	err := ensureMergeable(bitbucket.PR{ID: 826, Draft: true})
	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("draft PR: want *CLIError, got %v", err)
	}
	if cliErr.Code != ErrCodeValidationFailed {
		t.Errorf("code = %q, want %q", cliErr.Code, ErrCodeValidationFailed)
	}
	if want := "PR #826 is a draft; mark it ready first (bb pr update -p 826 --ready)"; cliErr.Message != want {
		t.Errorf("message = %q, want %q", cliErr.Message, want)
	}
}
