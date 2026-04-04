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

// List returns all members of the workspace.
func (r *MemberResource) List(ctx context.Context) ([]WorkspaceMember, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[WorkspaceMember]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
