package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestEnvironments_List(t *testing.T) {
	envs := []bitbucket.Environment{
		{
			UUID:            "{env-prod}",
			Name:            "Production",
			EnvironmentType: bitbucket.EnvironmentType{Name: "Production"},
			Lock:            bitbucket.EnvironmentLock{Name: "UNLOCKED"},
		},
		{
			UUID:            "{env-stg}",
			Name:            "Staging",
			EnvironmentType: bitbucket.EnvironmentType{Name: "Staging"},
			Lock:            bitbucket.EnvironmentLock{Name: "UNLOCKED"},
		},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/environments/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": envs})
	}))
	got, err := client.Environments("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Production" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[0].EnvironmentType.Name != "Production" {
		t.Errorf("expected EnvironmentType.Name=Production, got %s", got[0].EnvironmentType.Name)
	}
}
