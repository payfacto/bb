package bitbucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const mergeabilityChecksBody = `{
  "size": 4,
  "values": [
    {"type": "pullrequest_state_check", "status": "PASSED", "required": true, "blocking": false, "state": "OPEN"},
    {"type": "git_mergeability_check", "status": "FAILED", "required": true, "blocking": true, "reason": "conflicts",
     "links": {"details": {"href": "https://api.bitbucket.org/2.0/repositories/ws/repo/pullrequests/7/conflicts"}}},
    {"type": "standard_merge_check", "status": "FAILED", "required": false, "blocking": false,
     "check": {"type": "standard_merge_check_definition", "kind": "minimum_approvals"},
     "requirement": {"minimum_approvals": 2}, "observed": {"approval_count": 1}},
    {"type": "merge_queue_check", "status": "PASSED", "required": true, "blocking": false, "queued": true,
     "merge_queue": {"uuid": "{q1}", "name": "main-queue", "state": "ACTIVE"}}
  ]
}`

func TestPRs_MergeabilityChecks_DecodesChecksAndDerivesMergeable(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repositories/ws/repo/pullrequests/7/mergeability/checks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		_, _ = w.Write([]byte(mergeabilityChecksBody))
	}))

	got, err := client.PRs("ws", "repo").MergeabilityChecks(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Mergeable {
		t.Error("Mergeable = true, want false (git check is blocking)")
	}
	if len(got.Checks) != 4 {
		t.Fatalf("len(Checks) = %d, want 4", len(got.Checks))
	}
	assertCheck(t, got.Checks[0], "pullrequest_state_check", "PASSED", false)
	if got.Checks[0].State != "OPEN" {
		t.Errorf("state check State = %q, want OPEN", got.Checks[0].State)
	}
	assertCheck(t, got.Checks[1], "git_mergeability_check", "FAILED", true)
	if got.Checks[1].Reason != "conflicts" {
		t.Errorf("git check Reason = %q, want conflicts", got.Checks[1].Reason)
	}
	standard := got.Checks[2]
	if standard.Check == nil || standard.Check.Kind != "minimum_approvals" {
		t.Errorf("standard check definition = %+v, want kind minimum_approvals", standard.Check)
	}
	if standard.Requirement["minimum_approvals"] != float64(2) || standard.Observed["approval_count"] != float64(1) {
		t.Errorf("standard check requirement/observed = %v / %v", standard.Requirement, standard.Observed)
	}
	queue := got.Checks[3]
	if !queue.Queued || queue.MergeQueue == nil || queue.MergeQueue.Name != "main-queue" {
		t.Errorf("merge queue check = %+v", queue)
	}
}

func TestPRs_MergeabilityChecks_MergeableWhenNothingBlocks(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"values": [
			{"type": "current_user_permission_check", "status": "PASSED", "required": true, "blocking": false},
			{"type": "custom_pre_merge_check", "status": "FAILED", "required": false, "blocking": false,
			 "check": {"type": "custom_merge_check_definition", "id": "ari:x", "name": "Lint"}, "message": "warn only"}
		]}`))
	}))

	got, err := client.PRs("ws", "repo").MergeabilityChecks(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Mergeable {
		t.Error("Mergeable = false, want true (no blocking checks)")
	}
	if got.Checks[1].Check == nil || got.Checks[1].Check.Name != "Lint" || got.Checks[1].Message != "warn only" {
		t.Errorf("custom check = %+v", got.Checks[1])
	}
}

func TestPRs_MergeabilityChecks_EmptyChecksIsNonNilSlice(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"size": 0, "values": []}`))
	}))

	got, err := client.PRs("ws", "repo").MergeabilityChecks(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Checks == nil || !got.Mergeable {
		t.Errorf("got %#v, want mergeable with empty non-nil checks", got)
	}
}

func TestPRs_MergeabilityChecks_PropagatesAPIError(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error": {"message": "custom checks unavailable"}}`, http.StatusInternalServerError)
	}))

	_, err := client.PRs("ws", "repo").MergeabilityChecks(context.Background(), 7)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPRs_Conflicts_FollowsRedirectWithAuthAndPaginates(t *testing.T) {
	const filePath = "/repositories/ws/repo/file-conflicts/abc..def"
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); !ok || user != "testuser" || pass != "testtoken" {
			t.Errorf("missing auth on %s", r.URL)
		}
		switch {
		case r.URL.Path == "/repositories/ws/repo/pullrequests/7/conflicts":
			http.Redirect(w, r, filePath, http.StatusFound)
		case r.URL.Path == filePath && r.URL.Query().Get("page") == "":
			mustEncodeJSON(t, w, map[string]any{
				"values": []map[string]string{{"type": "file_conflict", "path": "README.md", "scenario": "content", "message": "modified in both"}},
				"next":   "http://" + r.Host + filePath + "?page=2",
			})
		case r.URL.Path == filePath && r.URL.Query().Get("page") == "2":
			mustEncodeJSON(t, w, map[string]any{
				"values": []map[string]string{{"type": "file_conflict", "path": "main.go", "scenario": "delete_modify"}},
			})
		default:
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
		}
	}))

	got, err := client.PRs("ws", "repo").Conflicts(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	want := []bitbucket.FileConflict{
		{Path: "README.md", Scenario: "content", Message: "modified in both"},
		{Path: "main.go", Scenario: "delete_modify"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d conflicts, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("conflict[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPRs_Conflicts_EmptyReturnsEmptySlice(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mustEncodeJSON(t, w, map[string]any{"values": []any{}})
	}))

	got, err := client.PRs("ws", "repo").Conflicts(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("got %#v, want empty non-nil slice", got)
	}
}

func assertCheck(t *testing.T, c bitbucket.MergeabilityCheck, wantType, wantStatus string, wantBlocking bool) {
	t.Helper()
	if c.Type != wantType || c.Status != wantStatus || c.Blocking != wantBlocking {
		t.Errorf("check = {%s %s blocking=%v}, want {%s %s blocking=%v}",
			c.Type, c.Status, c.Blocking, wantType, wantStatus, wantBlocking)
	}
}
