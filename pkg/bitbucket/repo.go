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

// Get returns a single repository by slug.
func (r *RepoResource) Get(ctx context.Context, slug string) (Repo, error) {
	path := fmt.Sprintf("/repositories/%s/%s", r.workspace, slug)
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Repo{}, err
	}
	return decode[Repo](data)
}

// List returns all repositories the authenticated user has access to in the workspace.
func (r *RepoResource) List(ctx context.Context) ([]Repo, error) {
	path := fmt.Sprintf("/repositories/%s", r.workspace)
	q := url.Values{"role": {"member"}, "pagelen": {pagelenLarge}}
	return fetchAllPages[Repo](ctx, r.client, path, q)
}

// ListByProject returns all repositories belonging to the given project key.
func (r *RepoResource) ListByProject(ctx context.Context, projectKey string) ([]Repo, error) {
	path := fmt.Sprintf("/repositories/%s", r.workspace)
	q := url.Values{
		"q":       {fmt.Sprintf(`project.key="%s"`, projectKey)},
		"pagelen": {pagelenLarge},
	}
	return fetchAllPages[Repo](ctx, r.client, path, q)
}

// Create creates a new repository in the workspace.
func (r *RepoResource) Create(ctx context.Context, slug string, input CreateRepoInput) (Repo, error) {
	path := fmt.Sprintf("/repositories/%s/%s", r.workspace, slug)
	data, err := r.client.do(ctx, "POST", path, input, nil)
	if err != nil {
		return Repo{}, err
	}
	return decode[Repo](data)
}

// Fork forks an existing repository into the workspace (or target workspace).
// sourceSlug is the slug of the repo to fork within r.workspace.
func (r *RepoResource) Fork(ctx context.Context, sourceSlug string, input ForkRepoInput) (Repo, error) {
	path := fmt.Sprintf("/repositories/%s/%s/forks", r.workspace, sourceSlug)
	data, err := r.client.do(ctx, "POST", path, input, nil)
	if err != nil {
		return Repo{}, err
	}
	return decode[Repo](data)
}
