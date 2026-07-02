package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
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
		mustEncodeJSON(t, w, map[string]any{"values": envs})
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

func TestEnvironments_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/environments/%7Benv-1%7D" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		mustEncodeJSON(t, w, bitbucket.Environment{UUID: "{env-1}", Name: "Staging", EnvironmentType: bitbucket.EnvironmentType{Name: "Staging"}})
	}))
	got, err := client.Environments("testws", "testrepo").Get(context.Background(), "{env-1}")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Staging" {
		t.Errorf("unexpected env: %+v", got)
	}
}

func TestEnvironments_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/environments/%7Benv-1%7D" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := client.Environments("testws", "testrepo").Delete(context.Background(), "{env-1}"); err != nil {
		t.Fatal(err)
	}
}

func TestEnvironments_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/environments/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["name"] != "QA" {
			t.Errorf("expected name=QA, got %v", body["name"])
		}
		et, ok := body["environment_type"].(map[string]any)
		if !ok || et["name"] != "Test" {
			t.Errorf("expected environment_type.name=Test, got %v", body["environment_type"])
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.Environment{UUID: "{env-9}", Name: "QA", EnvironmentType: bitbucket.EnvironmentType{Name: "Test"}})
	}))
	got, err := client.Environments("testws", "testrepo").Create(context.Background(), bitbucket.CreateEnvironmentInput{
		Name:            "QA",
		EnvironmentType: bitbucket.EnvironmentType{Name: "Test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "QA" {
		t.Errorf("expected name QA, got %s", got.Name)
	}
}
