package bitbucket_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestMembers_List(t *testing.T) {
	members := []bitbucket.WorkspaceMember{
		{User: bitbucket.User{AccountID: "acc-1", DisplayName: "Alice Smith", Nickname: "alice"}},
		{User: bitbucket.User{AccountID: "acc-2", DisplayName: "Bob Jones", Nickname: "bob"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/workspaces/testws/members" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": members})
	}))
	got, err := client.Members("testws").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].User.DisplayName != "Alice Smith" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[0].User.Nickname != "alice" {
		t.Errorf("expected Nickname=alice, got %s", got[0].User.Nickname)
	}
}
