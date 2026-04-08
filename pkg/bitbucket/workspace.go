package bitbucket

import (
	"context"
	"net/url"
)

// WorkspaceResource provides operations on Bitbucket workspaces.
type WorkspaceResource struct {
	client *Client
}

// List returns all workspaces accessible to the authenticated user.
func (r *WorkspaceResource) List(ctx context.Context) ([]Workspace, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	return fetchAllPages[Workspace](ctx, r.client, "/workspaces", q)
}
