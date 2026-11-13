package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const stopWebURL = "https://bitbucket.org/ws/repo/pipelines/results/7"

func TestStopOutcomeError_StillPausedIsConflictWithUILink(t *testing.T) {
	res := bitbucket.StopResult{Result: bitbucket.StopStillPaused, UUID: "{p}", BuildNumber: 7}
	err := stopOutcomeError(res, stopWebURL)
	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("err = %v, want *CLIError", err)
	}
	if cliErr.Code != ErrCodeConflict {
		t.Errorf("Code = %q, want %q", cliErr.Code, ErrCodeConflict)
	}
	if !strings.Contains(cliErr.Message, stopWebURL) {
		t.Errorf("Message %q does not link to %s", cliErr.Message, stopWebURL)
	}
	if cliErr.Details["result"] != bitbucket.StopStillPaused {
		t.Errorf("Details[result] = %v, want %q", cliErr.Details["result"], bitbucket.StopStillPaused)
	}
}

func TestStopOutcomeError_ConfirmedAndRequestedAreNotErrors(t *testing.T) {
	for _, outcome := range []bitbucket.StopOutcome{bitbucket.StopConfirmed, bitbucket.StopRequested} {
		if err := stopOutcomeError(bitbucket.StopResult{Result: outcome}, stopWebURL); err != nil {
			t.Errorf("%s: err = %v, want nil", outcome, err)
		}
	}
}
