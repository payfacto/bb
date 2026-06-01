package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/payfacto/bb/internal/config"
)

const (
	defaultAPIBase     = "https://api.bitbucket.org/2.0"
	defaultHTTPTimeout = 30 * time.Second
	pagelenDefault     = "50"  // standard page size for most list endpoints
	pagelenSmall       = "25"  // reduced page size for heavy payloads (pipelines, commits, repos)
	pagelenLarge       = "100" // larger page size for lightweight items (comments, tasks)
)

// Client is the Bitbucket Cloud HTTP client.
type Client struct {
	baseURL    string
	username   string
	token      string
	httpClient *http.Client

	// mu guards bearerToken and serialises token refresh so concurrent
	// requests don't refresh more than once.
	mu          sync.Mutex
	bearerToken string // set when using OAuth; takes priority over Basic auth
	// refresher, when set, is invoked once on an HTTP 401 from a bearer-auth
	// request to obtain a fresh access token. It returns the new token.
	refresher func() (string, error)
}

// New creates a Client from cfg using the live Bitbucket API.
func New(cfg *config.Config) *Client {
	return &Client{
		baseURL:    defaultAPIBase,
		username:   cfg.Username,
		token:      cfg.Token,
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
	c.mu.Lock()
	c.bearerToken = token
	c.mu.Unlock()
}

// SetTokenRefresher installs a callback invoked when a bearer-authenticated
// request returns HTTP 401. The callback should obtain and persist a fresh
// access token and return it; the client then retries the request once with
// the new token. Only meaningful for OAuth (bearer) auth.
func (c *Client) SetTokenRefresher(fn func() (string, error)) {
	c.mu.Lock()
	c.refresher = fn
	c.mu.Unlock()
}

// setAuth sets the Authorization header on req using Bearer token (OAuth) or Basic auth (app password).
func (c *Client) setAuth(req *http.Request) {
	c.mu.Lock()
	bearer := c.bearerToken
	c.mu.Unlock()
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	} else {
		req.SetBasicAuth(c.username, c.token)
	}
}

// send authenticates and executes req. When a bearer-auth request returns
// HTTP 401 and a refresher is installed, it refreshes the access token once and
// retries the request. Requests whose body cannot be replayed (e.g. streamed
// uploads) are not retried; the 401 is returned to the caller instead.
func (c *Client) send(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	tokenUsed := c.bearerToken
	refresher := c.refresher
	c.mu.Unlock()

	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized || tokenUsed == "" || refresher == nil {
		return resp, nil
	}

	retry, ok := cloneForRetry(req)
	if !ok {
		return resp, nil // body not replayable — surface the original 401
	}
	newToken, refreshErr := c.refresh(tokenUsed, refresher)
	if refreshErr != nil || newToken == tokenUsed {
		return resp, nil // refresh failed or no-op — surface the original 401
	}

	resp.Body.Close()
	c.setAuth(retry)
	return c.httpClient.Do(retry)
}

// refresh obtains a new access token via refresher, serialised so that
// concurrent 401s trigger only one refresh. If another request already
// refreshed (the current token differs from tokenUsed), it returns that token
// without calling refresher again.
func (c *Client) refresh(tokenUsed string, refresher func() (string, error)) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.bearerToken != tokenUsed {
		return c.bearerToken, nil
	}
	newToken, err := refresher()
	if err != nil {
		return "", err
	}
	c.bearerToken = newToken
	return newToken, nil
}

// cloneForRetry duplicates req for a second attempt, replaying the request body
// via GetBody. It reports false when the body cannot be replayed.
func cloneForRetry(req *http.Request) (*http.Request, bool) {
	clone := req.Clone(req.Context())
	if req.Body == nil {
		return clone, true
	}
	if req.GetBody == nil {
		return nil, false
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, false
	}
	clone.Body = body
	return clone, true
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

// Workspaces returns a resource for listing accessible workspaces.
func (c *Client) Workspaces() *WorkspaceResource {
	return &WorkspaceResource{client: c}
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

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.send(req)
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
	// Do not set Accept header — the log endpoint rejects Accept: text/plain
	// and may redirect (307) to S3 where JSON Accept headers are also rejected.

	resp, err := c.send(req)
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

// doMultipart executes an authenticated POST with a raw body and explicit content-type.
// Used for multipart/form-data uploads where JSON marshaling is not appropriate.
func (c *Client) doMultipart(ctx context.Context, path string, body io.Reader, contentType string) ([]byte, error) {
	u := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, "POST", u, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)

	resp, err := c.send(req)
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

// APIError is returned for any non-2xx HTTP response from the Bitbucket API.
// Callers can use errors.As to inspect Status and Body.
type APIError struct {
	Status  int    // HTTP status code (e.g. 404, 429)
	Message string // extracted from the API's JSON error envelope when present
	Body    string // raw response body (truncated to 4KB to bound memory)
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Body)
}

// parseHTTPError converts a non-2xx HTTP response into an *APIError.
// It attempts to extract the API's JSON error envelope; falls back to the raw body.
func parseHTTPError(statusCode int, data []byte) error {
	const maxBody = 4096
	body := string(data)
	if len(body) > maxBody {
		body = body[:maxBody]
	}
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Error.Message != "" {
		return &APIError{Status: statusCode, Message: envelope.Error.Message, Body: body}
	}
	return &APIError{Status: statusCode, Body: body}
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
	req.Header.Set("Accept", "application/json")

	resp, err := c.send(req)
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
