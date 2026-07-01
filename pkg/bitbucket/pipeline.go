package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

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
	case "COMPLETED":
		if p.State.Result != nil && p.State.Result.Name == "SUCCESSFUL" {
			return WatchSuccess, "", true
		}
		return WatchFailed, "", true
	case "IN_PROGRESS":
		if p.State.Stage != nil {
			switch p.State.Stage.Name {
			case "PAUSED", "HALTED":
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
		if s.State.Name != "COMPLETED" {
			return s.Name
		}
	}
	return ""
}

// Trigger starts a new pipeline on the given branch.
func (r *PipelineResource) Trigger(ctx context.Context, branch string) (Pipeline, error) {
	input := TriggerPipelineInput{
		Target: TriggerTarget{
			RefType: "branch",
			Type:    "pipeline_ref_target",
			RefName: branch,
		},
	}
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Pipeline{}, err
	}
	return decode[Pipeline](data)
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
