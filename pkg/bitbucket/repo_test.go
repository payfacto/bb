package bitbucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
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
