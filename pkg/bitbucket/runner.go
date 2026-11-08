package bitbucket

import (
	"context"
	"net/url"
)

// RunnerResource provides read and delete operations on self-hosted Pipelines
// runners. The same resource serves repository and workspace runners; only
// the collection path differs (see Client.RepoRunners and
// Client.WorkspaceRunners). Creating a runner is deliberately not exposed:
// the API returns a one-time OAuth secret that a CLI should not surface.
type RunnerResource struct {
	client         *Client
	collectionPath string
}

func (r *RunnerResource) itemPath(uuid string) string {
	return r.collectionPath + "/" + url.PathEscape(uuid)
}

// List returns every runner in scope, following pagination.
func (r *RunnerResource) List(ctx context.Context) ([]PipelineRunner, error) {
	return fetchAllPages[PipelineRunner](ctx, r.client, r.collectionPath, nil)
}

// Get returns a single runner by UUID.
func (r *RunnerResource) Get(ctx context.Context, uuid string) (PipelineRunner, error) {
	data, err := r.client.do(ctx, "GET", r.itemPath(uuid), nil, nil)
	if err != nil {
		return PipelineRunner{}, err
	}
	return decode[PipelineRunner](data)
}

// Delete removes a runner by UUID.
func (r *RunnerResource) Delete(ctx context.Context, uuid string) error {
	_, err := r.client.do(ctx, "DELETE", r.itemPath(uuid), nil, nil)
	return err
}
