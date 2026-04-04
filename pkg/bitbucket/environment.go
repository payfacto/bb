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

// List returns all deployment environments in the repository.
func (r *EnvironmentResource) List(ctx context.Context) ([]Environment, error) {
	q := url.Values{"pagelen": {"50"}}
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
