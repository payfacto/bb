package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestPipelineListString_empty(t *testing.T) {
	out := render.PipelineListString(nil)
	if !strings.Contains(out, "No pipelines found.") {
		t.Errorf("expected empty message, got: %q", out)
	}
}

func TestPipelineListString_row(t *testing.T) {
	pipelines := []bitbucket.Pipeline{
		{BuildNumber: 42, State: bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "SUCCESSFUL"}}, Target: bitbucket.PipelineTarget{RefName: "main"}, CreatedOn: "2026-04-01T10:00:00Z"},
	}
	out := render.PipelineListString(pipelines)
	for _, want := range []string{"#42", "COMPLETED", "SUCCESSFUL", "main", "2026-04-01"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got: %q", want, out)
		}
	}
}

func TestPipelineDetailString_fields(t *testing.T) {
	p := bitbucket.Pipeline{BuildNumber: 42, UUID: "{abc-123}", State: bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "SUCCESSFUL"}}, Target: bitbucket.PipelineTarget{RefName: "main", Commit: &bitbucket.PipelineCommit{Hash: "abc123def456"}}, CreatedOn: "2026-04-01T10:00:00Z", CompletedOn: "2026-04-01T10:05:00Z"}
	out := render.PipelineDetailString(p)
	for _, want := range []string{"#42", "{abc-123}", "COMPLETED", "SUCCESSFUL", "main", "abc123def456"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}

func TestPipelineStepsString_empty(t *testing.T) {
	out := render.PipelineStepsString(nil)
	if !strings.Contains(out, "No steps found.") {
		t.Errorf("expected empty message, got: %q", out)
	}
}

func TestPipelineStepsString_row(t *testing.T) {
	steps := []bitbucket.PipelineStep{{UUID: "{step-1}", Name: "Build", State: bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "SUCCESSFUL"}}}}
	out := render.PipelineStepsString(steps)
	if !strings.Contains(out, "Build") {
		t.Errorf("expected step name, got: %q", out)
	}
	if !strings.Contains(out, "COMPLETED") {
		t.Errorf("expected state, got: %q", out)
	}
}
