package bitbucket

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// StopOutcome is what StopAndConfirm observed after asking Bitbucket to stop a
// pipeline. Bitbucket answers the stop request with 204 before anything has
// actually stopped, so the outcome is read back from the pipeline's state.
type StopOutcome string

const (
	// StopConfirmed: the pipeline reached COMPLETED (normally result STOPPED).
	StopConfirmed StopOutcome = "stopped"
	// StopRequested: the stop was accepted but the pipeline had not completed
	// when the confirmation window ended (or confirmation was skipped).
	StopRequested StopOutcome = "stop_requested"
	// StopStillPaused: the pipeline is still paused on a manual or system gate,
	// so the stop request had no visible effect.
	StopStillPaused StopOutcome = "still_paused"
)

// ErrPipelineAlreadyCompleted is returned by StopAndConfirm, without sending a
// stop request, when the pipeline has already finished.
var ErrPipelineAlreadyCompleted = errors.New("pipeline already completed")

// ErrPipelineNotPaused is returned by StopAndConfirm, without sending a stop
// request, when StopOptions.RequireGated is set and the pipeline is no longer
// held at a manual or system gate (for example someone resumed it).
var ErrPipelineNotPaused = errors.New("pipeline is no longer paused")

// StopOptions bounds the confirmation poll. A zero Timeout skips confirmation
// and reports StopRequested straight after the stop request is accepted.
// RequireGated refuses to stop a pipeline that is not paused or halted when it
// is re-read just before the stop request.
type StopOptions struct {
	Interval     time.Duration
	Timeout      time.Duration
	RequireGated bool
}

// StopResult is the outcome of StopAndConfirm and the last state it saw.
type StopResult struct {
	Result      StopOutcome   `json:"result"`
	UUID        string        `json:"uuid"`
	BuildNumber int           `json:"build_number"`
	State       PipelineState `json:"state"`
}

// StopAndConfirm stops a pipeline and re-reads its state until it completes or
// opts.Timeout passes, so callers never report a stop that did not happen. A
// pipeline that has already completed is refused with
// ErrPipelineAlreadyCompleted (wrapped) and no stop request is sent.
func (r *PipelineResource) StopAndConfirm(ctx context.Context, pipelineUUID string, opts StopOptions) (StopResult, error) {
	before, err := r.Get(ctx, pipelineUUID)
	if err != nil {
		return StopResult{}, err
	}
	if before.State.Name == stateCompleted {
		return StopResult{UUID: pipelineUUID, BuildNumber: before.BuildNumber, State: before.State},
			fmt.Errorf("%w (result %s)", ErrPipelineAlreadyCompleted, resultName(before.State))
	}
	if opts.RequireGated && !isGated(before.State) {
		return StopResult{UUID: pipelineUUID, BuildNumber: before.BuildNumber, State: before.State},
			fmt.Errorf("%w (state %s)", ErrPipelineNotPaused, before.State.Name)
	}
	if err := r.Stop(ctx, pipelineUUID); err != nil {
		return StopResult{}, err
	}
	requested := StopResult{Result: StopRequested, UUID: pipelineUUID, BuildNumber: before.BuildNumber, State: before.State}
	if opts.Timeout <= 0 {
		return requested, nil
	}
	return r.awaitStopped(ctx, requested, opts)
}

// awaitStopped re-reads the pipeline until it completes or opts.Timeout passes
// and names the outcome. Read errors are retried rather than returned: the stop
// is already sent, and Bitbucket has been seen answering pipeline reads with
// intermittent 404s. Only cancellation of ctx itself is returned as an error.
func (r *PipelineResource) awaitStopped(ctx context.Context, requested StopResult, opts StopOptions) (StopResult, error) {
	pollCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	var observed *PipelineState // last state read after the stop; nil until one read succeeds
	err := pollUntil(pollCtx, opts.Interval, func(ctx context.Context) (bool, error) {
		p, err := r.Get(ctx, requested.UUID)
		if err != nil {
			return false, nil
		}
		observed = &p.State
		return p.State.Name == stateCompleted, nil
	})
	result := requested
	if observed != nil {
		result.State = *observed
	}
	switch {
	case err == nil:
		result.Result = StopConfirmed
	case errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil:
		result.Result = unconfirmedStopOutcome(observed)
	default:
		return result, err
	}
	return result, nil
}

// unconfirmedStopOutcome names a stop whose pipeline never completed: still
// gated (the stop had no visible effect) or still winding down. Gated is only
// claimed when a read after the stop proved it; with no such read (observed is
// nil) the outcome stays StopRequested.
func unconfirmedStopOutcome(observed *PipelineState) StopOutcome {
	if observed != nil && isGated(*observed) {
		return StopStillPaused
	}
	return StopRequested
}

// isGated reports an IN_PROGRESS pipeline held at a manual (PAUSED) or system
// (HALTED) gate.
func isGated(state PipelineState) bool {
	if state.Name != StateInProgress || state.Stage == nil {
		return false
	}
	return state.Stage.Name == stagePaused || state.Stage.Name == stageHalted
}

// resultName is the state's result name, or "unknown" when Bitbucket sent none.
func resultName(state PipelineState) string {
	if state.Result == nil {
		return "unknown"
	}
	return state.Result.Name
}
