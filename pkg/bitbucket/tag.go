package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// TagResource provides operations on repository tags.
type TagResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *TagResource) basePath() string {
	return fmt.Sprintf("%s/refs/tags", repoPath(r.workspace, r.repo))
}

// List returns all tags in the repository.
func (r *TagResource) List(ctx context.Context) ([]Tag, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Tag]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Create creates a new tag pointing at the given commit hash or branch name.
func (r *TagResource) Create(ctx context.Context, name, target string) (Tag, error) {
	input := CreateTagInput{
		Name:   name,
		Target: BranchTarget{Hash: target},
	}
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Tag{}, err
	}
	return decode[Tag](data)
}

// Delete removes a tag from the repository.
func (r *TagResource) Delete(ctx context.Context, name string) error {
	path := fmt.Sprintf("%s/%s", r.basePath(), url.PathEscape(name))
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
