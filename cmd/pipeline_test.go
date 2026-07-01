package cmd

import (
	"errors"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// pipelineSelector.validate is pure branching logic (no I/O), so it is unit
// tested here even though the surrounding Cobra RunE wiring is not. It backs
// the "exactly one of --pipeline-uuid / --build-number" contract for the
// pipeline get/stop/steps/log commands.
func TestPipelineSelectorValidate(t *testing.T) {
	tests := []struct {
		name    string
		uuid    string
		build   int
		wantErr bool
	}{
		{name: "uuid only", uuid: "{abc-123}", build: 0, wantErr: false},
		{name: "build only", uuid: "", build: 42, wantErr: false},
		{name: "neither", uuid: "", build: 0, wantErr: true},
		{name: "both", uuid: "{abc-123}", build: 42, wantErr: true},
		{name: "negative build treated as unset", uuid: "", build: -1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pipelineSelector{uuid: tt.uuid, build: tt.build}.validate()
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			var cliErr *CLIError
			if !errors.As(err, &cliErr) {
				t.Fatalf("expected *CLIError, got %T", err)
			}
			if cliErr.Code != ErrCodeValidationFailed {
				t.Errorf("expected code %q, got %q", ErrCodeValidationFailed, cliErr.Code)
			}
		})
	}
}

func TestRunningStep(t *testing.T) {
	steps := []bitbucket.PipelineStep{
		{Name: "build", State: bitbucket.PipelineState{Name: "COMPLETED"}},
		{Name: "test", State: bitbucket.PipelineState{Name: "IN_PROGRESS"}},
		{Name: "deploy", State: bitbucket.PipelineState{Name: "PENDING"}},
	}
	got := runningStep(steps)
	if got == nil || got.Name != "test" {
		t.Fatalf("expected running step 'test', got %+v", got)
	}
	if runningStep(nil) != nil {
		t.Error("expected nil for no steps")
	}
	done := []bitbucket.PipelineStep{{Name: "build", State: bitbucket.PipelineState{Name: "COMPLETED"}}}
	if runningStep(done) != nil {
		t.Error("expected nil when no step is running")
	}
}

func TestWatchExitCode(t *testing.T) {
	cases := map[bitbucket.PipelineWatchStatus]int{
		bitbucket.WatchSuccess: 0,
		bitbucket.WatchFailed:  1,
		bitbucket.WatchBlocked: 2,
		bitbucket.WatchTimeout: 3,
	}
	for status, want := range cases {
		if got := watchExitCode(status); got != want {
			t.Errorf("watchExitCode(%q) = %d, want %d", status, got, want)
		}
	}
	if got := watchExitCode(bitbucket.PipelineWatchStatus("unknown")); got != 0 {
		t.Errorf("watchExitCode(unknown) = %d, want 0", got)
	}
}

func TestValidateWatchSelector(t *testing.T) {
	tests := []struct {
		name    string
		sel     pipelineSelector
		wantErr bool
	}{
		{name: "none selects latest", sel: pipelineSelector{}, wantErr: false},
		{name: "uuid only", sel: pipelineSelector{uuid: "{u}"}, wantErr: false},
		{name: "build only", sel: pipelineSelector{build: 5}, wantErr: false},
		{name: "branch only", sel: pipelineSelector{branch: "main"}, wantErr: false},
		{name: "uuid and build", sel: pipelineSelector{uuid: "{u}", build: 5}, wantErr: true},
		{name: "build and branch", sel: pipelineSelector{build: 5, branch: "main"}, wantErr: true},
		{name: "all three", sel: pipelineSelector{uuid: "{u}", build: 5, branch: "main"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sel.validateWatchSelector()
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var cliErr *CLIError
			if !errors.As(err, &cliErr) || cliErr.Code != ErrCodeValidationFailed {
				t.Errorf("expected validation_failed CLIError, got %v", err)
			}
		})
	}
}
