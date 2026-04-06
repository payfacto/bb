package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// ProjectResource provides operations on workspace projects.
type ProjectResource struct {
	client    *Client
	workspace string
}

func (r *ProjectResource) basePath() string {
	return fmt.Sprintf("/workspaces/%s/projects", r.workspace)
}

// List returns all projects in the workspace.
func (r *ProjectResource) List(ctx context.Context) ([]Project, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	return fetchAllPages[Project](ctx, r.client, r.basePath(), q)
}

// Get returns a single project by its key (e.g. "PROJ").
func (r *ProjectResource) Get(ctx context.Context, projectKey string) (Project, error) {
	path := fmt.Sprintf("%s/%s", r.basePath(), projectKey)
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Project{}, err
	}
	return decode[Project](data)
}
