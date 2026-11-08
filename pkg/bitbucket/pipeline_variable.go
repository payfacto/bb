package bitbucket

import (
	"context"
	"net/url"
	"strings"
)

// PipelineVariableResource provides operations on pipeline variables. The same
// resource serves both repository-level and workspace-level variables; only
// the collection path differs (see Client.PipelineVariables and
// Client.WorkspaceVariables).
type PipelineVariableResource struct {
	client *Client
	// collectionPath is the list/create endpoint. It is kept verbatim (the
	// repository endpoint is historically addressed with a trailing slash).
	collectionPath string
}

func (r *PipelineVariableResource) itemPath(uuid string) string {
	return strings.TrimSuffix(r.collectionPath, "/") + "/" + url.PathEscape(uuid)
}

// List returns all pipeline variables in scope, following pagination.
func (r *PipelineVariableResource) List(ctx context.Context) ([]PipelineVariable, error) {
	return fetchAllPages[PipelineVariable](ctx, r.client, r.collectionPath, nil)
}

// Create adds a new pipeline variable.
func (r *PipelineVariableResource) Create(ctx context.Context, input CreatePipelineVariableInput) (PipelineVariable, error) {
	data, err := r.client.do(ctx, "POST", r.collectionPath, input, nil)
	if err != nil {
		return PipelineVariable{}, err
	}
	return decode[PipelineVariable](data)
}

// Get returns a single pipeline variable by UUID. Secured variables are
// returned without their value (the API hides it).
func (r *PipelineVariableResource) Get(ctx context.Context, uuid string) (PipelineVariable, error) {
	data, err := r.client.do(ctx, "GET", r.itemPath(uuid), nil, nil)
	if err != nil {
		return PipelineVariable{}, err
	}
	return decode[PipelineVariable](data)
}

// Update replaces a pipeline variable by UUID with the given representation.
// The Bitbucket PUT requires the full {key,value,secured} body.
func (r *PipelineVariableResource) Update(ctx context.Context, uuid string, input CreatePipelineVariableInput) (PipelineVariable, error) {
	data, err := r.client.do(ctx, "PUT", r.itemPath(uuid), input, nil)
	if err != nil {
		return PipelineVariable{}, err
	}
	return decode[PipelineVariable](data)
}

// Delete removes a pipeline variable by UUID.
func (r *PipelineVariableResource) Delete(ctx context.Context, uuid string) error {
	_, err := r.client.do(ctx, "DELETE", r.itemPath(uuid), nil, nil)
	return err
}
