package bitbucket

import (
	"context"
	"fmt"
	"net/url"
	"strings"
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

// List returns pull requests matching opts, following pagination to return the
// full result set (not just the first page). State is sent as a query
// parameter; SourceBranch, Since, and Until are combined into a Bitbucket BBQL
// "q" expression (Since/Until bound created_on). Sort optionally orders the
// results ("-" prefix for descending, e.g. "-updated_on").
func (r *PRResource) List(ctx context.Context, opts PRListOptions) ([]PR, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if opts.Sort != "" {
		q.Set("sort", opts.Sort)
	}

	var clauses []string
	if opts.SourceBranch != "" {
		clauses = append(clauses, fmt.Sprintf(`source.branch.name=%s`, bbqlQuote(opts.SourceBranch)))
	}
	if opts.Since != "" {
		clauses = append(clauses, fmt.Sprintf(`created_on>=%s`, bbqlQuote(opts.Since)))
	}
	if opts.Until != "" {
		clauses = append(clauses, fmt.Sprintf(`created_on<=%s`, bbqlQuote(opts.Until)))
	}
	if opts.Query != "" {
		clauses = append(clauses, fmt.Sprintf(`(title ~ %s OR description ~ %s)`, bbqlQuote(opts.Query), bbqlQuote(opts.Query)))
	}
	if len(clauses) > 0 {
		q.Set("q", strings.Join(clauses, " AND "))
	}

	return fetchPagesLimit[PR](ctx, r.client, repoPath(r.workspace, r.repo)+"/pullrequests", q, opts.Limit)
}

// ListByAuthor returns all pull requests authored by the given nickname,
// following pagination to return the full result set.
func (r *PRResource) ListByAuthor(ctx context.Context, nickname string) ([]PR, error) {
	q := url.Values{
		"q":       {fmt.Sprintf(`author.nickname=%s`, bbqlQuote(nickname))},
		"state":   {"ALL"},
		"pagelen": {pagelenDefault},
	}
	return fetchAllPages[PR](ctx, r.client, repoPath(r.workspace, r.repo)+"/pullrequests", q)
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

// Activity returns the full activity timeline for a pull request, following
// pagination across all pages.
func (r *PRResource) Activity(ctx context.Context, prID int) ([]Activity, error) {
	path := fmt.Sprintf("%s/activity", r.prPath(prID))
	return fetchAllPages[Activity](ctx, r.client, path, url.Values{"pagelen": {pagelenDefault}})
}

// AddReviewer adds a reviewer (by account_id) to an existing pull request.
// It fetches the current PR to preserve existing reviewers.
func (r *PRResource) AddReviewer(ctx context.Context, prID int, accountID string) error {
	pr, err := r.Get(ctx, prID)
	if err != nil {
		return fmt.Errorf("get PR: %w", err)
	}
	for _, rv := range pr.Reviewers {
		if rv.AccountID == accountID {
			return nil // already a reviewer
		}
	}
	reviewers := append(pr.Reviewers, Actor{AccountID: accountID})
	input := UpdatePRReviewersInput{Title: pr.Title, Reviewers: reviewers}
	_, err = r.client.do(ctx, "PUT", r.prPath(prID), input, nil)
	return err
}

// Statuses returns the build statuses associated with the pull request's source
// commit, following pagination across all pages.
func (r *PRResource) Statuses(ctx context.Context, prID int) ([]PRStatus, error) {
	path := fmt.Sprintf("%s/statuses", r.prPath(prID))
	return fetchAllPages[PRStatus](ctx, r.client, path, url.Values{"pagelen": {pagelenDefault}})
}
