package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestPRMergeabilityString_listsChecksAndVerdict(t *testing.T) {
	m := bitbucket.PRMergeability{
		Mergeable: false,
		Checks: []bitbucket.MergeabilityCheck{
			{Type: "pullrequest_state_check", Status: "PASSED", State: "OPEN"},
			{Type: "git_mergeability_check", Status: "FAILED", Blocking: true, Reason: "conflicts"},
			{Type: "standard_merge_check", Status: "FAILED",
				Check:       &bitbucket.MergeCheckDefinition{Kind: "minimum_approvals"},
				Requirement: map[string]any{"minimum_approvals": float64(2)},
				Observed:    map[string]any{"approval_count": float64(1)}},
			{Type: "custom_pre_merge_check", Status: "PASSED",
				Check: &bitbucket.MergeCheckDefinition{Name: "Lint"}, Message: "all good"},
			{Type: "merge_queue_check", Status: "PASSED", Queued: true,
				MergeQueue: &bitbucket.MergeQueue{Name: "main-queue", State: "ACTIVE"}},
		},
	}

	out := render.PRMergeabilityString(m)

	for _, want := range []string{
		"PASSED", "FAILED",
		"pullrequest_state_check", "state: OPEN",
		"git_mergeability_check", "reason: conflicts", "BLOCKING",
		"standard_merge_check", "minimum_approvals", "approval_count=1", "minimum_approvals=2",
		"custom_pre_merge_check", "Lint", "all good",
		"merge_queue_check", "main-queue", "queued",
		"mergeable: no",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if got := strings.Count(out, "BLOCKING"); got != 1 {
		t.Errorf("BLOCKING appears %d times, want 1:\n%s", got, out)
	}
}

func TestPRMergeabilityString_mergeableYes(t *testing.T) {
	out := render.PRMergeabilityString(bitbucket.PRMergeability{Mergeable: true, Checks: []bitbucket.MergeabilityCheck{}})
	if !strings.Contains(out, "No checks reported.") || !strings.Contains(out, "mergeable: yes") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestPRConflictsString_empty(t *testing.T) {
	if out := render.PRConflictsString(nil); !strings.Contains(out, "No conflicts.") {
		t.Errorf("expected empty message, got: %q", out)
	}
}

func TestPRConflictsString_rows(t *testing.T) {
	out := render.PRConflictsString([]bitbucket.FileConflict{
		{Path: "README.md", Scenario: "content", Message: "modified in both"},
		{Path: "main.go", Scenario: "delete_modify"},
	})
	for _, want := range []string{"PATH", "SCENARIO", "README.md", "content", "modified in both", "main.go", "delete_modify"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
