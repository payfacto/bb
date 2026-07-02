package bitbucket

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Pipeline and step state/stage names as reported by the Bitbucket Pipelines
// API. Hoisted to consts so the state vocabulary is defined once and shared by
// classifyPipelineState, firstIncompleteStep, and the cmd-layer step helpers,
// preventing typos and drift across call sites.
const (
	StateInProgress  = "IN_PROGRESS"
	stateCompleted   = "COMPLETED"
	resultSuccessful = "SUCCESSFUL"
	stagePaused      = "PAUSED"
	stageHalted      = "HALTED"
)

// ErrNoPipelines is returned by Latest when no pipeline matches the request.
// It is a domain condition (not an HTTP failure); cmd/errors.go maps it to the
// not_found CLI error code so it renders identically to a real 404.
var ErrNoPipelines = errors.New("no pipelines found")

// PipelineResource provides operations on repository pipelines.
type PipelineResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *PipelineResource) basePath() string {
	return fmt.Sprintf("%s/pipelines/", repoPath(r.workspace, r.repo))
}

// List returns pipelines for the repository. sort is the Bitbucket field
// name to sort by, optionally prefixed with "-" for descending order
// (e.g. "-created_on"). An empty string defaults to "-created_on".
func (r *PipelineResource) List(ctx context.Context, sort string) ([]Pipeline, error) {
	if sort == "" {
		sort = "-created_on"
	}
	q := url.Values{"sort": {sort}, "pagelen": {pagelenSmall}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Pipeline]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Get returns a single pipeline by UUID.
func (r *PipelineResource) Get(ctx context.Context, pipelineUUID string) (Pipeline, error) {
	path := fmt.Sprintf("%s%s", r.basePath(), url.PathEscape(pipelineUUID))
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Pipeline{}, err
	}
	return decode[Pipeline](data)
}

// GetByBuildNumber returns a single pipeline addressed by its integer build
// number - the "#42" agents already see in `pipeline list` output - rather
// than its UUID. Bitbucket Cloud's pipeline resource accepts either form:
//
//	GET /repositories/{ws}/{repo}/pipelines/{build_number}
//
// The build number is a plain integer path segment; unlike the UUID form it
// carries no surrounding braces.
func (r *PipelineResource) GetByBuildNumber(ctx context.Context, buildNumber int) (Pipeline, error) {
	if buildNumber <= 0 {
		return Pipeline{}, fmt.Errorf("build number must be positive, got %d", buildNumber)
	}
	path := fmt.Sprintf("%s%d", r.basePath(), buildNumber)
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Pipeline{}, err
	}
	return decode[Pipeline](data)
}

// classifyPipelineState returns the terminal watch status for a pipeline and
// whether it has reached a terminal state. gateStep names the step the pipeline
// is blocked on when the status is blocked (best-effort; may be "").
//
// A COMPLETED pipeline is success only when its result is SUCCESSFUL; any other
// result (FAILED, ERROR, STOPPED, ...) or a missing result is a failure. An
// IN_PROGRESS pipeline whose stage is PAUSED (a manual/workflow gate) or HALTED
// (a system gate) is blocked - it will not advance without manual intervention.
// Every other state (PENDING, IN_PROGRESS/RUNNING) is not yet terminal.
func classifyPipelineState(p Pipeline, steps []PipelineStep) (status PipelineWatchStatus, gateStep string, terminal bool) {
	switch p.State.Name {
	case stateCompleted:
		if p.State.Result != nil && p.State.Result.Name == resultSuccessful {
			return WatchSuccess, "", true
		}
		return WatchFailed, "", true
	case StateInProgress:
		if p.State.Stage != nil {
			switch p.State.Stage.Name {
			case stagePaused, stageHalted:
				return WatchBlocked, firstIncompleteStep(steps), true
			}
		}
	}
	return "", "", false
}

