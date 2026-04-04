# BB CLI Downloads Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `bb download list/upload/delete` to manage binary file releases attached to a Bitbucket repository.

**Architecture:** `DownloadResource` is workspace+repo scoped — same shape as `TagResource`. Upload requires multipart/form-data (not JSON), so a new `doMultipart` helper is added to `client.go`. The cmd layer follows the `cmd/tag.go` pattern using `workspaceAndRepo()`.

**Tech Stack:** Go 1.25, `github.com/spf13/cobra v1.10.2`, `mime/multipart` (stdlib), `net/http/httptest` (stdlib)

---

## File Structure

**New files:**

| File | Responsibility |
|------|---------------|
| `pkg/bitbucket/download.go` | DownloadResource: List, Upload, Delete |
| `pkg/bitbucket/download_test.go` | Tests for DownloadResource |
| `cmd/download.go` | `bb download list/upload/delete` |

**Modified files:**

| File | Change |
|------|--------|
| `pkg/bitbucket/types.go` | Append `Download` type |
| `pkg/bitbucket/client.go` | Add `doMultipart` helper + `Downloads()` accessor |

---

### Task 1: Add Download type to pkg/bitbucket/types.go

**Files:**
- Modify: `pkg/bitbucket/types.go`

The `Links` type already exists. `Download` is a simple value type.

- [ ] **Step 1: Append Download type to types.go**

Open `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/types.go` and append at the end of the file:

```go
// Download type

type Download struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Links     Links  `json:"links"`
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go build ./...
```

Expected: No output (no errors).

- [ ] **Step 3: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/types.go
git commit -m "feat: add Download type"
```

---

### Task 2: DownloadResource + doMultipart + accessor + tests

**Files:**
- Create: `pkg/bitbucket/download_test.go`
- Create: `pkg/bitbucket/download.go`
- Modify: `pkg/bitbucket/client.go`

**Bitbucket API endpoints:**
- `GET /repositories/{workspace}/{repo}/downloads?pagelen=50` — list downloads
- `POST /repositories/{workspace}/{repo}/downloads` — upload (multipart/form-data, field name: `files`)
- `DELETE /repositories/{workspace}/{repo}/downloads/{filename}` — delete by filename

The upload endpoint uses `multipart/form-data` rather than JSON. The existing `do()` method only handles JSON bodies. A `doMultipart` helper is added to `client.go` to handle the POST with a raw reader and explicit content-type.

- [ ] **Step 1: Create download_test.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/download_test.go`:

```go
package bitbucket_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestDownloads_List(t *testing.T) {
	downloads := []bitbucket.Download{
		{Name: "app-v1.0.0.zip", Size: 1048576},
		{Name: "app-v1.1.0.zip", Size: 2097152},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/downloads" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": downloads})
	}))
	got, err := client.Downloads("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "app-v1.0.0.zip" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[1].Size != 2097152 {
		t.Errorf("expected Size=2097152, got %d", got[1].Size)
	}
}

func TestDownloads_Upload(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/downloads" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		f, header, err := r.FormFile("files")
		if err != nil {
			t.Fatalf("get form file 'files': %v", err)
		}
		defer f.Close()
		if header.Filename != "release.zip" {
			t.Errorf("expected filename=release.zip, got %s", header.Filename)
		}
		content, _ := io.ReadAll(f)
		if string(content) != "binary content" {
			t.Errorf("unexpected content: %s", content)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	err := client.Downloads("testws", "testrepo").Upload(context.Background(), "release.zip", strings.NewReader("binary content"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestDownloads_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/downloads/app-v1.0.0.zip" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Downloads("testws", "testrepo").Delete(context.Background(), "app-v1.0.0.zip")
	if err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestDownloads -v 2>&1 | head -10
```

Expected: Compilation error — `client.Downloads undefined`.

- [ ] **Step 3: Create download.go**

Create `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/download.go`:

```go
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
```

- [ ] **Step 4: Add doMultipart helper and Downloads() accessor to client.go**

In `/Users/jay/code/cli-bitbucket-cloud/pkg/bitbucket/client.go`, add the `doMultipart` helper after the `do()` method (after line 154, before the `decode` function):

```go
// doMultipart executes an authenticated POST with a raw body and explicit content-type.
// Used for multipart/form-data uploads where JSON marshaling is not appropriate.
func (c *Client) doMultipart(ctx context.Context, path string, body io.Reader, contentType string) ([]byte, error) {
	u := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, "POST", u, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.token)
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
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}
```

Also add the `Downloads()` accessor after the existing `Members()` method (after line 103):

```go
// Downloads returns a resource for download artifact operations on the given repo.
func (c *Client) Downloads(workspace, repo string) *DownloadResource {
	return &DownloadResource{client: c, workspace: workspace, repo: repo}
}
```

Note: `client.go` already imports `"io"` (used in `do()` for `io.Reader`, `io.ReadAll`) — no new imports are needed.

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -run TestDownloads -count=1 -v
```

Expected: `TestDownloads_List`, `TestDownloads_Upload`, `TestDownloads_Delete` all PASS.

- [ ] **Step 6: Run full test suite**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go test ./pkg/bitbucket/... -count=1
```

Expected: All tests pass.

