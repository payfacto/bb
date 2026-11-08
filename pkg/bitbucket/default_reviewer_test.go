package bitbucket_test

import (
	"context"
	"net/http"
	"testing"
)

func TestDefaultReviewers_ListWrapsAccountsAndPaginates(t *testing.T) {
	hits := 0
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/default-reviewers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") == "2" {
			mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{{"display_name": "Bob", "account_id": "acc-2"}}})
			return
		}
		mustEncodeJSON(t, w, map[string]any{
			"values": []map[string]any{{"display_name": "Alice", "account_id": "acc-1", "nickname": "alice"}},
			"next":   "http://" + r.Host + r.URL.Path + "?page=2",
		})
	}))
	got, err := client.DefaultReviewers("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hits != 2 || len(got) != 2 {
		t.Fatalf("expected 2 page fetches and 2 reviewers, got %d fetches, %d reviewers", hits, len(got))
	}
	if got[0].User.DisplayName != "Alice" || got[0].User.AccountID != "acc-1" || got[1].User.DisplayName != "Bob" {
		t.Errorf("unexpected reviewers: %+v", got)
	}
	if got[0].ReviewerType != "" {
		t.Errorf("plain default reviewers carry no reviewer_type, got %q", got[0].ReviewerType)
	}
}

func TestDefaultReviewers_Effective(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/effective-default-reviewers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{
			{"type": "default_reviewer", "reviewer_type": "project", "user": map[string]any{"display_name": "Patrick"}},
			{"type": "default_reviewer", "reviewer_type": "repository", "user": map[string]any{"display_name": "Davis"}},
		}})
	}))
	got, err := client.DefaultReviewers("testws", "testrepo").Effective(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 reviewers, got %d", len(got))
	}
	if got[0].ReviewerType != "project" || got[0].User.DisplayName != "Patrick" || got[1].ReviewerType != "repository" {
		t.Errorf("unexpected reviewers: %+v", got)
	}
}
