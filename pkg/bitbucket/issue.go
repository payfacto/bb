package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// IssueResource provides operations on repository issues.
type IssueResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *IssueResource) basePath() string {
	return fmt.Sprintf("%s/issues", repoPath(r.workspace, r.repo))
}

// List returns all issues in the repository.
func (r *IssueResource) List(ctx context.Context) ([]Issue, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Issue]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Get returns a single issue by its integer ID.
func (r *IssueResource) Get(ctx context.Context, id int) (Issue, error) {
	path := fmt.Sprintf("%s/%d", r.basePath(), id)
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Issue{}, err
	}
	return decode[Issue](data)
}

// Create creates a new issue with the provided fields.
func (r *IssueResource) Create(ctx context.Context, input CreateIssueInput) (Issue, error) {
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Issue{}, err
	}
	return decode[Issue](data)
}
