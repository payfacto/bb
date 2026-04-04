package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestComments_List(t *testing.T) {
	want := []bitbucket.Comment{
		{ID: 1, Content: bitbucket.Content{Raw: "Looks good"}, User: bitbucket.Actor{DisplayName: "Alice"}},
		{ID: 2, Content: bitbucket.Content{Raw: "Please fix line 42"}, User: bitbucket.Actor{DisplayName: "Bob"}},
	}
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pagelen") != "100" {
			t.Errorf("expected pagelen=100, got %s", r.URL.Query().Get("pagelen"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": want})
	}))
	got, err := c.Comments("ws", "repo", 42).List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 {
		t.Errorf("unexpected comments: %+v", got)
	}
}

func TestComments_Add(t *testing.T) {
	var receivedBody map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		mustEncodeJSON(t, w, bitbucket.Comment{ID: 55})
	}))
	input := bitbucket.AddCommentInput{
		Content: bitbucket.Content{Raw: "Great work!"},
	}
	got, err := c.Comments("ws", "repo", 42).Add(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 55 {
		t.Errorf("expected ID 55, got %d", got.ID)
	}
	content, ok := receivedBody["content"].(map[string]any)
	if !ok || content["raw"] != "Great work!" {
		t.Errorf("unexpected request body content: %v", receivedBody)
	}
}

func TestComments_AddInline(t *testing.T) {
	var receivedBody map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		mustEncodeJSON(t, w, bitbucket.Comment{ID: 56})
	}))
	input := bitbucket.AddCommentInput{
		Content: bitbucket.Content{Raw: "Fix this line"},
		Inline:  &bitbucket.Inline{Path: "cmd/root.go", To: 42},
	}
	got, err := c.Comments("ws", "repo", 42).Add(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 56 {
		t.Errorf("expected ID 56, got %d", got.ID)
	}
	inline, ok := receivedBody["inline"].(map[string]any)
	if !ok {
		t.Fatal("expected inline field in request body")
	}
	if inline["path"] != "cmd/root.go" {
		t.Errorf("expected path=cmd/root.go, got %v", inline["path"])
	}
}

func TestComments_Reply(t *testing.T) {
	var receivedBody map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		mustEncodeJSON(t, w, bitbucket.Comment{ID: 57})
	}))
	got, err := c.Comments("ws", "repo", 42).Reply(context.Background(), 55, "Fixed!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != 57 {
		t.Errorf("expected ID 57, got %d", got.ID)
	}
	parent, ok := receivedBody["parent"].(map[string]any)
	if !ok {
		t.Fatal("expected parent field in request body")
	}
	if int(parent["id"].(float64)) != 55 {
		t.Errorf("expected parent.id=55, got %v", parent["id"])
	}
	content, ok := receivedBody["content"].(map[string]any)
	if !ok || content["raw"] != "Fixed!" {
		t.Errorf("unexpected content in request body: %v", receivedBody)
	}
}
