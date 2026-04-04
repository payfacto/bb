package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// PRResource provides operations on pull requests within a specific repo.
type PRResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *PRResource) prPath(prID int) string {
	return fmt.Sprintf("%s/pullrequests/%d", repoPath(r.workspace, r.repo), prID)
}

// List returns pull requests filtered by state (e.g. "OPEN", "MERGED").
func (r *PRResource) List(ctx context.Context, state string) ([]PR, error) {
	q := url.Values{"state": {state}, "pagelen": {"50"}}
	data, err := r.client.do(ctx, "GET", repoPath(r.workspace, r.repo)+"/pullrequests", nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[PR]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Get returns a single pull request by ID.
func (r *PRResource) Get(ctx context.Context, prID int) (PR, error) {
	data, err := r.client.do(ctx, "GET", r.prPath(prID), nil, nil)
	if err != nil {
		return PR{}, err
	}
	return decode[PR](data)
}

// Create opens a new pull request.
func (r *PRResource) Create(ctx context.Context, input CreatePRInput) (PR, error) {
	data, err := r.client.do(ctx, "POST", repoPath(r.workspace, r.repo)+"/pullrequests", input, nil)
	if err != nil {
		return PR{}, err
	}
	return decode[PR](data)
}

// Diff returns the raw patch text for the pull request.
// The response is plain text (not JSON).
func (r *PRResource) Diff(ctx context.Context, prID int) (string, error) {
	data, err := r.client.do(ctx, "GET", r.prPath(prID)+"/diff", nil, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Approve approves the pull request.
func (r *PRResource) Approve(ctx context.Context, prID int) error {
	_, err := r.client.do(ctx, "POST", r.prPath(prID)+"/approve", nil, nil)
	return err
}

// Merge merges the pull request using the given strategy.
// strategy: "merge_commit", "squash", or "fast_forward".
func (r *PRResource) Merge(ctx context.Context, prID int, strategy string) error {
	body := map[string]string{"type": "pullrequest", "merge_strategy": strategy}
	_, err := r.client.do(ctx, "POST", r.prPath(prID)+"/merge", body, nil)
	return err
}

// Decline declines the pull request.
func (r *PRResource) Decline(ctx context.Context, prID int) error {
	_, err := r.client.do(ctx, "POST", r.prPath(prID)+"/decline", nil, nil)
	return err
}
