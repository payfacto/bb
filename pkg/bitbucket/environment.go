package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// EnvironmentResource provides operations on repository deployment environments.
type EnvironmentResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *EnvironmentResource) basePath() string {
	return fmt.Sprintf("%s/environments/", repoPath(r.workspace, r.repo))
}

// Get returns a single deployment environment by UUID.
func (r *EnvironmentResource) Get(ctx context.Context, uuid string) (Environment, error) {
	data, err := r.client.do(ctx, "GET", r.basePath()+url.PathEscape(uuid), nil, nil)
	if err != nil {
		return Environment{}, err
	}
	return decode[Environment](data)
}

// Delete removes a deployment environment by UUID.
func (r *EnvironmentResource) Delete(ctx context.Context, uuid string) error {
	_, err := r.client.do(ctx, "DELETE", r.basePath()+url.PathEscape(uuid), nil, nil)
	return err
}

// List returns all deployment environments in the repository.
func (r *EnvironmentResource) List(ctx context.Context) ([]Environment, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Environment]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
