package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestDeployKeys_List(t *testing.T) {
	keys := []bitbucket.DeployKey{
		{ID: 1, Label: "CI server", Key: "ssh-rsa AAAAB3NzaC1yc2E ci@example.com"},
		{ID: 2, Label: "Deploy bot", Key: "ssh-rsa AAAAB3NzaC1yc2E bot@example.com"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/deploy-keys" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": keys})
	}))
	got, err := client.DeployKeys("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[0].Label != "CI server" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got[1].ID != 2 {
		t.Errorf("expected ID=2, got %d", got[1].ID)
	}
}

func TestDeployKeys_Add(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/deploy-keys" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["label"] != "My Key" {
			t.Errorf("expected label=My Key, got %s", body["label"])
		}
		if body["key"] != "ssh-rsa AAAAB3NzaC1yc2E test@example.com" {
			t.Errorf("unexpected key: %s", body["key"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bitbucket.DeployKey{
			ID:    42,
			Label: "My Key",
			Key:   "ssh-rsa AAAAB3NzaC1yc2E test@example.com",
		})
	}))
	got, err := client.DeployKeys("testws", "testrepo").Add(context.Background(), "My Key", "ssh-rsa AAAAB3NzaC1yc2E test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 42 || got.Label != "My Key" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestDeployKeys_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/deploy-keys/7" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.DeployKeys("testws", "testrepo").Delete(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
}
