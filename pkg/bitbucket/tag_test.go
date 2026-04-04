package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestTags_List(t *testing.T) {
	tags := []bitbucket.Tag{
		{Name: "v1.0.0", Target: bitbucket.BranchTarget{Hash: "abc123"}},
		{Name: "v0.9.0", Target: bitbucket.BranchTarget{Hash: "def456"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/refs/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": tags})
	}))
	got, err := client.Tags("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "v1.0.0" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestTags_Create(t *testing.T) {
	created := bitbucket.Tag{Name: "v2.0.0", Target: bitbucket.BranchTarget{Hash: "abc123"}}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/refs/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["name"] != "v2.0.0" {
			t.Errorf("expected name=v2.0.0, got %v", body["name"])
		}
		target, _ := body["target"].(map[string]any)
		if target["hash"] != "abc123" {
			t.Errorf("expected target.hash=abc123, got %v", target["hash"])
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, created)
	}))
	got, err := client.Tags("testws", "testrepo").Create(context.Background(), "v2.0.0", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "v2.0.0" {
		t.Errorf("expected v2.0.0, got %s", got.Name)
	}
	if got.Target.Hash != "abc123" {
		t.Errorf("expected target hash abc123, got %s", got.Target.Hash)
	}
}

func TestTags_Delete(t *testing.T) {
	deleted := false
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		rawPath := r.URL.RawPath
		if rawPath == "" {
			rawPath = r.URL.Path
		}
		if !strings.HasSuffix(rawPath, "release%2Fv1.0") {
			t.Errorf("expected URL-escaped tag name in path, got %s", rawPath)
		}
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Tags("testws", "testrepo").Delete(context.Background(), "release/v1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Error("delete handler was not called")
	}
}
