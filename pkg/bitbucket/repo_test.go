package bitbucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestRepos_List(t *testing.T) {
	repos := []bitbucket.Repo{
		{Slug: "whosoncall", Name: "Who's On Call", IsPrivate: true},
		{Slug: "skill-java", Name: "Java Skills", IsPrivate: true},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("role") != "member" {
			t.Errorf("expected role=member, got %s", r.URL.Query().Get("role"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": repos})
	}))
	got, err := client.Repos("testws").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Slug != "whosoncall" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestRepos_Create(t *testing.T) {
	created := bitbucket.Repo{
		Slug:      "new-repo",
		Name:      "New Repo",
		FullName:  "testws/new-repo",
		IsPrivate: true,
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/new-repo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, created)
	}))
	input := bitbucket.CreateRepoInput{
		Scm:       "git",
		Name:      "New Repo",
		IsPrivate: true,
	}
	got, err := client.Repos("testws").Create(context.Background(), "new-repo", input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "new-repo" {
		t.Errorf("expected slug new-repo, got %s", got.Slug)
	}
	if got.FullName != "testws/new-repo" {
		t.Errorf("unexpected full_name: %s", got.FullName)
	}
}

func TestRepos_Fork(t *testing.T) {
	forked := bitbucket.Repo{
		Slug:     "my-fork",
		Name:     "my-fork",
		FullName: "testws/my-fork",
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/source-repo/forks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, forked)
	}))
	got, err := client.Repos("testws").Fork(context.Background(), "source-repo", bitbucket.ForkRepoInput{Name: "my-fork"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "my-fork" {
		t.Errorf("expected slug my-fork, got %s", got.Slug)
	}
}
