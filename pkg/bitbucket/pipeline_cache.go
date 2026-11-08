package bitbucket

import (
	"context"
	"net/url"
)

// PipelineCacheResource provides operations on a repository's Pipelines
// dependency caches.
type PipelineCacheResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *PipelineCacheResource) basePath() string {
	return repoPath(r.workspace, r.repo) + "/pipelines-config/caches"
}

// List returns every cache version stored for the repository, following
// pagination. A cache name can have several versions (one per key hash).
func (r *PipelineCacheResource) List(ctx context.Context) ([]PipelineCache, error) {
	return fetchAllPages[PipelineCache](ctx, r.client, r.basePath(), nil)
}

// DeleteByName removes every version of the named cache.
func (r *PipelineCacheResource) DeleteByName(ctx context.Context, name string) error {
	_, err := r.client.do(ctx, "DELETE", r.basePath(), nil, url.Values{"name": {name}})
	return err
}

// ContentURI returns a short-lived download URI for a cache version's archive.
func (r *PipelineCacheResource) ContentURI(ctx context.Context, uuid string) (PipelineCacheContentURI, error) {
	data, err := r.client.do(ctx, "GET", r.basePath()+"/"+url.PathEscape(uuid)+"/content-uri", nil, nil)
	if err != nil {
		return PipelineCacheContentURI{}, err
	}
	return decode[PipelineCacheContentURI](data)
}
