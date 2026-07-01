package bitbucket

import "testing"

// classifyPipelineState is unexported pure logic, so it is tested from inside
// the package. It backs the terminal-state decisions of PipelineResource.Watch.
func TestClassifyPipelineState(t *testing.T) {
	completed := func(result string) PipelineState {
		st := PipelineState{Name: "COMPLETED"}
		if result != "" {
			st.Result = &PipelineResult{Name: result}
		}
		return st
	}
	inProgress := func(stage string) PipelineState {
		st := PipelineState{Name: "IN_PROGRESS"}
		if stage != "" {
			st.Stage = &PipelineStage{Name: stage}
		}
		return st
	}

	tests := []struct {
		name         string
		state        PipelineState
		steps        []PipelineStep
		wantStatus   PipelineWatchStatus
		wantGateStep string
		wantTerminal bool
	}{
		{name: "completed successful", state: completed("SUCCESSFUL"), wantStatus: WatchSuccess, wantTerminal: true},
		{name: "completed failed", state: completed("FAILED"), wantStatus: WatchFailed, wantTerminal: true},
		{name: "completed error", state: completed("ERROR"), wantStatus: WatchFailed, wantTerminal: true},
		{name: "completed stopped", state: completed("STOPPED"), wantStatus: WatchFailed, wantTerminal: true},
		{name: "completed nil result treated as failed", state: completed(""), wantStatus: WatchFailed, wantTerminal: true},
		{
			name:  "in progress paused is a blocked manual gate",
			state: inProgress("PAUSED"),
			steps: []PipelineStep{
				{Name: "build", State: PipelineState{Name: "COMPLETED"}},
				{Name: "deploy", State: PipelineState{Name: "PENDING"}},
			},
			wantStatus:   WatchBlocked,
			wantGateStep: "deploy",
			wantTerminal: true,
		},
		{name: "in progress halted is blocked", state: inProgress("HALTED"), wantStatus: WatchBlocked, wantTerminal: true},
		{name: "in progress running is not terminal", state: inProgress("RUNNING"), wantTerminal: false},
		{name: "in progress no stage is not terminal", state: inProgress(""), wantTerminal: false},
		{name: "pending is not terminal", state: PipelineState{Name: "PENDING"}, wantTerminal: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pipeline{State: tt.state}
			status, gateStep, terminal := classifyPipelineState(p, tt.steps)
			if terminal != tt.wantTerminal {
				t.Fatalf("terminal = %v, want %v", terminal, tt.wantTerminal)
			}
			if !tt.wantTerminal {
				return
			}
			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
			if gateStep != tt.wantGateStep {
				t.Errorf("gateStep = %q, want %q", gateStep, tt.wantGateStep)
			}
		})
	}
}
