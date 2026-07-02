package bitbucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestDeployments_Get(t *testing.T) {
	dep := bitbucket.Deployment{
		UUID:        "{dep-1}",
		State:       bitbucket.DeploymentState{Name: "COMPLETED", Status: &bitbucket.DeploymentStatus{Name: "SUCCESSFUL"}},
		Environment: bitbucket.DeploymentEnvRef{UUID: "{env-prod}"},
		Deployable: bitbucket.Deployable{
			Commit:   &bitbucket.DeployableCommit{Hash: "abc123"},
			Pipeline: &bitbucket.DeployablePipeline{UUID: "{pipe-1}"},
		},
		LastUpdateTime: "2024-01-15T10:00:00+00:00",
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/deployments/{dep-1}" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, dep)
	}))
	got, err := client.Deployments("testws", "testrepo").Get(context.Background(), "{dep-1}")
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{dep-1}" || got.State.Status == nil || got.State.Status.Name != "SUCCESSFUL" {
		t.Errorf("unexpected deployment: %+v", got)
	}
}

func TestDeployments_List(t *testing.T) {
	deployments := []bitbucket.Deployment{
		{
			UUID:        "{dep-1}",
			State:       bitbucket.DeploymentState{Name: "COMPLETED", Status: &bitbucket.DeploymentStatus{Name: "SUCCESSFUL"}},
			Environment: bitbucket.DeploymentEnvRef{UUID: "{env-prod}"},
			Deployable: bitbucket.Deployable{
				Commit:   &bitbucket.DeployableCommit{Hash: "abc123"},
				Pipeline: &bitbucket.DeployablePipeline{UUID: "{pipe-1}"},
			},
			LastUpdateTime: "2024-01-15T10:00:00+00:00",
		},
		{
			UUID:           "{dep-2}",
			State:          bitbucket.DeploymentState{Name: "IN_PROGRESS"},
			Environment:    bitbucket.DeploymentEnvRef{UUID: "{env-stg}"},
			Deployable:     bitbucket.Deployable{},
			LastUpdateTime: "2024-01-14T09:00:00+00:00",
		},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/deployments/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "25" {
			t.Errorf("expected pagelen=25, got %s", r.URL.Query().Get("pagelen"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": deployments})
	}))
	got, err := client.Deployments("testws", "testrepo").List(context.Background(), bitbucket.DeploymentListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].UUID != "{dep-1}" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[0].State.Status == nil || got[0].State.Status.Name != "SUCCESSFUL" {
		t.Errorf("expected State.Status.Name=SUCCESSFUL, got %+v", got[0].State.Status)
	}
	if got[0].Deployable.Commit == nil || got[0].Deployable.Commit.Hash != "abc123" {
		t.Errorf("expected Deployable.Commit.Hash=abc123, got %+v", got[0].Deployable.Commit)
	}
}

func TestDeployments_List_EnvFilterAndSort(t *testing.T) {
	// The Bitbucket deployments endpoint silently ignores q= BBQL filters, so
	// no q param is sent; env filtering is done client-side after the fetch.
	all := []bitbucket.Deployment{
		{UUID: "{dep-1}", Environment: bitbucket.DeploymentEnvRef{UUID: "{env-prod}"}},
		{UUID: "{dep-2}", Environment: bitbucket.DeploymentEnvRef{UUID: "{env-stg}"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("q"); got != "" {
			t.Errorf("expected no q param, got %q", got)
		}
		if got := q.Get("sort"); got != "-last_update_time" {
			t.Errorf("unexpected sort: %q", got)
		}
		mustEncodeJSON(t, w, map[string]any{"values": all})
	}))
	got, err := client.Deployments("testws", "testrepo").List(context.Background(), bitbucket.DeploymentListOptions{
		EnvUUID: "{env-prod}",
		Sort:    "-last_update_time",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].UUID != "{dep-1}" {
		t.Errorf("expected client-side filter to return only env-prod deployment, got %+v", got)
	}
}
