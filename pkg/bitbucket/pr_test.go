package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
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
		mustEncodeJSON(t, w, map[string]any{"values": want})
	}))
	got, err := c.PRs("ws", "repo").List(context.Background(), bitbucket.PRListOptions{State: "OPEN"})
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

func TestPRs_List_SourceBranchFilter(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got, want := q.Get("state"), "OPEN"; got != want {
			t.Errorf("state: got %q, want %q", got, want)
		}
		if got, want := q.Get("q"), `source.branch.name="feat/x"`; got != want {
			t.Errorf("q: got %q, want %q", got, want)
		}
		mustEncodeJSON(t, w, map[string]any{"values": []bitbucket.PR{{ID: 7}}})
	}))
	got, err := c.PRs("ws", "repo").List(context.Background(), bitbucket.PRListOptions{State: "OPEN", SourceBranch: "feat/x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 7 {
		t.Errorf("unexpected PRs: %+v", got)
	}
}

func TestPRs_List_FollowsPaginationAndDecodesDates(t *testing.T) {
	var hits int
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Query().Get("page") == "2" {
			mustEncodeJSON(t, w, map[string]any{"values": []bitbucket.PR{{ID: 3}}})
			return
		}
		// First page: advertise a next link back to this server.
		next := "http://" + r.Host + r.URL.Path + "?page=2"
		mustEncodeJSON(t, w, map[string]any{
			"next": next,
			"values": []bitbucket.PR{
				{ID: 1, CreatedOn: "2025-01-02T00:00:00+00:00", UpdatedOn: "2025-01-03T00:00:00+00:00", CommentCount: 4, TaskCount: 1},
				{ID: 2},
			},
		})
	}))
	got, err := c.PRs("ws", "repo").List(context.Background(), bitbucket.PRListOptions{State: "MERGED"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits != 2 {
		t.Fatalf("expected 2 page fetches, got %d", hits)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 PRs across pages, got %d", len(got))
	}
	if got[0].CreatedOn != "2025-01-02T00:00:00+00:00" || got[0].UpdatedOn != "2025-01-03T00:00:00+00:00" {
		t.Errorf("dates not decoded: %+v", got[0])
	}
	if got[0].CommentCount != 4 || got[0].TaskCount != 1 {
		t.Errorf("counts not decoded: %+v", got[0])
	}
}

func TestPRs_List_DateRangeFilter(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		for _, want := range []string{
			`source.branch.name="feat/x"`,
			`created_on>="2023-01-01T00:00:00+00:00"`,
			`created_on<="2024-01-01T00:00:00+00:00"`,
		} {
			if !strings.Contains(q, want) {
				t.Errorf("q %q missing clause %q", q, want)
			}
		}
		mustEncodeJSON(t, w, map[string]any{"values": []bitbucket.PR{{ID: 9}}})
	}))
	_, err := c.PRs("ws", "repo").List(context.Background(), bitbucket.PRListOptions{
		State:        "MERGED",
		SourceBranch: "feat/x",
		Since:        "2023-01-01T00:00:00+00:00",
		Until:        "2024-01-01T00:00:00+00:00",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPRs_Create_Draft(t *testing.T) {
	var receivedBody map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.PR{ID: 1})
	}))
	input := bitbucket.CreatePRInput{
		Title:       "draft pr",
		Source:      bitbucket.NewEndpoint("feature/foo"),
		Destination: bitbucket.NewEndpoint("main"),
		Draft:       true,
	}
	if _, err := c.PRs("ws", "repo").Create(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := receivedBody["draft"], true; got != want {
		t.Errorf("draft: got %v, want %v", got, want)
	}
}

func TestPRs_Get(t *testing.T) {
	want := bitbucket.PR{ID: 42, Title: "refactor: clean up", State: "OPEN"}
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mustEncodeJSON(t, w, want)
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
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.PR{ID: 99, Title: "feat: new feature"})
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
		if _, err := w.Write([]byte(wantDiff)); err != nil {
			t.Fatal(err)
		}
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
		mustEncodeJSON(t, w, map[string]string{"state": "approved"})
	}))
	if err := c.PRs("ws", "repo").Approve(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPRs_Merge(t *testing.T) {
	var receivedBody map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		mustEncodeJSON(t, w, map[string]string{"state": "MERGED"})
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
		mustEncodeJSON(t, w, bitbucket.PR{ID: 1, State: "DECLINED"})
	}))
	if err := c.PRs("ws", "repo").Decline(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPRs_HTTPError(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte(`{"error":{"message":"repository not found"}}`)); err != nil {
			t.Fatal(err)
		}
	}))
	_, err := c.PRs("ws", "repo").Get(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}

func TestPRs_Activity(t *testing.T) {
	activities := []bitbucket.Activity{
		{Approval: &bitbucket.Approval{User: bitbucket.Actor{DisplayName: "Jane"}, Date: "2024-01-15T10:00:00+00:00"}},
		{Comment: &bitbucket.Comment{ID: 1, Content: bitbucket.Content{Raw: "LGTM"}, User: bitbucket.Actor{DisplayName: "Bob"}}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories/testws/testrepo/pullrequests/42/activity" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"values": activities})
	}))
	got, err := client.PRs("testws", "testrepo").Activity(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(got))
	}
	if got[0].Approval == nil || got[0].Approval.User.DisplayName != "Jane" {
		t.Errorf("unexpected activity[0]: %+v", got[0])
	}
}

func TestPRs_Statuses(t *testing.T) {
	statuses := []bitbucket.PRStatus{
		{State: "SUCCESSFUL", Key: "bitbucket-pipelines", Name: "Build", Description: "Pipeline passed"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories/testws/testrepo/pullrequests/42/statuses" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"values": statuses})
	}))
	got, err := client.PRs("testws", "testrepo").Statuses(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].State != "SUCCESSFUL" {
		t.Errorf("unexpected statuses: %+v", got)
	}
}

func TestPRList_QueryFiltersTitleAndDescription(t *testing.T) {
	var gotQ string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQ = r.URL.Query().Get("q")
		mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{{"id": 1, "title": "fix login"}}})
	})
	c := newTestClient(t, handler)

	prs, err := c.PRs("ws", "repo").List(context.Background(), bitbucket.PRListOptions{Query: "login"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := `(title ~ "login" OR description ~ "login")`
	if gotQ != want {
		t.Errorf("q = %q, want %q", gotQ, want)
	}
	if len(prs) != 1 {
		t.Fatalf("got %d PRs, want 1", len(prs))
	}
}
