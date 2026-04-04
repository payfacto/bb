package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestBranches_List(t *testing.T) {
	branches := []bitbucket.Branch{
		{Name: "main", Target: bitbucket.BranchTarget{Hash: "abc123"}},
		{Name: "develop", Target: bitbucket.BranchTarget{Hash: "def456"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/refs/branches" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"values": branches})
	}))
	got, err := client.Branches("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "main" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestBranches_Create(t *testing.T) {
	created := bitbucket.Branch{Name: "feature/new", Target: bitbucket.BranchTarget{Hash: "abc123"}}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "feature/new" {
			t.Errorf("expected name=feature/new, got %v", body["name"])
		}
		target, _ := body["target"].(map[string]any)
		if target["hash"] != "abc123" {
			t.Errorf("expected target.hash=abc123, got %v", target["hash"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	}))
	got, err := client.Branches("testws", "testrepo").Create(context.Background(), "feature/new", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "feature/new" {
		t.Errorf("expected feature/new, got %s", got.Name)
	}
}

func TestBranches_Delete(t *testing.T) {
	deleted := false
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		// Check RawPath for URL-encoded characters
		rawPath := r.URL.RawPath
		if rawPath == "" {
			rawPath = r.URL.Path
		}
		if !strings.HasSuffix(rawPath, "feature%2Fold") {
			t.Errorf("expected URL-escaped branch name in path, got %s", rawPath)
		}
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Branches("testws", "testrepo").Delete(context.Background(), "feature/old")
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Error("delete handler was not called")
	}
}
