package bitbucket

import (
	"context"
)

// PipelineConfigResource provides operations on the repository's pipelines_config
// resource - currently limited to reading and toggling whether Pipelines is
// enabled for the repository.
type PipelineConfigResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *PipelineConfigResource) path() string {
	return repoPath(r.workspace, r.repo) + "/pipelines_config"
}

// Get returns the repository's pipelines configuration.
func (r *PipelineConfigResource) Get(ctx context.Context) (PipelineConfig, error) {
	data, err := r.client.do(ctx, "GET", r.path(), nil, nil)
	if err != nil {
		return PipelineConfig{}, err
	}
	return decode[PipelineConfig](data)
}

// Enable turns on Pipelines for the repository.
func (r *PipelineConfigResource) Enable(ctx context.Context) (PipelineConfig, error) {
	return r.setEnabled(ctx, true)
}

// Disable turns off Pipelines for the repository.
func (r *PipelineConfigResource) Disable(ctx context.Context) (PipelineConfig, error) {
	return r.setEnabled(ctx, false)
}

func (r *PipelineConfigResource) setEnabled(ctx context.Context, enabled bool) (PipelineConfig, error) {
	data, err := r.client.do(ctx, "PUT", r.path(), PipelineConfig{Enabled: enabled}, nil)
	if err != nil {
		return PipelineConfig{}, err
	}
	return decode[PipelineConfig](data)
}
