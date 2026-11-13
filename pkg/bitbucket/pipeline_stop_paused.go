package bitbucket

import (
	"context"
	"errors"
)

// SupersededScanPages bounds FindSupersededPaused's newest-first scan:
// 10 pages of 25 is the newest 250 pipelines.
const SupersededScanPages = 10

// SupersededPaused is what FindSupersededPaused found. Kept is the newest
// pipeline that has not completed (running, paused or queued); nil when every
// scanned pipeline has completed. KeptPaused is the newest pipeline paused at a
// gate - the one ready to deploy - when it is not Kept itself; it is never a
// target either. Targets are the older pipelines paused at a manual or system
// gate, newest first. Truncated means older pipelines exist beyond the scanned
// pages.
type SupersededPaused struct {
	Kept       *Pipeline  `json:"kept"`
	KeptPaused *Pipeline  `json:"kept_paused,omitempty"`
	Targets    []Pipeline `json:"targets"`
	Scanned    int        `json:"scanned"`
	Truncated  bool       `json:"truncated"`
}

// FindSupersededPaused scans the newest maxPages pages of pipelines across the
// whole repository (every branch) and returns the newest active pipeline, the
// newest paused pipeline, and every older pipeline paused at a gate. Older
// pipelines that are still running and completed pipelines are never targets.
func (r *PipelineResource) FindSupersededPaused(ctx context.Context, maxPages int) (SupersededPaused, error) {
	found := SupersededPaused{Targets: []Pipeline{}}
	newestPausedSeen := false
	truncated, err := r.scanNewest(ctx, maxPages, func(p Pipeline) bool {
		found.Scanned++
		gated := isGated(p.State)
		switch {
		case found.Kept == nil && p.State.Name != stateCompleted:
			kept := p
			found.Kept = &kept
		case gated && !newestPausedSeen:
			keptPaused := p
			found.KeptPaused = &keptPaused
		case gated:
			found.Targets = append(found.Targets, p)
		}
		newestPausedSeen = newestPausedSeen || gated
		return true
	})
	if err != nil {
		return SupersededPaused{}, err
	}
	found.Truncated = truncated
	return found, nil
}

// StopIssue names a pipeline StopPaused did not stop, and why.
type StopIssue struct {
	BuildNumber int    `json:"build_number"`
	UUID        string `json:"uuid"`
	Reason      string `json:"reason"`
}

// StopPausedBatch sorts StopPaused's targets by outcome. Skipped pipelines
// changed state after the scan (resumed or finished) and got no stop request;
// Failed pipelines errored or were still paused when confirmation ended.
type StopPausedBatch struct {
	Stopped []StopResult `json:"stopped"`
	Skipped []StopIssue  `json:"skipped"`
	Failed  []StopIssue  `json:"failed"`
}

func newStopPausedBatch() StopPausedBatch {
	return StopPausedBatch{Stopped: []StopResult{}, Skipped: []StopIssue{}, Failed: []StopIssue{}}
}

// HasFailures reports whether any target failed to stop.
func (b StopPausedBatch) HasFailures() bool { return len(b.Failed) > 0 }

// StopPaused stops each target in turn with StopAndConfirm, requiring it to
// still be paused at a gate when re-read. One target's failure never stops the
// rest; every target ends up in exactly one of the batch's lists.
func (r *PipelineResource) StopPaused(ctx context.Context, targets []Pipeline, opts StopOptions) StopPausedBatch {
	opts.RequireGated = true
	batch := newStopPausedBatch()
	for _, target := range targets {
		issue := StopIssue{BuildNumber: target.BuildNumber, UUID: target.UUID}
		if err := ctx.Err(); err != nil {
			issue.Reason = "not attempted: " + err.Error()
			batch.Failed = append(batch.Failed, issue)
			continue
		}
		result, err := r.StopAndConfirm(ctx, target.UUID, opts)
		switch {
		case errors.Is(err, ErrPipelineNotPaused), errors.Is(err, ErrPipelineAlreadyCompleted):
			issue.Reason = err.Error()
			batch.Skipped = append(batch.Skipped, issue)
		case err != nil:
			issue.Reason = err.Error()
			batch.Failed = append(batch.Failed, issue)
		case result.Result == StopStillPaused:
			issue.Reason = "still paused after the stop request; close it in the UI: " + r.WebURL(target.BuildNumber)
			batch.Failed = append(batch.Failed, issue)
		default:
			batch.Stopped = append(batch.Stopped, result)
		}
	}
	return batch
}

// StopPausedReport is the full result of `bb pipeline stop-paused`: what the
// scan found and, unless DryRun, what happened to each target. On a dry run
// the batch lists are empty.
type StopPausedReport struct {
	SupersededPaused
	StopPausedBatch
	DryRun bool `json:"dry_run"`
}

// NewStopPausedReport wraps a scan result in a report with empty (not null)
// batch lists, ready for StopPaused's outcome or for a dry run as it is.
func NewStopPausedReport(found SupersededPaused) StopPausedReport {
	if found.Targets == nil {
		found.Targets = []Pipeline{}
	}
	return StopPausedReport{SupersededPaused: found, StopPausedBatch: newStopPausedBatch()}
}
