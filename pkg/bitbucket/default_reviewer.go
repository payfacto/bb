package bitbucket

import "context"

// DefaultReviewerResource provides read access to a repository's default
// reviewers.
type DefaultReviewerResource struct {
	client    *Client
	workspace string
	repo      string
}

// List returns the reviewers configured directly on the repository. The
// endpoint returns bare accounts, so each is wrapped in a DefaultReviewer
// with an empty ReviewerType, keeping one shape for both List and Effective.
func (r *DefaultReviewerResource) List(ctx context.Context) ([]DefaultReviewer, error) {
	path := repoPath(r.workspace, r.repo) + "/default-reviewers"
	accounts, err := fetchAllPages[Actor](ctx, r.client, path, nil)
	if err != nil {
		return nil, err
	}
	reviewers := make([]DefaultReviewer, len(accounts))
	for i, account := range accounts {
		reviewers[i] = DefaultReviewer{User: account}
	}
	return reviewers, nil
}

// Effective returns the reviewers actually applied to new pull requests:
// repository reviewers plus those inherited from the project, each tagged
// with its ReviewerType ("repository" or "project").
func (r *DefaultReviewerResource) Effective(ctx context.Context) ([]DefaultReviewer, error) {
	path := repoPath(r.workspace, r.repo) + "/effective-default-reviewers"
	return fetchAllPages[DefaultReviewer](ctx, r.client, path, nil)
}
