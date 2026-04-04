package bitbucket

import (
	"context"
	"fmt"
	"net/url"
)

// WebhookResource provides operations on repository webhooks.
type WebhookResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *WebhookResource) basePath() string {
	return fmt.Sprintf("%s/hooks", repoPath(r.workspace, r.repo))
}

// List returns all webhooks for the repository.
func (r *WebhookResource) List(ctx context.Context) ([]Webhook, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Webhook]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Create adds a new webhook to the repository.
func (r *WebhookResource) Create(ctx context.Context, input CreateWebhookInput) (Webhook, error) {
	data, err := r.client.do(ctx, "POST", r.basePath(), input, nil)
	if err != nil {
		return Webhook{}, err
	}
	return decode[Webhook](data)
}

// Delete removes a webhook by its UUID string.
func (r *WebhookResource) Delete(ctx context.Context, uuid string) error {
	path := fmt.Sprintf("%s/%s", r.basePath(), url.PathEscape(uuid))
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
