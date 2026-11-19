package bitbucket

import (
	"context"
	"errors"
	"net/url"
	"strings"
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
// returned value holds the public key only, and an error never contains
// private key material even if Bitbucket's response echoes the request.
func (r *PipelineSSHKeyResource) Set(ctx context.Context, privateKey, publicKey string) (PipelineSSHKeyPair, error) {
	body := pipelineSSHKeyPairInput{PrivateKey: privateKey, PublicKey: publicKey}
	data, err := r.client.do(ctx, "PUT", r.path(), body, nil)
	if err != nil {
		return PipelineSSHKeyPair{}, redactSecret(err, privateKey)
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

const (
	redactedMarker = "[REDACTED]"
	// minSecretFragment is the shortest secret line worth redacting; shorter
	// ones (blank lines, padding tails) would blank out ordinary text.
	minSecretFragment = 8
)

// redactSecret removes every line of secret from an *APIError's Message and
// Body, so an error response that echoes the request cannot leak it. Matching
// is per line, which also catches a JSON-escaped echo (lines survive between
// the \n escapes). PEM armor lines are not secret and are left readable.
func redactSecret(err error, secret string) error {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	for _, line := range strings.Split(secret, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < minSecretFragment || strings.HasPrefix(line, "-----") {
			continue
		}
		apiErr.Message = strings.ReplaceAll(apiErr.Message, line, redactedMarker)
		apiErr.Body = strings.ReplaceAll(apiErr.Body, line, redactedMarker)
	}
	return err
}
