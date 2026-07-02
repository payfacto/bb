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

func TestResolveTriggerRef(t *testing.T) {
	tests := []struct {
		name           string
		branch         string
		tag            string
		commit         string
		wantErr        bool
		wantBranch     string
		wantTag        string
		wantCommitHash string
	}{
		{name: "branch", branch: "main", wantBranch: "main"},
		{name: "tag", tag: "v1.0", wantTag: "v1.0"},
		{name: "commit", commit: "abc123", wantCommitHash: "abc123"},
		{name: "none", wantErr: true},
		{name: "branch and tag", branch: "main", tag: "v1.0", wantErr: true},
		{name: "all three", branch: "main", tag: "v1.0", commit: "abc123", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := resolveTriggerRef(tt.branch, tt.tag, tt.commit)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var cliErr *CLIError
				if !errors.As(err, &cliErr) || cliErr.Code != ErrCodeValidationFailed {
					t.Errorf("expected validation_failed CLIError, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if ref.Branch != tt.wantBranch || ref.Tag != tt.wantTag || ref.Commit != tt.wantCommitHash {
				t.Errorf("ref = %+v, want branch=%q tag=%q commit=%q", ref, tt.wantBranch, tt.wantTag, tt.wantCommitHash)
			}
		})
	}
}

func TestTriggerRefLabel(t *testing.T) {
	tests := []struct {
		name   string
		ref    bitbucket.TriggerRef
		custom string
		want   string
	}{
		{name: "branch", ref: bitbucket.TriggerRef{Branch: "main"}, want: "branch main"},
		{name: "tag", ref: bitbucket.TriggerRef{Tag: "v1.0"}, want: "tag v1.0"},
		{name: "commit", ref: bitbucket.TriggerRef{Commit: "abc123"}, want: "commit abc123"},
		{name: "branch with custom", ref: bitbucket.TriggerRef{Branch: "main"}, custom: "deploy",
			want: "branch main (custom pipeline deploy)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := triggerRefLabel(tt.ref, tt.custom); got != tt.want {
				t.Errorf("triggerRefLabel = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseTriggerVars(t *testing.T) {
	t.Run("valid pairs, value may contain equals", func(t *testing.T) {
		got, err := parseTriggerVars([]string{"A=1", "B=x=y"})
		if err != nil {
			t.Fatal(err)
		}
		want := []bitbucket.TriggerVariable{{Key: "A", Value: "1"}, {Key: "B", Value: "x=y"}}
		if len(got) != len(want) {
			t.Fatalf("got %d vars, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("var[%d] = %+v, want %+v", i, got[i], want[i])
			}
		}
	})
	t.Run("empty slice returns nil", func(t *testing.T) {
		got, err := parseTriggerVars(nil)
		if err != nil || got != nil {
			t.Errorf("expected (nil, nil), got (%v, %v)", got, err)
		}
	})
	for _, bad := range []string{"noequals", "=novalue"} {
		t.Run("invalid "+bad, func(t *testing.T) {
			_, err := parseTriggerVars([]string{bad})
			var cliErr *CLIError
			if !errors.As(err, &cliErr) || cliErr.Code != ErrCodeValidationFailed {
				t.Errorf("expected validation_failed CLIError for %q, got %v", bad, err)
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