// firstIncompleteStep returns the name of the first step not yet COMPLETED, or
// "" when every step is complete or there are none.
func firstIncompleteStep(steps []PipelineStep) string {
	for _, s := range steps {
		if s.State.Name != stateCompleted {
			return s.Name
		}
	}
	return ""
}

// Latest returns the most recent pipeline in the repository, or the most recent
// on branch when branch != "". It scans the -created_on (newest-first) list one
// page at a time, following the "next" pagination link, and returns the first
// pipeline whose target ref matches branch - so an infrequently-built branch is
// found even when its newest run is older than the first page of repo-wide runs.
// The scan stops early at the first match. When branch == "" the newest overall
// pipeline (first item of the first page) is returned without a full scan.
// It returns ErrNoPipelines (wrapped with branch context) when nothing matches.
func (r *PipelineResource) Latest(ctx context.Context, branch string) (Pipeline, error) {
	q := url.Values{"sort": {"-created_on"}, "pagelen": {pagelenSmall}}
	nextURL := ""
	for {
		var data []byte
		var err error
		if nextURL != "" {
			data, err = r.client.fetchPage(ctx, nextURL)
		} else {
			data, err = r.client.do(ctx, "GET", r.basePath(), nil, q)
		}
		if err != nil {
			return Pipeline{}, err
		}
		page, err := decode[paged[Pipeline]](data)
		if err != nil {
			return Pipeline{}, err
		}
		for _, p := range page.Values {
			if branch == "" || p.Target.RefName == branch {
				return p, nil
			}
		}
		if page.Next == "" {
			break
		}
		nextURL = page.Next
	}
	if branch != "" {
		return Pipeline{}, fmt.Errorf("%w for branch %q", ErrNoPipelines, branch)
	}
	return Pipeline{}, ErrNoPipelines
}

// WatchOptions configures Watch. Interval defaults to 5s when <= 0. Timeout <= 0
// means watch indefinitely (until a terminal or blocked state). OnPoll, when
// set, is invoked with each poll's pipeline and steps before the stop check.
type WatchOptions struct {
	Interval time.Duration
	Timeout  time.Duration
	OnPoll   func(Pipeline, []PipelineStep)
}

// Watch polls a pipeline until it reaches a terminal state (completed or blocked
// on a manual gate) or the timeout elapses, then returns the classified result.
// A blocked result carries the ManualGate with the web URL to resume it.
//
// When opts.Timeout > 0 a context.WithTimeout child bounds both the poll loop
// and each in-flight request, so the wall-clock return is tightly bounded rather
// than overrunning by up to interval + two round-trips. A fired deadline
// (context.DeadlineExceeded) returns the WatchTimeout status with a nil error;
// an external cancellation (context.Canceled, e.g. Ctrl-C) returns that error so
// the caller can map it to the interrupt path.
func (r *PipelineResource) Watch(ctx context.Context, pipelineUUID string, opts WatchOptions) (PipelineWatchResult, error) {
	interval := opts.Interval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}
	// A single ticker drives the poll cadence for the whole watch, rather than a
	// fresh time.After per iteration (which leaks a timer until it fires).
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var last PipelineWatchResult
	for {
		p, err := r.Get(ctx, pipelineUUID)
		if err != nil {
			return r.watchCtxResult(ctx, last, err)
		}
		steps, err := r.Steps(ctx, pipelineUUID)
		if err != nil {
			return r.watchCtxResult(ctx, last, err)
		}
		if opts.OnPoll != nil {
			opts.OnPoll(p, steps)
		}
		last = PipelineWatchResult{Pipeline: p, Steps: steps}
		if status, gateStep, terminal := classifyPipelineState(p, steps); terminal {
			result := PipelineWatchResult{Pipeline: p, Steps: steps, Status: status}
			if status == WatchBlocked {
				result.ManualGate = &ManualGate{Step: gateStep, URL: r.pipelineWebURL(p.BuildNumber)}
			}
			return result, nil
		}
		select {
		case <-ctx.Done():
			return r.watchCtxResult(ctx, last, ctx.Err())
		case <-ticker.C:
		}
	}
}

