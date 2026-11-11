package bitbucket

import (
	"context"
	"errors"
	"time"
)

// Build status states reported on a pull request's source commit.
const (
	statusInProgress = "INPROGRESS"
	statusSuccessful = "SUCCESSFUL"
)

// ChecksWaitStatus is the outcome of waiting on a pull request's build checks.
type ChecksWaitStatus string

const (
	ChecksSuccess ChecksWaitStatus = "success" // every status is SUCCESSFUL
	ChecksFailed  ChecksWaitStatus = "failed"  // all terminal, at least one not SUCCESSFUL
	ChecksTimeout ChecksWaitStatus = "timeout" // Timeout elapsed first
	ChecksNone    ChecksWaitStatus = "none"    // no status appeared within AppearTimeout
)

// WaitChecksOptions configures WaitChecks. Timeout <= 0 waits indefinitely.
// AppearTimeout > 0 bounds how long to wait for the first status to appear (a
// repository without PR pipelines never reports one); <= 0 waits for statuses
// as long as Timeout allows. OnPoll, when set, sees each poll's statuses.
type WaitChecksOptions struct {
	Interval      time.Duration
	Timeout       time.Duration
	AppearTimeout time.Duration
	OnPoll        func([]PRStatus)
}

// PRChecksWaitResult is what `pr wait-checks` reports when it stops waiting.
type PRChecksWaitResult struct {
	Status   ChecksWaitStatus `json:"status"`
	Statuses []PRStatus       `json:"statuses"`
}

// WaitChecks polls the pull request's build statuses until every one is
// terminal (no INPROGRESS left), no status has appeared within AppearTimeout,
// or Timeout elapses. A timeout is an outcome (ChecksTimeout, nil error); an
// external cancellation such as Ctrl-C is returned as the error.
func (r *PRResource) WaitChecks(ctx context.Context, prID int, opts WaitChecksOptions) (PRChecksWaitResult, error) {
	ctx, cancel := withOptionalTimeout(ctx, opts.Timeout)
	defer cancel()
	start := time.Now()
	var result PRChecksWaitResult
	err := pollUntil(ctx, opts.Interval, func(ctx context.Context) (bool, error) {
		statuses, err := r.Statuses(ctx, prID)
		if err != nil {
			return false, err
		}
		if opts.OnPoll != nil {
			opts.OnPoll(statuses)
		}
		result = PRChecksWaitResult{Statuses: statuses}
		if len(statuses) == 0 {
			if opts.AppearTimeout > 0 && time.Since(start) >= opts.AppearTimeout {
				result.Status = ChecksNone
				return true, nil
			}
			return false, nil
		}
		status, terminal := classifyChecks(statuses)
		result.Status = status
		return terminal, nil
	})
	if errors.Is(err, context.DeadlineExceeded) {
		result.Status = ChecksTimeout
		return result, nil
	}
	if err != nil {
		return PRChecksWaitResult{}, err
	}
	return result, nil
}

// classifyChecks reports whether a non-empty status set is terminal and, if
// so, whether it passed. Any INPROGRESS status means still running.
func classifyChecks(statuses []PRStatus) (ChecksWaitStatus, bool) {
	outcome := ChecksSuccess
	for _, s := range statuses {
		switch s.State {
		case statusInProgress:
			return "", false
		case statusSuccessful:
		default:
			outcome = ChecksFailed
		}
	}
	return outcome, true
}
