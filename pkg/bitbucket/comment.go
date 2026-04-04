package bitbucket

// CommentResource provides operations on comments within a specific PR.
type CommentResource struct {
	client    *Client
	workspace string
	repo      string
	prID      int
}
