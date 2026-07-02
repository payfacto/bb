package bitbucket

import (
	"context"
)

// PipelineVariableResource provides operations on repository-level pipeline variables.
type PipelineVariableResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *PipelineVariableResource) basePath() string {
	return repoPath(r.workspace, r.repo) + "/pipelines_config/variables/"
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

// Get returns a single pipeline variable by UUID. Secured variables are
// returned without their value (the API hides it).
func (r *PipelineVariableResource) Get(ctx context.Context, uuid string) (PipelineVariable, error) {
	data, err := r.client.do(ctx, "GET", r.basePath()+uuid, nil, nil)
	if err != nil {
		return PipelineVariable{}, err
	}
	return decode[PipelineVariable](data)
}

// Update replaces a pipeline variable by UUID with the given representation.
// The Bitbucket PUT requires the full {key,value,secured} body.
func (r *PipelineVariableResource) Update(ctx context.Context, uuid string, input CreatePipelineVariableInput) (PipelineVariable, error) {
	data, err := r.client.do(ctx, "PUT", r.basePath()+uuid, input, nil)
	if err != nil {
		return PipelineVariable{}, err
	}
	return decode[PipelineVariable](data)
}

// Delete removes a pipeline variable by UUID.
func (r *PipelineVariableResource) Delete(ctx context.Context, uuid string) error {
	_, err := r.client.do(ctx, "DELETE", r.basePath()+uuid, nil, nil)
	return err
}
