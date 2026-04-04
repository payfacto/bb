package bitbucket

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
)

// DownloadResource provides operations on repository downloads.
type DownloadResource struct {
	client    *Client
	workspace string
	repo      string
}

func (r *DownloadResource) basePath() string {
	return fmt.Sprintf("%s/downloads", repoPath(r.workspace, r.repo))
}

// List returns all downloads attached to the repository.
func (r *DownloadResource) List(ctx context.Context) ([]Download, error) {
	q := url.Values{"pagelen": {"50"}}
	data, err := r.client.do(ctx, "GET", r.basePath(), nil, q)
	if err != nil {
		return nil, err
	}
	page, err := decode[paged[Download]](data)
	if err != nil {
		return nil, err
	}
	return page.Values, nil
}

// Upload uploads a file as a repository download artifact.
// name is the filename as it will appear in Bitbucket (typically filepath.Base of the local path).
// content is the file data to upload.
func (r *DownloadResource) Upload(ctx context.Context, name string, content io.Reader) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("files", name)
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, content); err != nil {
		return err
	}
	w.Close()
	_, err = r.client.doMultipart(ctx, r.basePath(), &buf, w.FormDataContentType())
	return err
}

// Delete removes a download artifact by filename.
func (r *DownloadResource) Delete(ctx context.Context, filename string) error {
	path := fmt.Sprintf("%s/%s", r.basePath(), url.PathEscape(filename))
	_, err := r.client.do(ctx, "DELETE", path, nil, nil)
	return err
}
