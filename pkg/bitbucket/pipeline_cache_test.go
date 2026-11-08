package bitbucket_test

import (
	"context"
	"net/http"
	"testing"
)

const cachesPath = "/repositories/testws/testrepo/pipelines-config/caches"

func TestPipelineCaches_ListFollowsPaginationAndDecodes(t *testing.T) {
	hits := 0
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != cachesPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") == "2" {
			mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{{"uuid": "{c2}", "name": "node"}}})
			return
		}
		mustEncodeJSON(t, w, map[string]any{
			"values": []map[string]any{{
				"uuid":            "{c1}",
				"pipeline_uuid":   "{p1}",
				"step_uuid":       "{s1}",
				"name":            "gradle",
				"key_hash":        "abc123",
				"path":            "~/.gradle/caches",
				"file_size_bytes": 1048576,
				"created_on":      "2026-11-01T10:00:00Z",
			}},
			"next": "http://" + r.Host + r.URL.Path + "?page=2",
		})
	}))
	got, err := client.PipelineCaches("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hits != 2 || len(got) != 2 {
		t.Fatalf("expected 2 page fetches and 2 caches, got %d fetches, %d caches", hits, len(got))
	}
	c := got[0]
	if c.UUID != "{c1}" || c.Name != "gradle" || c.Path != "~/.gradle/caches" || c.FileSizeBytes != 1048576 {
		t.Errorf("unexpected cache: %+v", c)
	}
	if c.PipelineUUID != "{p1}" || c.StepUUID != "{s1}" || c.KeyHash != "abc123" || c.CreatedOn != "2026-11-01T10:00:00Z" {
		t.Errorf("unexpected cache provenance: %+v", c)
	}
}

func TestPipelineCaches_DeleteByName(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != cachesPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("name"); got != "node modules" {
			t.Errorf("expected name=node modules, got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := client.PipelineCaches("testws", "testrepo").DeleteByName(context.Background(), "node modules"); err != nil {
		t.Fatal(err)
	}
}

func TestPipelineCaches_ContentURI(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != cachesPath+"/{c1}/content-uri" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"uri": "https://storage.example/cache.tar.gz?sig=x"})
	}))
	got, err := client.PipelineCaches("testws", "testrepo").ContentURI(context.Background(), "{c1}")
	if err != nil {
		t.Fatal(err)
	}
	if got.URI != "https://storage.example/cache.tar.gz?sig=x" {
		t.Errorf("unexpected uri: %+v", got)
	}
}
