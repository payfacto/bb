package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// CommentResource provides operations on pull request comments.
type CommentResource struct {
	client    *Client
	workspace string
	repo      string
	prID      int
}

func (r *CommentResource) basePath() string {
	return fmt.Sprintf("%s/pullrequests/%d/comments",
		repoPath(r.workspace, r.repo), r.prID)
}

// List returns all comments on the pull request, following pagination across
// all pages.
func (r *CommentResource) List(ctx context.Context) ([]Comment, error) {
	q := url.Values{"pagelen": {pagelenLarge}}
	return fetchAllPages[Comment](ctx, r.client, r.basePath(), q)
}

// Get returns a single comment by ID.
func (r *CommentResource) Get(ctx context.Context, commentID int) (Comment, error) {
	path := fmt.Sprintf("%s/%d", r.basePath(), commentID)
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Comment{}, err
	}
	return decode[Comment](data)
}

// Add posts a new comment. Set input.Inline for an inline comment on a specific file/line.
func (r *CommentResource) Add(ctx context.Context, input AddCommentInput) (Comment, error) {
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Comment{}, err
	}
	return decode[Comment](data)
}

// Resolve marks a top-level comment thread as resolved and returns the
// resolution details.
func (r *CommentResource) Resolve(ctx context.Context, commentID int) (CommentResolution, error) {
	data, err := r.client.do(ctx, "POST", r.resolvePath(commentID), nil, nil)
	if err != nil {
		return CommentResolution{}, err
	}
	return decode[CommentResolution](data)
}

// Reopen reopens a previously resolved comment thread.
func (r *CommentResource) Reopen(ctx context.Context, commentID int) error {
	_, err := r.client.do(ctx, "DELETE", r.resolvePath(commentID), nil, nil)
	return err
}

func (r *CommentResource) resolvePath(commentID int) string {
	return fmt.Sprintf("%s/%d/resolve", r.basePath(), commentID)
}

// Reply posts a reply to an existing comment identified by parentID.
func (r *CommentResource) Reply(ctx context.Context, parentID int, text string) (Comment, error) {
	input := AddCommentInput{
		Content: Content{Raw: text},
		Parent:  &Parent{ID: parentID},
	}
	return r.Add(ctx, input)
}
