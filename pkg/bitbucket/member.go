package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// MemberResource provides operations on workspace members.
type MemberResource struct {
	client    *Client
	workspace string
}

func (r *MemberResource) basePath() string {
	return fmt.Sprintf("/workspaces/%s/members", r.workspace)
}

// List returns all members of the workspace, following pagination across all
// pages.
func (r *MemberResource) List(ctx context.Context) ([]WorkspaceMember, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	return fetchAllPages[WorkspaceMember](ctx, r.client, r.basePath(), q)
}
