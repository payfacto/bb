package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestEnvironmentVariables_List(t *testing.T) {
	vars := []bitbucket.PipelineVariable{
		{UUID: "{v-1}", Key: "API_URL", Value: "https://x", Secured: false},
		{UUID: "{v-2}", Key: "SECRET", Value: "", Secured: true},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/deployments_config/environments/%7Benv-1%7D/variables/" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		mustEncodeJSON(t, w, map[string]any{"values": vars})
	}))
	got, err := client.EnvironmentVariables("testws", "testrepo", "{env-1}").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Key != "SECRET" || !got[1].Secured {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestEnvironmentVariables_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/deployments_config/environments/%7Benv-1%7D/variables/" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["key"] != "API_URL" || body["value"] != "https://x" {
			t.Errorf("unexpected body: %+v", body)
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.PipelineVariable{UUID: "{v-9}", Key: "API_URL", Value: "https://x"})
	}))
	got, err := client.EnvironmentVariables("testws", "testrepo", "{env-1}").Create(context.Background(), bitbucket.CreatePipelineVariableInput{
		Key:   "API_URL",
		Value: "https://x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "API_URL" {
		t.Errorf("expected key API_URL, got %s", got.Key)
	}
}

func TestEnvironmentVariables_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.EscapedPath() != "/repositories/testws/testrepo/deployments_config/environments/%7Benv-1%7D/variables/%7Bv-1%7D" {
			t.Errorf("unexpected path: %s", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := client.EnvironmentVariables("testws", "testrepo", "{env-1}").Delete(context.Background(), "{v-1}"); err != nil {
		t.Fatal(err)
	}
}
