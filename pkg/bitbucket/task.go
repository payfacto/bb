package bitbucket

// TaskResource provides operations on tasks within a specific PR.
type TaskResource struct {
	client    *Client
	workspace string
	repo      string
	prID      int
}
