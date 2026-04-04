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

// List returns all comments on the pull request.
func (r *CommentResource) List(ctx context.Context) ([]Comment, error) {
	q := url.Values{"pagelen": {"100"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Comment]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Add posts a new comment. Set input.Inline for an inline comment on a specific file/line.
func (r *CommentResource) Add(ctx context.Context, input AddCommentInput) (Comment, error) {
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Comment{}, err
	}
	return decode[Comment](data)
}

// Reply posts a reply to an existing comment identified by parentID.
func (r *CommentResource) Reply(ctx context.Context, parentID int, text string) (Comment, error) {
	input := AddCommentInput{
		Content: Content{Raw: text},
		Parent:  &Parent{ID: parentID},
	}
	return r.Add(ctx, input)
}
