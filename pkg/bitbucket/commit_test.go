package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestCommits_List(t *testing.T) {
	commits := []bitbucket.Commit{
		{Hash: "abc123", Message: "Fix bug", Author: bitbucket.CommitAuthor{Raw: "Jay <jay@example.com>"}, Date: "2024-01-15T10:00:00+00:00"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/commits/main" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"values": commits})
	}))
	got, err := client.Commits("testws", "testrepo").List(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Hash != "abc123" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestCommits_Get(t *testing.T) {
	commit := bitbucket.Commit{Hash: "abc123", Message: "Fix bug"}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories/testws/testrepo/commit/abc123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(commit)
	}))
	got, err := client.Commits("testws", "testrepo").Get(context.Background(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash != "abc123" {
		t.Errorf("expected abc123, got %s", got.Hash)
	}
}

func TestCommits_File(t *testing.T) {
	fileContent := "package main\n\nfunc main() {}\n"
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories/testws/testrepo/src/main/main.go" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(fileContent))
	}))
	got, err := client.Commits("testws", "testrepo").File(context.Background(), "main", "main.go")
	if err != nil {
		t.Fatal(err)
	}
	if got != fileContent {
		t.Errorf("expected %q, got %q", fileContent, got)
	}
}
