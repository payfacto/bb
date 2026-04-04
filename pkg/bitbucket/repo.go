package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// RepoResource provides operations on repositories within a workspace.
type RepoResource struct {
	client    *Client
	workspace string
}

// List returns all repositories the authenticated user has access to in the workspace.
func (r *RepoResource) List(ctx context.Context) ([]Repo, error) {
	path := fmt.Sprintf("/repositories/%s", r.workspace)
	q := url.Values{"role": {"member"}, "pagelen": {"25"}}
	data, err := r.client.do(ctx, "GET", path, nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Repo]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
