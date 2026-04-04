package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// BranchRestrictionResource provides operations on repository branch restrictions.
type BranchRestrictionResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *BranchRestrictionResource) basePath() string {
	return fmt.Sprintf("%s/branch-restrictions", repoPath(r.workspace, r.repo))
}

// List returns all branch restrictions for the repository.
func (r *BranchRestrictionResource) List(ctx context.Context) ([]BranchRestriction, error) {
	q := url.Values{"pagelen": {"50"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[BranchRestriction]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Create adds a new branch restriction to the repository.
func (r *BranchRestrictionResource) Create(ctx context.Context, input CreateBranchRestrictionInput) (BranchRestriction, error) {
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return BranchRestriction{}, err
	}
	return decode[BranchRestriction](data)
}

// Delete removes a branch restriction by its integer ID.
func (r *BranchRestrictionResource) Delete(ctx context.Context, id int) error {
	path := fmt.Sprintf("%s/%d", r.basePath(), id)
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