// watchCtxResult classifies a context termination during Watch: a fired deadline
// is the timeout outcome (WatchTimeout status, nil error) while any other
// cancellation is surfaced as the error so the caller maps it to the interrupt
// path. Non-context errors (and the no-error case) are returned unchanged.
func (r *PipelineResource) watchCtxResult(ctx context.Context, last PipelineWatchResult, err error) (PipelineWatchResult, error) {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		last.Status = WatchTimeout
		return last, nil
	}
	return PipelineWatchResult{}, err
}

// pipelineWebURL builds the Bitbucket web URL for a pipeline's result page.
// Workspace/repo are path-escaped for consistency with the API path builders.
func (r *PipelineResource) pipelineWebURL(buildNumber int) string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s/pipelines/results/%d",
		url.PathEscape(r.workspace), url.PathEscape(r.repo), buildNumber)
}

// Trigger starts a new pipeline. opts.Ref selects the target (exactly one of
// Branch/Tag/Commit; the cmd layer validates this). opts.Custom runs a named
// custom pipeline; opts.Variables are per-run pipeline variables.
func (r *PipelineResource) Trigger(ctx context.Context, opts TriggerOptions) (Pipeline, error) {
	target, err := buildTriggerTarget(opts)
	if err != nil {
		return Pipeline{}, err
	}
	input := TriggerPipelineInput{Target: target, Variables: opts.Variables}
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Pipeline{}, err
	}
	return decode[Pipeline](data)
}

// buildTriggerTarget maps a TriggerOptions to the API target body. It expects
// exactly one ref to be set (guaranteed by the cmd layer) and errors otherwise
// so a misuse is not silently sent as an empty target.
func buildTriggerTarget(opts TriggerOptions) (TriggerTarget, error) {
	var t TriggerTarget
	switch {
	case opts.Ref.Commit != "":
		t.Type = "pipeline_commit_target"
		t.Commit = &TriggerCommit{Type: "commit", Hash: opts.Ref.Commit}
	case opts.Ref.Branch != "":
		t.Type = "pipeline_ref_target"
		t.RefType = "branch"
		t.RefName = opts.Ref.Branch
	case opts.Ref.Tag != "":
		t.Type = "pipeline_ref_target"
		t.RefType = "tag"
		t.RefName = opts.Ref.Tag
	default:
		return TriggerTarget{}, fmt.Errorf("trigger: no ref selected (branch, tag, or commit required)")
	}
	if opts.Custom != "" {
		t.Selector = &TriggerSelector{Type: "custom", Pattern: opts.Custom}
	}
	return t, nil
}

// Stop requests cancellation of a running pipeline.
func (r *PipelineResource) Stop(ctx context.Context, pipelineUUID string) error {
	path := fmt.Sprintf("%s%s/stopPipeline", r.basePath(), url.PathEscape(pipelineUUID))
	_, err := r.client.do(ctx, "POST", path, nil, nil)
	return err
}

// Steps returns all steps of a pipeline.
func (r *PipelineResource) Steps(ctx context.Context, pipelineUUID string) ([]PipelineStep, error) {
	path := fmt.Sprintf("%s%s/steps/", r.basePath(), url.PathEscape(pipelineUUID))
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[PipelineStep]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Log returns the raw log output for a pipeline step (plain text, not JSON).
//
// Bitbucket Cloud's pipeline endpoints require the UUIDs to be passed with
// their surrounding curly braces (URL-encoded as %7B...%7D). Earlier versions
// stripped the braces, which produced 404s against the real API. Pass UUIDs
// through verbatim — `bb pipeline steps` already returns them braced.
func (r *PipelineResource) Log(ctx context.Context, pipelineUUID, stepUUID string) (string, error) {
	path := fmt.Sprintf("%s%s/steps/%s/log",
		r.basePath(), url.PathEscape(pipelineUUID), url.PathEscape(stepUUID))
	data, err := r.client.doText(ctx, path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
