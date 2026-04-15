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

	"github.com/payfacto/bb/internal/config"
)

const (
	defaultAPIBase    = "https://api.bitbucket.org/2.0"
	defaultHTTPTimeout = 30 * time.Second
	pagelenDefault    = "50"  // standard page size for most list endpoints
	pagelenSmall      = "25"  // reduced page size for heavy payloads (pipelines, commits, repos)
	pagelenLarge      = "100" // larger page size for lightweight items (comments, tasks)
)

// Client is the Bitbucket Cloud HTTP client.
type Client struct {
	baseURL     string
	username    string
	token       string
	bearerToken string // set when using OAuth; takes priority over Basic auth
	httpClient  *http.Client
}

// New creates a Client from cfg using the live Bitbucket API.
func New(cfg *config.Config) *Client {
	return &Client{
		baseURL:  defaultAPIBase,
		username: cfg.Username,
		token:    cfg.Token,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// NewWithBaseURL creates a Client with a custom base URL. Used in tests.
func NewWithBaseURL(cfg *config.Config, baseURL string) *Client {
	c := New(cfg)
	c.baseURL = baseURL
	return c
}

// NewWithBearerToken creates a Client authenticated with an OAuth Bearer token.
// Used when no full config is available (e.g., during bb auth login).
func NewWithBearerToken(token string) *Client {
	return &Client{
		baseURL:     defaultAPIBase,
		bearerToken: token,
		httpClient:  &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// SetBearerToken configures the client to use OAuth Bearer token auth instead of Basic auth.
func (c *Client) SetBearerToken(token string) {
	c.bearerToken = token
}

// setAuth sets the Authorization header on req using Bearer token (OAuth) or Basic auth (app password).
func (c *Client) setAuth(req *http.Request) {
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	} else {
		req.SetBasicAuth(c.username, c.token)
	}
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

// Tags returns a resource for tag operations on the given repo.
func (c *Client) Tags(workspace, repo string) *TagResource {
	return &TagResource{client: c, workspace: workspace, repo: repo}
}

// Environments returns a resource for deployment environment operations on the given repo.
func (c *Client) Environments(workspace, repo string) *EnvironmentResource {
	return &EnvironmentResource{client: c, workspace: workspace, repo: repo}
}

// Deployments returns a resource for deployment operations on the given repo.
func (c *Client) Deployments(workspace, repo string) *DeploymentResource {
	return &DeploymentResource{client: c, workspace: workspace, repo: repo}
}

// Commits returns a resource for commit history and file access on the given repo.
func (c *Client) Commits(workspace, repo string) *CommitResource {
	return &CommitResource{client: c, workspace: workspace, repo: repo}
}

// User returns a resource for authenticated user operations.
func (c *Client) User() *UserResource {
	return &UserResource{client: c}
}

// Repos returns a resource for listing repositories in a workspace.
func (c *Client) Repos(workspace string) *RepoResource {
	return &RepoResource{client: c, workspace: workspace}
}

// Members returns a resource for workspace member operations.
func (c *Client) Members(workspace string) *MemberResource {
	return &MemberResource{client: c, workspace: workspace}
}

// Projects returns a resource for workspace project operations.
func (c *Client) Projects(workspace string) *ProjectResource {
	return &ProjectResource{client: c, workspace: workspace}
}

// Snippets returns a resource for workspace snippet operations.
func (c *Client) Snippets(workspace string) *SnippetResource {
	return &SnippetResource{client: c, workspace: workspace}
}

// Downloads returns a resource for download artifact operations on the given repo.
func (c *Client) Downloads(workspace, repo string) *DownloadResource {
	return &DownloadResource{client: c, workspace: workspace, repo: repo}
}

// DeployKeys returns a resource for deploy key operations on the given repo.
func (c *Client) DeployKeys(workspace, repo string) *DeployKeyResource {
	return &DeployKeyResource{client: c, workspace: workspace, repo: repo}
}

// Issues returns a resource for issue operations on the given repo.
func (c *Client) Issues(workspace, repo string) *IssueResource {
	return &IssueResource{client: c, workspace: workspace, repo: repo}
}

// Restrictions returns a resource for branch restriction operations on the given repo.
func (c *Client) Restrictions(workspace, repo string) *BranchRestrictionResource {
	return &BranchRestrictionResource{client: c, workspace: workspace, repo: repo}
}

// PipelineVariables returns a resource for pipeline variable operations on the given repo.
func (c *Client) PipelineVariables(workspace, repo string) *PipelineVariableResource {
	return &PipelineVariableResource{client: c, workspace: workspace, repo: repo}
}

// Webhooks returns a resource for webhook operations on the given repo.
func (c *Client) Webhooks(workspace, repo string) *WebhookResource {
	return &WebhookResource{client: c, workspace: workspace, repo: repo}
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

	c.setAuth(req)
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
		return nil, parseHTTPError(resp.StatusCode, data)
	}

	return data, nil
}

// doText executes an authenticated GET that returns plain text (e.g. log endpoints).
func (c *Client) doText(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)
	// Do not set Accept header — the log endpoint rejects Accept: text/plain
	// and may redirect (307) to S3 where JSON Accept headers are also rejected.

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

// doMultipart executes an authenticated POST with a raw body and explicit content-type.
// Used for multipart/form-data uploads where JSON marshaling is not appropriate.
func (c *Client) doMultipart(ctx context.Context, path string, body io.Reader, contentType string) ([]byte, error) {
	u := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, "POST", u, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)

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
		return nil, parseHTTPError(resp.StatusCode, data)
	}

	return data, nil
}

// parseHTTPError converts a non-2xx HTTP response into an error.
// It attempts to extract the API's JSON error envelope; falls back to the raw body.
func parseHTTPError(statusCode int, data []byte) error {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Error.Message != "" {
		return fmt.Errorf("HTTP %d: %s", statusCode, envelope.Error.Message)
	}
	return fmt.Errorf("HTTP %d: %s", statusCode, string(data))
}

// decode unmarshals JSON data into a typed value T.
func decode[T any](data []byte) (T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return v, fmt.Errorf("decode response: %w", err)
	}
	return v, nil
}

// fetchPage fetches a single absolute URL and returns its body. Used by fetchAllPages to follow
// "next" pagination links. Each call closes its own response body before returning.
func (c *Client) fetchPage(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)
	req.Header.Set("Accept", "application/json")

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
		return nil, parseHTTPError(resp.StatusCode, data)
	}
	return data, nil
}

// fetchAllPages fetches all pages for a given path+query, following "next" links.
func fetchAllPages[T any](ctx context.Context, c *Client, path string, q url.Values) ([]T, error) {
	var all []T
	nextURL := ""
	for {
		var data []byte
		var err error
		if nextURL != "" {
			data, err = c.fetchPage(ctx, nextURL)
		} else {
			data, err = c.do(ctx, "GET", path, nil, q)
		}
		if err != nil {
			return nil, err
		}
		page, err := decode[paged[T]](data)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Values...)
		if page.Next == "" {
			break
		}
		nextURL = page.Next
	}
	return all, nil
}
