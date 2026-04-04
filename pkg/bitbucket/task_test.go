package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestTasks_List(t *testing.T) {
	want := []bitbucket.Task{
		{ID: 1, Description: "Add tests", State: "UNRESOLVED"},
		{ID: 2, Description: "Update docs", State: "RESOLVED"},
	}
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pagelen") != "100" {
			t.Errorf("expected pagelen=100, got %s", r.URL.Query().Get("pagelen"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": want})
	}))
	got, err := c.Tasks("ws", "repo", 42).List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got))
	}
	if got[0].ID != 1 || got[1].State != "RESOLVED" {
		t.Errorf("unexpected tasks: %+v", got)
	}
}

func TestTasks_SetState_Resolved(t *testing.T) {
	var receivedBody map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		mustEncodeJSON(t, w, bitbucket.Task{ID: 1, State: "RESOLVED"})
	}))
	if err := c.Tasks("ws", "repo", 42).SetState(context.Background(), 1, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["state"] != "RESOLVED" {
		t.Errorf("expected state=RESOLVED in request body, got %v", receivedBody["state"])
	}
}

func TestTasks_SetState_Unresolved(t *testing.T) {
	var receivedBody map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		mustEncodeJSON(t, w, bitbucket.Task{ID: 1, State: "UNRESOLVED"})
	}))
	if err := c.Tasks("ws", "repo", 42).SetState(context.Background(), 1, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["state"] != "UNRESOLVED" {
		t.Errorf("expected state=UNRESOLVED in request body, got %v", receivedBody["state"])
	}
}
