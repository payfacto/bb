package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// CommitResource provides operations on commits and file contents within a repository.
type CommitResource struct {
	client    *Client
	workspace string
	repo      string
}

// List returns commits, newest first. If branch is empty, returns commits from the repo's default branch.
func (r *CommitResource) List(ctx context.Context, branch string) ([]Commit, error) {
	var path string
	if branch == "" {
		path = fmt.Sprintf("%s/commits", repoPath(r.workspace, r.repo))
	} else {
		path = fmt.Sprintf("%s/commits/%s", repoPath(r.workspace, r.repo), url.PathEscape(branch))
	}
	q := url.Values{"pagelen": {pagelenSmall}}
	data, err := r.client.do(ctx, "GET", path, nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Commit]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Get returns a single commit by hash.
// Note: Bitbucket uses the singular "/commit/" path for single commit lookup.
func (r *CommitResource) Get(ctx context.Context, hash string) (Commit, error) {
	path := fmt.Sprintf("%s/commit/%s", repoPath(r.workspace, r.repo), hash)
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Commit{}, err
	}
	return decode[Commit](data)
}

// File returns the raw content of a file at the given ref (branch, tag, or commit hash).
// The response is plain text, not JSON.
func (r *CommitResource) File(ctx context.Context, ref, filePath string) (string, error) {
	apiPath := fmt.Sprintf("%s/src/%s/%s", repoPath(r.workspace, r.repo), ref, filePath)
	data, err := r.client.do(ctx, "GET", apiPath, nil, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
