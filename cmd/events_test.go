package cmd

import (
	"bytes"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestFormatEvent(t *testing.T) {
	tests := []struct {
		name   string
		fields []eventField
		want   string
	}{
		{"plain", []eventField{{"step", "build"}, {"state", "COMPLETED"}}, "step=build state=COMPLETED"},
		{"space is quoted", []eventField{{"step", "Deploy to prod"}}, `step="Deploy to prod"`},
		{"quote and equals are escaped", []eventField{{"a", `x="y"`}}, `a="x=\"y\""`},
		{"empty value is skipped", []eventField{{"state", "PENDING"}, {"result", ""}}, "state=PENDING"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatEvent(tt.fields...); got != tt.want {
				t.Errorf("formatEvent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func step(uuid, name, state string, manual bool) bitbucket.PipelineStep {
	return bitbucket.PipelineStep{UUID: uuid, Name: name, State: bitbucket.PipelineState{Name: state}, Manual: manual}
}

func TestStepEvents_EmitsOnlyOnChange(t *testing.T) {
	var out bytes.Buffer
	ev := newStepEvents(&out)
	ev.poll([]bitbucket.PipelineStep{step("{1}", "build", "IN_PROGRESS", false), step("{2}", "Deploy to prod", "PENDING", true)})
	ev.poll([]bitbucket.PipelineStep{step("{1}", "build", "IN_PROGRESS", false), step("{2}", "Deploy to prod", "PENDING", true)})
	done := step("{1}", "build", "COMPLETED", false)
	done.State.Result = &bitbucket.PipelineResult{Name: "SUCCESSFUL"}
	ev.poll([]bitbucket.PipelineStep{done, step("{2}", "Deploy to prod", "PENDING", true)})

	want := "step=build state=IN_PROGRESS\n" +
		`step="Deploy to prod" state=PENDING manual=true` + "\n" +
		"step=build state=COMPLETED result=SUCCESSFUL\n"
	if out.String() != want {
		t.Errorf("events =\n%s\nwant\n%s", out.String(), want)
	}
}

func TestPipelineOutcomeEvent(t *testing.T) {
	result := bitbucket.PipelineWatchResult{
		Pipeline:   bitbucket.Pipeline{BuildNumber: 77},
		Status:     bitbucket.WatchBlocked,
		ManualGate: &bitbucket.ManualGate{Step: "Deploy to prod", URL: "https://x/77"},
	}
	got := pipelineOutcomeEvent(result, "https://x/77")
	want := `pipeline=77 outcome=blocked gate="Deploy to prod" url=https://x/77`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	result = bitbucket.PipelineWatchResult{Pipeline: bitbucket.Pipeline{BuildNumber: 5}, Status: bitbucket.WatchSuccess}
	if got := pipelineOutcomeEvent(result, "https://x/5"); got != "pipeline=5 outcome=success url=https://x/5" {
		t.Errorf("got %q", got)
	}
}
