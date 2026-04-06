package bitbucket

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
)

// SnippetResource provides operations on workspace snippets.
type SnippetResource struct {
	client    *Client
	workspace string
}

func (r *SnippetResource) basePath() string {
	return fmt.Sprintf("/snippets/%s", r.workspace)
}

// List returns all snippets owned by the workspace.
func (r *SnippetResource) List(ctx context.Context) ([]Snippet, error) {
	q := url.Values{"pagelen": {pagelenDefault}}
	return fetchAllPages[Snippet](ctx, r.client, r.basePath(), q)
}

// Get returns a single snippet by its ID.
func (r *SnippetResource) Get(ctx context.Context, snippetID string) (Snippet, error) {
	path := fmt.Sprintf("%s/%s", r.basePath(), snippetID)
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return Snippet{}, err
	}
	return decode[Snippet](data)
}

// Create creates a new snippet with the given title and optional file content.
// If content is non-nil, it is uploaded as a file named filename.
func (r *SnippetResource) Create(ctx context.Context, title, filename string, private bool, content io.Reader) (Snippet, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if err := mw.WriteField("title", title); err != nil {
		return Snippet{}, fmt.Errorf("write title field: %w", err)
	}

	isPrivateVal := "false"
	if private {
		isPrivateVal = "true"
	}
	if err := mw.WriteField("is_private", isPrivateVal); err != nil {
		return Snippet{}, fmt.Errorf("write is_private field: %w", err)
	}

	if content != nil {
		name := filename
		if name == "" {
			name = "snippet.txt"
		}
		fw, err := mw.CreateFormFile("file", name)
		if err != nil {
			return Snippet{}, fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(fw, content); err != nil {
			return Snippet{}, fmt.Errorf("write file content: %w", err)
		}
	}

	if err := mw.Close(); err != nil {
		return Snippet{}, fmt.Errorf("close multipart writer: %w", err)
	}

	data, err := r.client.doMultipart(ctx, r.basePath(), &buf, mw.FormDataContentType())
	if err != nil {
		return Snippet{}, err
	}
	return decode[Snippet](data)
}

// Delete removes a snippet by its ID.
func (r *SnippetResource) Delete(ctx context.Context, snippetID string) error {
	path := fmt.Sprintf("%s/%s", r.basePath(), snippetID)
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
