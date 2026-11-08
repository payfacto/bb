package cmd

import (
	"errors"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestScheduleInputFromFlags_defaultsSelectorToBranch(t *testing.T) {
	got := scheduleCreateFlags{branch: "main", cron: "0 0 12 * * ? *"}.input()
	if got.Target.RefType != "branch" || got.Target.RefName != "main" {
		t.Errorf("unexpected target: %+v", got.Target)
	}
	if got.Target.Selector.Type != "branches" || got.Target.Selector.Pattern != "main" {
		t.Errorf("expected branches:main selector, got %+v", got.Target.Selector)
	}
	if got.Enabled != nil {
		t.Errorf("expected enabled unset, got %v", *got.Enabled)
	}
	if got.CronPattern != "0 0 12 * * ? *" {
		t.Errorf("unexpected cron: %q", got.CronPattern)
	}
}

func TestScheduleInputFromFlags_customSelectorAndDisabled(t *testing.T) {
	got := scheduleCreateFlags{
		branch: "main", cron: "0 0 2 * * ? *",
		selectorType: "custom", pattern: "nightly", disabled: true,
	}.input()
	if got.Target.Selector.Type != "custom" || got.Target.Selector.Pattern != "nightly" {
		t.Errorf("unexpected selector: %+v", got.Target.Selector)
	}
	if got.Enabled == nil || *got.Enabled {
		t.Errorf("expected enabled=false, got %v", got.Enabled)
	}
}

func TestResolveScheduleEnabled(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		disabled bool
		want     bool
		wantErr  bool
	}{
		{name: "enable", enabled: true, want: true},
		{name: "disable", disabled: true, want: false},
		{name: "neither", wantErr: true},
		{name: "both", enabled: true, disabled: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveScheduleEnabled(tt.enabled, tt.disabled)
			if tt.wantErr {
				var cliErr *CLIError
				if !errors.As(err, &cliErr) || cliErr.Code != ErrCodeValidationFailed {
					t.Fatalf("expected validation_failed CLIError, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScheduleUpdateFromStdin_rejectsMissingEnabled(t *testing.T) {
	err := validateScheduleUpdate(bitbucket.UpdatePipelineScheduleInput{})
	var cliErr *CLIError
	if !errors.As(err, &cliErr) || cliErr.Code != ErrCodeValidationFailed {
		t.Fatalf("expected validation_failed, got %v", err)
	}
}

func TestScheduleUpdateFromStdin_acceptsExplicitFalse(t *testing.T) {
	disabled := false
	if err := validateScheduleUpdate(bitbucket.UpdatePipelineScheduleInput{Enabled: &disabled}); err != nil {
		t.Fatalf("expected explicit enabled=false to be accepted, got %v", err)
	}
}
