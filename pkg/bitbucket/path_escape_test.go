package bitbucket_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// A UUID taken from user input must stay one path segment: "../" or "/"
// in it may not redirect the request to a different API resource.
func TestItemPaths_escapeUserSuppliedUUIDs(t *testing.T) {
	const hostile = "../../hooks/{x}"
	var gotPaths []string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.EscapedPath())
		mustEncodeJSON(t, w, map[string]any{})
	}))
	ctx := context.Background()
	_, _ = client.RepoRunners("ws", "repo").Get(ctx, hostile)
	_, _ = client.WorkspaceVariables("ws").Get(ctx, hostile)
	_, _ = client.PipelineVariables("ws", "repo").Get(ctx, hostile)
	_, _ = client.PipelineCaches("ws", "repo").ContentURI(ctx, hostile)
	if len(gotPaths) != 4 {
		t.Fatalf("expected 4 requests, got %v", gotPaths)
	}
	for _, p := range gotPaths {
		if strings.Contains(p, "/hooks/") || !strings.Contains(p, "..%2F..%2Fhooks%2F%7Bx%7D") {
			t.Errorf("uuid not escaped as a single segment: %s", p)
		}
	}
}
