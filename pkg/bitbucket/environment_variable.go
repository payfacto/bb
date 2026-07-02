package bitbucket

import (
	"context"
	"net/url"
)

// EnvironmentVariableResource provides operations on deployment-environment
// variables. The API returns the same shape as repository pipeline variables
// (type "pipeline_variable"), so it reuses PipelineVariable and
// CreatePipelineVariableInput.
type EnvironmentVariableResource struct {
	client    *Client
	workspace string
	repo      string
	envUUID   string
}

func (r *EnvironmentVariableResource) basePath() string {
	return repoPath(r.workspace, r.repo) + "/deployments_config/environments/" + url.PathEscape(r.envUUID) + "/variables/"
}

// List returns all variables for the environment. Secured variables are
// returned with an empty value (the API hides it).
func (r *EnvironmentVariableResource) List(ctx context.Context) ([]PipelineVariable, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[PipelineVariable]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}
