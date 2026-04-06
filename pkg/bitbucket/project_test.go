package bitbucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestProjects_List(t *testing.T) {
	projects := []bitbucket.Project{
		{UUID: "{uuid-1}", Key: "MARS", Name: "Mars Project", IsPrivate: true},
		{UUID: "{uuid-2}", Key: "VENUS", Name: "Venus Project", IsPrivate: false, HasPubliclyVisibleRepos: true},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/workspaces/testws/projects" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": projects})
	}))
	got, err := client.Projects("testws").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(got))
	}
	if got[0].Key != "MARS" {
		t.Errorf("expected first project key MARS, got %s", got[0].Key)
	}
	if got[1].HasPubliclyVisibleRepos != true {
		t.Errorf("expected HasPubliclyVisibleRepos=true for VENUS")
	}
}

func TestProjects_Get(t *testing.T) {
	project := bitbucket.Project{
		UUID:        "{uuid-1}",
		Key:         "MARS",
		Name:        "Mars Project",
		Description: "The red planet project",
		IsPrivate:   true,
		CreatedOn:   "2024-01-01T00:00:00Z",
		UpdatedOn:   "2024-06-01T00:00:00Z",
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/workspaces/testws/projects/MARS" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, project)
	}))
	got, err := client.Projects("testws").Get(context.Background(), "MARS")
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "MARS" {
		t.Errorf("expected key MARS, got %s", got.Key)
	}
	if got.Description != "The red planet project" {
		t.Errorf("unexpected description: %s", got.Description)
	}
}

func TestRepos_ListByProject(t *testing.T) {
	repos := []bitbucket.Repo{
		{Slug: "alpha", Name: "Alpha", IsPrivate: true, Project: &bitbucket.ProjectRef{Key: "MARS", Name: "Mars Project"}},
		{Slug: "beta", Name: "Beta", IsPrivate: false, Project: &bitbucket.ProjectRef{Key: "MARS", Name: "Mars Project"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if q := r.URL.Query().Get("q"); q != `project.key="MARS"` {
			t.Errorf("unexpected q param: %s", q)
		}
		mustEncodeJSON(t, w, map[string]any{"values": repos})
	}))
	got, err := client.Repos("testws").ListByProject(context.Background(), "MARS")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(got))
	}
	if got[0].Project == nil || got[0].Project.Key != "MARS" {
		t.Errorf("expected project key MARS on first repo")
	}
}
