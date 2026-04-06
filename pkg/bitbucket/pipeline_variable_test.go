package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestPipelineVariables_List(t *testing.T) {
	vars := []bitbucket.PipelineVariable{
		{UUID: "uuid-1", Key: "API_KEY", Value: "secret", Secured: true},
		{UUID: "uuid-2", Key: "ENV", Value: "production", Secured: false},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/pipelines_config/variables/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"values": vars})
	}))
	got, err := client.PipelineVariables("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(got))
	}
	if got[0].Key != "API_KEY" || !got[0].Secured {
		t.Errorf("unexpected first variable: %+v", got[0])
	}
}

func TestPipelineVariables_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/pipelines_config/variables/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["key"] != "MY_VAR" {
			t.Errorf("expected key=MY_VAR, got %v", body["key"])
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.PipelineVariable{UUID: "uuid-3", Key: "MY_VAR", Value: "val"})
	}))
	got, err := client.PipelineVariables("testws", "testrepo").Create(context.Background(), bitbucket.CreatePipelineVariableInput{
		Key:   "MY_VAR",
		Value: "val",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "MY_VAR" {
		t.Errorf("expected key MY_VAR, got %s", got.Key)
	}
}

func TestPipelineVariables_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/pipelines_config/variables/uuid-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := client.PipelineVariables("testws", "testrepo").Delete(context.Background(), "uuid-1"); err != nil {
		t.Fatal(err)
	}
}
