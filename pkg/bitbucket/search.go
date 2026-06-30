package bitbucket

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// SearchResource provides workspace-scoped search operations.
type SearchResource struct {
	client    *Client
	workspace string
}

// Code searches file contents across the workspace's indexed default branches.
// The raw Query is passed to Bitbucket verbatim; Ext/Lang/Repo/Project are
// folded in as search modifiers. Results are capped by opts.Limit (<= 0 = all).
func (s *SearchResource) Code(ctx context.Context, opts CodeSearchOptions) ([]CodeSearchResult, error) {
	path := fmt.Sprintf("/workspaces/%s/search/code", s.workspace)
	q := url.Values{
		"search_query": {opts.searchQuery()},
		"pagelen":      {pagelenSmall},
	}
	return fetchPagesLimit[CodeSearchResult](ctx, s.client, path, q, opts.Limit)
}

// Repos finds repositories in the workspace whose name or description matches
// term (BBQL "~" contains). Results are capped by limit (<= 0 = all).
func (s *SearchResource) Repos(ctx context.Context, term string, limit int) ([]Repo, error) {
	path := fmt.Sprintf("/repositories/%s", s.workspace)
	q := url.Values{
		"q":       {fmt.Sprintf(`name ~ "%s" OR description ~ "%s"`, term, term)},
		"pagelen": {pagelenSmall},
	}
	return fetchPagesLimit[Repo](ctx, s.client, path, q, limit)
}

// searchQuery assembles the search_query string: raw terms first, then each
// modifier. Comma-separated modifier values become repeated modifiers, which
// Bitbucket OR-combines (e.g. Ext "js,jsx" -> "ext:js ext:jsx").
func (o CodeSearchOptions) searchQuery() string {
	var parts []string
	if q := strings.TrimSpace(o.Query); q != "" {
		parts = append(parts, q)
	}
	addMod := func(key, val string) {
		for _, v := range strings.Split(val, ",") {
			if v = strings.TrimSpace(v); v != "" {
				parts = append(parts, key+":"+v)
			}
		}
	}
	addMod("ext", o.Ext)
	addMod("lang", o.Lang)
	addMod("repo", o.Repo)
	addMod("project", o.Project)
	return strings.Join(parts, " ")
}