- [ ] **Step 7: Commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
git add pkg/bitbucket/download.go pkg/bitbucket/download_test.go pkg/bitbucket/client.go
git commit -m "feat: add DownloadResource (list, upload, delete)"
```

---

### Task 3: cmd/download.go

**Files:**
- Create: `cmd/download.go`

**Commands:** `bb download list`, `bb download upload --file <path>`, `bb download delete --name <filename>`

These commands need workspace and repo. Follow the `cmd/tag.go` pattern: use `workspaceAndRepo()`.

Text output for list: one download per line, filename left-padded to 40 chars, size in bytes right-padded to 12.
Text output for upload: `"Uploaded '<filename>'.\n"`
Text output for delete: `"Download '<filename>' deleted.\n"`

`truncate()` is defined in `cmd/comment.go` and is available to all files in the `cmd` package.

- [ ] **Step 1: Create cmd/download.go**

Create `/Users/jay/code/cli-bitbucket-cloud/cmd/download.go`:

```go
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Manage repository download artifacts",
}

var downloadListCmd = &cobra.Command{
	Use:   "list",
	Short: "List download artifacts in the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		downloads, err := client.Downloads(ws, repo).List(context.Background())
		if err != nil {
			return err
		}
		return printOutput(downloads, func() {
			if len(downloads) == 0 {
				fmt.Println("No downloads found.")
				return
			}
			for _, d := range downloads {
				fmt.Printf("%-40s  %d bytes\n", truncate(d.Name, 40), d.Size)
			}
		})
	},
}

var downloadUploadFile string

var downloadUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file as a download artifact",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		f, err := os.Open(downloadUploadFile)
		if err != nil {
			return err
		}
		defer f.Close()
		name := filepath.Base(downloadUploadFile)
		if err := client.Downloads(ws, repo).Upload(context.Background(), name, f); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "uploaded", "file": name}, func() {
			fmt.Printf("Uploaded '%s'.\n", name)
		})
	},
}

var downloadDeleteName string

var downloadDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a download artifact by filename",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, repo, err := workspaceAndRepo()
		if err != nil {
			return err
		}
		if err := client.Downloads(ws, repo).Delete(context.Background(), downloadDeleteName); err != nil {
			return err
		}
		return printOutput(map[string]string{"result": "deleted", "name": downloadDeleteName}, func() {
			fmt.Printf("Download '%s' deleted.\n", downloadDeleteName)
		})
	},
}

func init() {
	downloadUploadCmd.Flags().StringVar(&downloadUploadFile, "file", "", "path to the file to upload (required)")
	downloadUploadCmd.MarkFlagRequired("file")

	downloadDeleteCmd.Flags().StringVar(&downloadDeleteName, "name", "", "filename to delete (required)")
	downloadDeleteCmd.MarkFlagRequired("name")

	downloadCmd.AddCommand(downloadListCmd, downloadUploadCmd, downloadDeleteCmd)
	rootCmd.AddCommand(downloadCmd)
}
```

- [ ] **Step 2: Build to verify compilation**

```bash
cd /Users/jay/code/cli-bitbucket-cloud && go build -o bb .
```

Expected: No output (binary built).

- [ ] **Step 3: Smoke tests**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
./bb download --help
./bb download list --help
./bb download upload --help
./bb --help | grep download
```

Expected for `./bb download --help`:
```
Manage repository download artifacts

Usage:
  bb download [command]

Available Commands:
  delete      Delete a download artifact by filename
  list        List download artifacts in the repository
  upload      Upload a file as a download artifact
```

Expected for `./bb download upload --help` (confirms --file flag):
```
  --file string   path to the file to upload (required)
```

Expected for `./bb --help | grep download`: `  download    Manage repository download artifacts`

- [ ] **Step 4: Clean up and commit**

```bash
cd /Users/jay/code/cli-bitbucket-cloud
rm bb
git add cmd/download.go
git commit -m "feat: add bb download commands (list, upload, delete)"
```

---

## Self-Review

**Spec coverage:**
- `bb download list` → Task 3 `downloadListCmd` ✅
- `bb download upload` → Task 3 `downloadUploadCmd` ✅
- `bb download delete` → Task 3 `downloadDeleteCmd` ✅
- `DownloadResource.List` → Task 2 ✅
- `DownloadResource.Upload` → Task 2 ✅
- `DownloadResource.Delete` → Task 2 ✅
- `Download` type → Task 1 ✅
- `Downloads()` accessor on `*Client` → Task 2 Step 4 ✅
- `doMultipart` helper → Task 2 Step 4 ✅

**Placeholder scan:** No TBDs. All code blocks complete.

**Type consistency:**
- `client.Downloads("testws", "testrepo")` returns `*DownloadResource` — used in tests and cmd ✅
- `Download.Name` (string), `Download.Size` (int64) — accessed as `d.Name`, `d.Size` in cmd ✅
- `DownloadResource.Upload(ctx, name string, content io.Reader) error` — cmd passes `filepath.Base(path)` + open file ✅
- `truncate(d.Name, 40)` — `truncate` defined in `cmd/comment.go`, available to all `cmd` package files ✅
- `doMultipart` — added to `client.go` which already imports `"io"` (no new import needed in client.go) ✅
