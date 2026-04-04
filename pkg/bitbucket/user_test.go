package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestUser_Me(t *testing.T) {
	user := bitbucket.User{
		AccountID:   "abc123",
		DisplayName: "Jay Madore",
		Nickname:    "jay",
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/user" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(user)
	}))
	got, err := client.User().Me(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != "Jay Madore" || got.Nickname != "jay" {
		t.Errorf("unexpected user: %+v", got)
	}
}
