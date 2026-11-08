package bitbucket

import (
	"context"
	"net/url"
)

// WorkspaceResource provides operations on Bitbucket workspaces.
type WorkspaceResource struct {
	client *Client
}

// workspaceAccess is one entry of GET /user/workspaces: a workspace plus the
// caller's admin flag on it.
type workspaceAccess struct {
	Administrator bool      `json:"administrator"`
	Workspace     Workspace `json:"workspace"`
}

// List returns all workspaces accessible to the authenticated user.
//
// It uses GET /user/workspaces; the older GET /workspaces was removed in
// Bitbucket's cross-workspace API end-of-life and now returns 404. That
// endpoint returns only slug/uuid/links, so each entry is enriched with
// GET /workspaces/{slug} for its name and web link. Enrichment is best effort:
// if a detail lookup fails, the slug-only entry is kept. A cancelled context
// still aborts the whole call.
func (r *WorkspaceResource) List(ctx context.Context) ([]Workspace, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	entries, err := fetchAllPages[workspaceAccess](ctx, r.client, "/user/workspaces", q)
	if err != nil {
		return nil, err
	}
	workspaces := make([]Workspace, len(entries))
	for i, entry := range entries {
		if ctx.Err() != nil {
			break
		}
		workspaces[i] = r.withDetails(ctx, entry.Workspace)
		workspaces[i].Administrator = entry.Administrator
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return workspaces, nil
}

// withDetails returns the full workspace for base, or base itself when the
// detail lookup fails.
func (r *WorkspaceResource) withDetails(ctx context.Context, base Workspace) Workspace {
	data, err := r.client.do(ctx, "GET", "/workspaces/"+url.PathEscape(base.Slug), nil, nil)
	if err != nil {
		return base
	}
	full, err := decode[Workspace](data)
	if err != nil {
		return base
	}
	return full
}
