package bitbucket_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// statusesHandler serves PR statuses; perPoll returns the statuses for the nth
// request (1-based).
func statusesHandler(t *testing.T, perPoll func(n int32) []bitbucket.PRStatus) http.HandlerFunc {
	var polls atomic.Int32
	return func(w http.ResponseWriter, r *http.Request) {
		mustEncodeJSON(t, w, map[string]any{"values": perPoll(polls.Add(1))})
	}
}

func waitChecks(t *testing.T, h http.Handler, opts bitbucket.WaitChecksOptions) bitbucket.PRChecksWaitResult {
	t.Helper()
	if opts.Interval == 0 {
		opts.Interval = time.Millisecond
	}
	res, err := newTestClient(t, h).PRs("ws", "repo").WaitChecks(context.Background(), 33, opts)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestPRs_WaitChecks_SuccessAfterInProgress(t *testing.T) {
	h := statusesHandler(t, func(n int32) []bitbucket.PRStatus {
		if n < 3 {
			return []bitbucket.PRStatus{{Key: "build", State: "INPROGRESS"}, {Key: "lint", State: "SUCCESSFUL"}}
		}
		return []bitbucket.PRStatus{{Key: "build", State: "SUCCESSFUL"}, {Key: "lint", State: "SUCCESSFUL"}}
	})
	if got := waitChecks(t, h, bitbucket.WaitChecksOptions{}); got.Status != bitbucket.ChecksSuccess {
		t.Errorf("expected success, got %s", got.Status)
	}
}

func TestPRs_WaitChecks_FailedWaitsForAllTerminal(t *testing.T) {
	h := statusesHandler(t, func(n int32) []bitbucket.PRStatus {
		if n < 2 {
			return []bitbucket.PRStatus{{Key: "build", State: "FAILED"}, {Key: "lint", State: "INPROGRESS"}}
		}
		return []bitbucket.PRStatus{{Key: "build", State: "FAILED"}, {Key: "lint", State: "SUCCESSFUL"}}
	})
	got := waitChecks(t, h, bitbucket.WaitChecksOptions{})
	if got.Status != bitbucket.ChecksFailed {
		t.Errorf("expected failed, got %s", got.Status)
	}
	if got.Statuses[1].State != "SUCCESSFUL" {
		t.Errorf("expected to wait until lint finished, got %+v", got.Statuses)
	}
}

func TestPRs_WaitChecks_StoppedIsFailure(t *testing.T) {
	h := statusesHandler(t, func(int32) []bitbucket.PRStatus {
		return []bitbucket.PRStatus{{Key: "build", State: "STOPPED"}}
	})
	if got := waitChecks(t, h, bitbucket.WaitChecksOptions{}); got.Status != bitbucket.ChecksFailed {
		t.Errorf("expected failed, got %s", got.Status)
	}
}

func TestPRs_WaitChecks_NoneWhenNothingAppears(t *testing.T) {
	h := statusesHandler(t, func(int32) []bitbucket.PRStatus { return nil })
	got := waitChecks(t, h, bitbucket.WaitChecksOptions{AppearTimeout: 15 * time.Millisecond})
	if got.Status != bitbucket.ChecksNone {
		t.Errorf("expected none, got %s", got.Status)
	}
}

func TestPRs_WaitChecks_AppearsLate(t *testing.T) {
	h := statusesHandler(t, func(n int32) []bitbucket.PRStatus {
		if n < 3 {
			return nil
		}
		return []bitbucket.PRStatus{{Key: "build", State: "SUCCESSFUL"}}
	})
	got := waitChecks(t, h, bitbucket.WaitChecksOptions{AppearTimeout: time.Second})
	if got.Status != bitbucket.ChecksSuccess {
		t.Errorf("expected success, got %s", got.Status)
	}
}

func TestPRs_WaitChecks_Timeout(t *testing.T) {
	h := statusesHandler(t, func(int32) []bitbucket.PRStatus {
		return []bitbucket.PRStatus{{Key: "build", State: "INPROGRESS"}}
	})
	got := waitChecks(t, h, bitbucket.WaitChecksOptions{Timeout: 15 * time.Millisecond})
	if got.Status != bitbucket.ChecksTimeout {
		t.Errorf("expected timeout, got %s", got.Status)
	}
	if len(got.Statuses) != 1 {
		t.Errorf("expected last statuses on timeout, got %+v", got.Statuses)
	}
}
