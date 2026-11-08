package bitbucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	repoRunnersPath      = "/repositories/testws/testrepo/pipelines-config/runners"
	workspaceRunnersPath = "/workspaces/testws/pipelines-config/runners"
)

func TestRepoRunners_ListFollowsPaginationAndDecodes(t *testing.T) {
	hits := 0
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != repoRunnersPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") == "2" {
			mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{{"uuid": "{r2}", "name": "second"}}})
			return
		}
		mustEncodeJSON(t, w, map[string]any{
			"values": []map[string]any{{
				"uuid":       "{r1}",
				"name":       "linux-runner",
				"labels":     []string{"self.hosted", "linux"},
				"state":      map[string]any{"status": "ONLINE", "cordoned": true, "updated_on": "2026-11-01T10:00:00Z"},
				"created_on": "2026-10-01T10:00:00Z",
				"updated_on": "2026-11-01T10:00:00Z",
			}},
			"next": "http://" + r.Host + r.URL.Path + "?page=2",
		})
	}))
	got, err := client.RepoRunners("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hits != 2 || len(got) != 2 {
		t.Fatalf("expected 2 page fetches and 2 runners, got %d fetches, %d runners", hits, len(got))
	}
	first := got[0]
	if first.UUID != "{r1}" || first.Name != "linux-runner" || len(first.Labels) != 2 {
		t.Errorf("unexpected runner: %+v", first)
	}
	if first.State.Status != "ONLINE" || !first.State.Cordoned || first.CreatedOn != "2026-10-01T10:00:00Z" {
		t.Errorf("unexpected runner state/dates: %+v", first)
	}
}

func TestRepoRunners_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != repoRunnersPath+"/{r1}" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, bitbucket.PipelineRunner{UUID: "{r1}", Name: "linux-runner"})
	}))
	got, err := client.RepoRunners("testws", "testrepo").Get(context.Background(), "{r1}")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "linux-runner" {
		t.Errorf("unexpected runner: %+v", got)
	}
}

func TestRepoRunners_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != repoRunnersPath+"/{r1}" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := client.RepoRunners("testws", "testrepo").Delete(context.Background(), "{r1}"); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceRunners_ListGetDeletePaths(t *testing.T) {
	var seen []string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			if r.URL.Path == workspaceRunnersPath {
				mustEncodeJSON(t, w, map[string]any{"values": []bitbucket.PipelineRunner{{UUID: "{w1}"}}})
				return
			}
			mustEncodeJSON(t, w, bitbucket.PipelineRunner{UUID: "{w1}"})
		}
	}))
	runners := client.WorkspaceRunners("testws")
	ctx := context.Background()
	if _, err := runners.List(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := runners.Get(ctx, "{w1}"); err != nil {
		t.Fatal(err)
	}
	if err := runners.Delete(ctx, "{w1}"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET " + workspaceRunnersPath,
		"GET " + workspaceRunnersPath + "/{w1}",
		"DELETE " + workspaceRunnersPath + "/{w1}",
	}
	if len(seen) != len(want) {
		t.Fatalf("expected requests %v, got %v", want, seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("request %d: want %q, got %q", i, want[i], seen[i])
		}
	}
}
