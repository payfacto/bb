package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// DeployKeyResource provides operations on repository deploy keys.
type DeployKeyResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *DeployKeyResource) basePath() string {
	return fmt.Sprintf("%s/deploy-keys", repoPath(r.workspace, r.repo))
}

// List returns all deploy keys for the repository.
func (r *DeployKeyResource) List(ctx context.Context) ([]DeployKey, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[DeployKey]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Add creates a new deploy key with the given label and SSH public key.
func (r *DeployKeyResource) Add(ctx context.Context, label, key string) (DeployKey, error) {
	input := AddDeployKeyInput{Label: label, Key: key}
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return DeployKey{}, err
	}
	return decode[DeployKey](data)
}

// Delete removes a deploy key by its integer ID.
func (r *DeployKeyResource) Delete(ctx context.Context, id int) error {
	path := fmt.Sprintf("%s/%d", r.basePath(), id)
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
