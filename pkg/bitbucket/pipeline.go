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

// List returns the most recent pipelines, newest first.
func (r *PipelineResource) List(ctx context.Context) ([]Pipeline, error) {
	q := url.Values{"sort": {"-created_on"}, "pagelen": {pagelenSmall}}
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
func (r *PipelineResource) Log(ctx context.Context, pipelineUUID, stepUUID string) (string, error) {
	path := fmt.Sprintf("%s%s/steps/%s/log",
		r.basePath(), url.PathEscape(pipelineUUID), url.PathEscape(stepUUID))
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
