package bitbucket

import (
	"context"
	"net/url"
)

// PipelineSSHKeyResource manages the SSH key pair Pipelines builds of a
// repository use to reach other hosts.
type PipelineSSHKeyResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *PipelineSSHKeyResource) path() string {
	return repoPath(r.workspace, r.repo) + "/pipelines_config/ssh/key_pair"
}

// Get returns the repository's Pipelines SSH key pair. Bitbucket never
// returns the private key, so only the public key is available. A repository
// without a key pair yields a 404 *APIError.
func (r *PipelineSSHKeyResource) Get(ctx context.Context) (PipelineSSHKeyPair, error) {
	data, err := r.client.do(ctx, "GET", r.path(), nil, nil)
	if err != nil {
		return PipelineSSHKeyPair{}, err
	}
	return decode[PipelineSSHKeyPair](data)
}

// Set creates or replaces the repository's Pipelines SSH key pair. The
// returned value holds the public key only.
func (r *PipelineSSHKeyResource) Set(ctx context.Context, privateKey, publicKey string) (PipelineSSHKeyPair, error) {
	body := pipelineSSHKeyPairInput{PrivateKey: privateKey, PublicKey: publicKey}
	data, err := r.client.do(ctx, "PUT", r.path(), body, nil)
	if err != nil {
		return PipelineSSHKeyPair{}, err
	}
	return decode[PipelineSSHKeyPair](data)
}

// Delete removes the repository's Pipelines SSH key pair.
func (r *PipelineSSHKeyResource) Delete(ctx context.Context) error {
	_, err := r.client.do(ctx, "DELETE", r.path(), nil, nil)
	return err
}

// PipelineKnownHostResource manages the SSH known hosts Pipelines builds of a
// repository trust.
type PipelineKnownHostResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *PipelineKnownHostResource) collectionPath() string {
	return repoPath(r.workspace, r.repo) + "/pipelines_config/ssh/known_hosts"
}

// List returns every known host of the repository, following pagination.
func (r *PipelineKnownHostResource) List(ctx context.Context) ([]PipelineKnownHost, error) {
	return fetchAllPages[PipelineKnownHost](ctx, r.client, r.collectionPath(), url.Values{"pagelen": {pagelenDefault}})
}

// Add registers a known host and returns it with its Bitbucket-assigned UUID
// and fingerprints.
func (r *PipelineKnownHostResource) Add(ctx context.Context, input AddKnownHostInput) (PipelineKnownHost, error) {
	data, err := r.client.do(ctx, "POST", r.collectionPath(), input, nil)
	if err != nil {
		return PipelineKnownHost{}, err
	}
	return decode[PipelineKnownHost](data)
}

// Delete removes a known host by UUID.
func (r *PipelineKnownHostResource) Delete(ctx context.Context, uuid string) error {
	_, err := r.client.do(ctx, "DELETE", r.collectionPath()+"/"+url.PathEscape(uuid), nil, nil)
	return err
}
