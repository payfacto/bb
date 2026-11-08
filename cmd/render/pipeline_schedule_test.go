package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func sampleSchedule() bitbucket.PipelineSchedule {
	return bitbucket.PipelineSchedule{
		UUID:        "{s-1}",
		Enabled:     true,
		CronPattern: "0 0 12 * * ? *",
		Target: bitbucket.PipelineScheduleTarget{
			RefType:  "branch",
			RefName:  "main",
			Selector: bitbucket.PipelineSelector{Type: "custom", Pattern: "nightly"},
		},
		CreatedOn: "2026-11-01T10:00:00Z",
	}
}

func TestPipelineScheduleListString_empty(t *testing.T) {
	out := render.PipelineScheduleListString(nil)
	if !strings.Contains(out, "No pipeline schedules found.") {
		t.Errorf("expected empty message, got: %q", out)
	}
}

func TestPipelineScheduleListString_row(t *testing.T) {
	out := render.PipelineScheduleListString([]bitbucket.PipelineSchedule{sampleSchedule()})
	for _, want := range []string{"{s-1}", "0 0 12 * * ? *", "main", "custom: nightly", "yes"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}

func TestPipelineScheduleDetailString_fields(t *testing.T) {
	out := render.PipelineScheduleDetailString(sampleSchedule())
	for _, want := range []string{"{s-1}", "0 0 12 * * ? *", "main", "custom: nightly", "yes", "2026-11-01"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}

func TestPipelineScheduleExecutionListString(t *testing.T) {
	if out := render.PipelineScheduleExecutionListString(nil); !strings.Contains(out, "No schedule executions found.") {
		t.Errorf("expected empty message, got: %q", out)
	}
	out := render.PipelineScheduleExecutionListString([]bitbucket.PipelineScheduleExecution{
		{"type": "pipeline_schedule_execution_executed", "pipeline": map[string]any{"build_number": 7}},
	})
	for _, want := range []string{"pipeline_schedule_execution_executed", `"build_number":7`} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}
