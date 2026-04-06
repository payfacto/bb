package bitbucket

import (
	"context"
	"fmt"
)

// PipelineVariableResource provides operations on repository-level pipeline variables.
type PipelineVariableResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *PipelineVariableResource) basePath() string {
	return fmt.Sprintf("%s/pipelines_config/variables/", repoPath(r.workspace, r.repo))
}

// List returns all pipeline variables for the repository.
func (r *PipelineVariableResource) List(ctx context.Context) ([]PipelineVariable, error) {
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, nil)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[PipelineVariable]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Create adds a new pipeline variable to the repository.
func (r *PipelineVariableResource) Create(ctx context.Context, input CreatePipelineVariableInput) (PipelineVariable, error) {
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return PipelineVariable{}, err
	}
	return decode[PipelineVariable](data)
}

// Delete removes a pipeline variable by UUID.
func (r *PipelineVariableResource) Delete(ctx context.Context, uuid string) error {
	path := fmt.Sprintf("%s%s", r.basePath(), uuid)
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
