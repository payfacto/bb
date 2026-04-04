package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestPRs_List(t *testing.T) {
	want := []bitbucket.PR{
		{ID: 1, Title: "feat: add login", State: "OPEN"},
		{ID: 2, Title: "fix: crash on empty", State: "OPEN"},
	}
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("state") != "OPEN" {
			t.Errorf("expected state=OPEN, got %s", r.URL.Query().Get("state"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": want})
	}))
	got, err := c.PRs("ws", "repo").List(context.Background(), "OPEN")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 2 {
		t.Errorf("unexpected PRs: %+v", got)
	}
}

func TestPRs_Get(t *testing.T) {
	want := bitbucket.PR{ID: 42, Title: "refactor: clean up", State: "OPEN"}
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(want)
	}))
	got, err := c.PRs("ws", "repo").Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 42 || got.Title != want.Title {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestPRs_Create(t *testing.T) {
	var receivedBody map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bitbucket.PR{ID: 99, Title: "feat: new feature"})
	}))
	input := bitbucket.CreatePRInput{
		Title:       "feat: new feature",
		Source:      bitbucket.NewEndpoint("feature/foo"),
		Destination: bitbucket.NewEndpoint("main"),
	}
	got, err := c.PRs("ws", "repo").Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 99 {
		t.Errorf("expected PR ID 99, got %d", got.ID)
	}
	if receivedBody["title"] != "feat: new feature" {
		t.Errorf("expected title in request body, got %v", receivedBody["title"])
	}
}

func TestPRs_Diff(t *testing.T) {
	wantDiff := "diff --git a/foo.go b/foo.go\n+added line\n"
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/x-patch")
		w.Write([]byte(wantDiff))
	}))
	got, err := c.PRs("ws", "repo").Diff(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != wantDiff {
		t.Errorf("got %q, want %q", got, wantDiff)
	}
}

func TestPRs_Approve(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]string{"state": "approved"})
	}))
	if err := c.PRs("ws", "repo").Approve(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPRs_Merge(t *testing.T) {
	var receivedBody map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		json.NewEncoder(w).Encode(map[string]string{"state": "MERGED"})
	}))
	if err := c.PRs("ws", "repo").Merge(context.Background(), 1, "squash"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["merge_strategy"] != "squash" {
		t.Errorf("expected merge_strategy=squash, got %v", receivedBody["merge_strategy"])
	}
}

func TestPRs_Decline(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(bitbucket.PR{ID: 1, State: "DECLINED"})
	}))
	if err := c.PRs("ws", "repo").Decline(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPRs_HTTPError(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"message":"repository not found"}}`))
	}))
	_, err := c.PRs("ws", "repo").Get(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}
