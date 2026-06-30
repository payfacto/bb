package bitbucket_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestSearchCode_BuildsQueryAndDecodes(t *testing.T) {
	var gotPath, gotQuery, gotPagelen string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("search_query")
		gotPagelen = r.URL.Query().Get("pagelen")
		mustEncodeJSON(t, w, map[string]any{
			"values": []map[string]any{
				{
					"type":                "code_search_result",
					"content_match_count": 1,
					"content_matches": []map[string]any{
						{"lines": []map[string]any{
							{"line": 10, "segments": []map[string]any{
								{"text": "func "},
								{"text": "parseConfig", "match": true},
								{"text": "() {"},
							}},
						}},
					},
					"file": map[string]any{
						"path": "src/foo.go",
						"type": "commit_file",
						"commit": map[string]any{
							"hash":       "abc123",
							"repository": map[string]any{"name": "repo", "full_name": "ws/repo"},
						},
					},
				},
			},
		})
	})
	c := newTestClient(t, handler)

	res, err := c.Search("ws").Code(context.Background(), bitbucket.CodeSearchOptions{Query: "parseConfig"})
	if err != nil {
		t.Fatalf("Code: %v", err)
	}
	if gotPath != "/workspaces/ws/search/code" {
		t.Errorf("path = %q, want /workspaces/ws/search/code", gotPath)
	}
	if gotQuery != "parseConfig" {
		t.Errorf("search_query = %q, want parseConfig", gotQuery)
	}
	if gotPagelen == "" {
		t.Errorf("pagelen not set")
	}
	if len(res) != 1 {
		t.Fatalf("got %d results, want 1", len(res))
	}
	if res[0].File.Path != "src/foo.go" {
		t.Errorf("file path = %q, want src/foo.go", res[0].File.Path)
	}
	if res[0].File.Commit == nil || res[0].File.Commit.Repository == nil ||
		res[0].File.Commit.Repository.FullName != "ws/repo" {
		t.Fatalf("repository full_name not decoded: %+v", res[0].File)
	}
	if len(res[0].ContentMatches) != 1 || len(res[0].ContentMatches[0].Lines) != 1 {
		t.Fatalf("content matches not decoded: %+v", res[0].ContentMatches)
	}
	line := res[0].ContentMatches[0].Lines[0]
	if line.Line != 10 || len(line.Segments) != 3 || !line.Segments[1].Match {
		t.Errorf("line decoded wrong: %+v", line)
	}
}

func TestSearchCode_FoldsModifiers(t *testing.T) {
	var gotQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("search_query")
		mustEncodeJSON(t, w, map[string]any{"values": []any{}})
	})
	c := newTestClient(t, handler)

	_, err := c.Search("ws").Code(context.Background(), bitbucket.CodeSearchOptions{
		Query: "parseConfig", Ext: "go,mod", Lang: "go", Repo: "api", Project: "PLAT",
	})
	if err != nil {
		t.Fatalf("Code: %v", err)
	}
	want := "parseConfig ext:go ext:mod lang:go repo:api project:PLAT"
	if gotQuery != want {
		t.Errorf("search_query = %q, want %q", gotQuery, want)
	}
}

func TestSearchCode_LimitStopsPaging(t *testing.T) {
	var calls int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		// Every page returns 2 results and always advertises a next page.
		next := "http://" + r.Host + "/workspaces/ws/search/code?search_query=x&page=" + r.URL.Query().Get("nextpage")
		mustEncodeJSON(t, w, map[string]any{
			"values": []map[string]any{
				{"file": map[string]any{"path": "a.go"}},
				{"file": map[string]any{"path": "b.go"}},
			},
			"next": next,
		})
	})
	c := newTestClient(t, handler)

	res, err := c.Search("ws").Code(context.Background(), bitbucket.CodeSearchOptions{Query: "x", Limit: 3})
	if err != nil {
		t.Fatalf("Code: %v", err)
	}
	if len(res) != 3 {
		t.Errorf("got %d results, want 3 (limit)", len(res))
	}
	if calls != 2 {
		t.Errorf("made %d requests, want 2 (2 per page, stop after limit reached)", calls)
	}
}

func TestSearchCode_APIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		mustEncodeJSON(t, w, map[string]any{"error": map[string]any{"message": "no such workspace"}})
	})
	c := newTestClient(t, handler)

	_, err := c.Search("ws").Code(context.Background(), bitbucket.CodeSearchOptions{Query: "x"})
	var apiErr *bitbucket.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
		t.Fatalf("want *APIError status 404, got %v", err)
	}
}
