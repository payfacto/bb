package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// BranchResource provides operations on repository branches.
type BranchResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *BranchResource) basePath() string {
	return fmt.Sprintf("%s/refs/branches", repoPath(r.workspace, r.repo))
}

// List returns all branches in the repository.
func (r *BranchResource) List(ctx context.Context) ([]Branch, error) {
	q := url.Values{"pagelen": {"50"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Branch]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Create creates a new branch from the given commit hash or branch name.
func (r *BranchResource) Create(ctx context.Context, name, target string) (Branch, error) {
	input := CreateBranchInput{
		Name:   name,
		Target: BranchTarget{Hash: target},
	}
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Branch{}, err
	}
	return decode[Branch](data)
}

// Delete removes a branch from the repository.
func (r *BranchResource) Delete(ctx context.Context, name string) error {
	path := fmt.Sprintf("%s/%s", r.basePath(), url.PathEscape(name))
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
