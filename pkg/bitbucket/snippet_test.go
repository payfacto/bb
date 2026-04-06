package bitbucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestSnippets_List(t *testing.T) {
	snippets := []bitbucket.Snippet{
		{ID: "pB7R", Title: "Hello World", IsPrivate: false},
		{ID: "qC9S", Title: "Secret Stuff", IsPrivate: true},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/snippets/testws" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"values": snippets})
	}))
	got, err := client.Snippets("testws").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 snippets, got %d", len(got))
	}
	if got[0].ID != "pB7R" {
		t.Errorf("expected first snippet ID pB7R, got %s", got[0].ID)
	}
	if !got[1].IsPrivate {
		t.Errorf("expected second snippet to be private")
	}
}

func TestSnippets_Get(t *testing.T) {
	snippet := bitbucket.Snippet{
		ID:        "pB7R",
		Title:     "Hello World",
		IsPrivate: false,
		Files:     map[string]bitbucket.SnippetFile{"hello.txt": {}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/snippets/testws/pB7R" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, snippet)
	}))
	got, err := client.Snippets("testws").Get(context.Background(), "pB7R")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "pB7R" {
		t.Errorf("expected id pB7R, got %s", got.ID)
	}
	if _, ok := got.Files["hello.txt"]; !ok {
		t.Error("expected hello.txt in files")
	}
}

func TestSnippets_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/snippets/testws/pB7R" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := client.Snippets("testws").Delete(context.Background(), "pB7R"); err != nil {
		t.Fatal(err)
	}
}
