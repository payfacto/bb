package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestIssues_List(t *testing.T) {
	issues := []bitbucket.Issue{
		{ID: 1, Title: "Login broken", State: "open", Kind: "bug", Priority: "major"},
		{ID: 2, Title: "Add dark mode", State: "new", Kind: "enhancement", Priority: "minor"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": issues})
	}))
	got, err := client.Issues("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[0].Title != "Login broken" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[1].Kind != "enhancement" {
		t.Errorf("expected Kind=enhancement, got %s", got[1].Kind)
	}
}

func TestIssues_Get(t *testing.T) {
	issue := bitbucket.Issue{
		ID:       5,
		Title:    "Fix pagination",
		State:    "open",
		Kind:     "bug",
		Priority: "critical",
		Reporter: bitbucket.Actor{DisplayName: "Alice"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/issues/5" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, issue)
	}))
	got, err := client.Issues("testws", "testrepo").Get(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 5 || got.Title != "Fix pagination" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got.Reporter.DisplayName != "Alice" {
		t.Errorf("expected Reporter=Alice, got %s", got.Reporter.DisplayName)
	}
}

func TestIssues_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["title"] != "New feature request" {
			t.Errorf("expected title=New feature request, got %v", body["title"])
		}
		if body["kind"] != "enhancement" {
			t.Errorf("expected kind=enhancement, got %v", body["kind"])
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.Issue{
			ID:    99,
			Title: "New feature request",
			State: "new",
			Kind:  "enhancement",
		})
	}))
	got, err := client.Issues("testws", "testrepo").Create(context.Background(), bitbucket.CreateIssueInput{
		Title: "New feature request",
		Kind:  "enhancement",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 99 || got.Title != "New feature request" {
		t.Errorf("unexpected result: %+v", got)
	}
}
