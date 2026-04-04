package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/payfactopay/bb/internal/config"
)

const defaultAPIBase = "https://api.bitbucket.org/2.0"

// Client is the Bitbucket Cloud HTTP client.
type Client struct {
	baseURL    string
	username   string
	token      string
	httpClient *http.Client
}

// New creates a Client from cfg using the live Bitbucket API.
func New(cfg *config.Config) *Client {
	return &Client{
		baseURL:  defaultAPIBase,
		username: cfg.Username,
		token:    cfg.Token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewWithBaseURL creates a Client with a custom base URL. Used in tests.
func NewWithBaseURL(cfg *config.Config, baseURL string) *Client {
	c := New(cfg)
	c.baseURL = baseURL
	return c
}

// PRs returns a PRResource scoped to the given workspace and repo.
func (c *Client) PRs(workspace, repo string) *PRResource {
	return &PRResource{client: c, workspace: workspace, repo: repo}
}

// Comments returns a CommentResource scoped to a specific PR.
func (c *Client) Comments(workspace, repo string, prID int) *CommentResource {
	return &CommentResource{client: c, workspace: workspace, repo: repo, prID: prID}
}

// Tasks returns a TaskResource scoped to a specific PR.
func (c *Client) Tasks(workspace, repo string, prID int) *TaskResource {
	return &TaskResource{client: c, workspace: workspace, repo: repo, prID: prID}
}

// Pipelines returns a resource for pipeline operations on the given repo.
func (c *Client) Pipelines(workspace, repo string) *PipelineResource {
	return &PipelineResource{client: c, workspace: workspace, repo: repo}
}

// Branches returns a resource for branch operations on the given repo.
func (c *Client) Branches(workspace, repo string) *BranchResource {
	return &BranchResource{client: c, workspace: workspace, repo: repo}
}

// repoPath returns the API path prefix for a repository.
func repoPath(workspace, repo string) string {
	return fmt.Sprintf("/repositories/%s/%s", workspace, repo)
}

// do executes an authenticated HTTP request and returns the raw response body.
// It returns an error for HTTP 4xx/5xx responses.
func (c *Client) do(ctx context.Context, method, path string, body any, query url.Values) ([]byte, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// decode unmarshals JSON data into a typed value T.
func decode[T any](data []byte) (T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return v, fmt.Errorf("decode response: %w", err)
	}
	return v, nil
}
