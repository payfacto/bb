package cmd

import (
	"errors"
	"testing"
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
