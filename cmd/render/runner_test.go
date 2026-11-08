package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestRunnerListString_empty(t *testing.T) {
	if got := render.RunnerListString(nil); got != "No runners found.\n" {
		t.Errorf("unexpected empty output: %q", got)
	}
}

func TestRunnerListString_rows(t *testing.T) {
	runners := []bitbucket.PipelineRunner{{
		UUID:   "{runner-1}",
		Name:   "linux-runner",
		Labels: []string{"self.hosted", "linux"},
		State:  bitbucket.PipelineRunnerState{Status: "ONLINE"},
	}}
	out := render.RunnerListString(runners)
	for _, want := range []string{"NAME", "STATUS", "linux-runner", "{runner-1}", "ONLINE", "self.hosted,linux"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %q", want, out)
		}
	}
}

func TestRunnerDetailString_cordoned(t *testing.T) {
	r := bitbucket.PipelineRunner{
		UUID:      "{runner-1}",
		Name:      "linux-runner",
		Labels:    []string{"self.hosted"},
		State:     bitbucket.PipelineRunnerState{Status: "OFFLINE", Cordoned: true},
		CreatedOn: "2026-10-01T10:00:00Z",
	}
	out := render.RunnerDetailString(r)
	for _, want := range []string{"linux-runner", "{runner-1}", "OFFLINE", "Cordoned:", "yes", "self.hosted", "2026-10-01T10:00:00Z"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %q", want, out)
		}
	}
}
