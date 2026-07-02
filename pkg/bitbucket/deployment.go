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

// Get returns a single deployment by UUID.
func (r *DeploymentResource) Get(ctx context.Context, uuid string) (Deployment, error) {
	data, err := r.client.do(ctx, "GET", r.basePath()+uuid, nil, nil)
	if err != nil {
		return Deployment{}, err
	}
	return decode[Deployment](data)
}

// DeploymentListOptions filters and orders a deployment listing.
type DeploymentListOptions struct {
	EnvUUID string // filter to a single environment UUID (empty = all)
	Sort    string // Bitbucket sort field, "-" prefix for descending (empty = default)
}

// List returns the most recent deployments, optionally filtered by environment
// and ordered by Sort.
//
// NOTE: The Bitbucket deployments endpoint silently ignores q= BBQL filters,
// so EnvUUID filtering is applied client-side after fetching the first page
// (pagelen=25). Deployments beyond the first page are not matched when
// EnvUUID is set.
func (r *DeploymentResource) List(ctx context.Context, opts DeploymentListOptions) ([]Deployment, error) {
	q := url.Values{"pagelen": {pagelenSmall}}
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Deployment]](data)
	if err != nil {
		return nil, err
	}
	if opts.EnvUUID == "" {
		return page.Values, nil
	}
	filtered := make([]Deployment, 0, len(page.Values))
	for _, d := range page.Values {
		if d.Environment.UUID == opts.EnvUUID {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}
