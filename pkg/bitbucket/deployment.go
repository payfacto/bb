package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// DeploymentResource provides operations on repository deployments.
type DeploymentResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *DeploymentResource) basePath() string {
	return fmt.Sprintf("%s/deployments/", repoPath(r.workspace, r.repo))
}

// List returns the most recent deployments, newest first.
func (r *DeploymentResource) List(ctx context.Context) ([]Deployment, error) {
	q := url.Values{"sort": {"-last_update_time"}, "pagelen": {"25"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Deployment]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
