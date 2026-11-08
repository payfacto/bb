package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const workspaceVarsPath = "/workspaces/testws/pipelines-config/variables"

func TestWorkspaceVariables_ListFollowsPagination(t *testing.T) {
	hits := 0
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != workspaceVarsPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") == "2" {
			mustEncodeJSON(t, w, map[string]any{"values": []bitbucket.PipelineVariable{{UUID: "uuid-2", Key: "B"}}})
			return
		}
		mustEncodeJSON(t, w, map[string]any{
			"values": []bitbucket.PipelineVariable{{UUID: "uuid-1", Key: "A", Secured: true}},
			"next":   "http://" + r.Host + r.URL.Path + "?page=2",
		})
	}))
	got, err := client.WorkspaceVariables("testws").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hits != 2 || len(got) != 2 {
		t.Fatalf("expected 2 page fetches and 2 variables, got %d fetches, %d vars", hits, len(got))
	}
	if got[0].Key != "A" || !got[0].Secured || got[1].Key != "B" {
		t.Errorf("unexpected variables: %+v", got)
	}
}

func TestWorkspaceVariables_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != workspaceVarsPath+"/uuid-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, bitbucket.PipelineVariable{UUID: "uuid-1", Key: "ENV", Value: "prod"})
	}))
	got, err := client.WorkspaceVariables("testws").Get(context.Background(), "uuid-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "ENV" || got.Value != "prod" {
		t.Errorf("unexpected variable: %+v", got)
	}
}

func TestWorkspaceVariables_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != workspaceVarsPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["key"] != "TOKEN" || body["value"] != "s3cr3t" || body["secured"] != true {
			t.Errorf("unexpected body: %+v", body)
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.PipelineVariable{UUID: "uuid-9", Key: "TOKEN", Secured: true})
	}))
	got, err := client.WorkspaceVariables("testws").Create(context.Background(), bitbucket.CreatePipelineVariableInput{
		Key: "TOKEN", Value: "s3cr3t", Secured: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "uuid-9" {
		t.Errorf("unexpected variable: %+v", got)
	}
}

func TestWorkspaceVariables_Update(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != workspaceVarsPath+"/uuid-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["key"] != "ENV" || body["value"] != "staging" || body["secured"] != false {
			t.Errorf("unexpected body: %+v", body)
		}
		mustEncodeJSON(t, w, bitbucket.PipelineVariable{UUID: "uuid-1", Key: "ENV", Value: "staging"})
	}))
	got, err := client.WorkspaceVariables("testws").Update(context.Background(), "uuid-1", bitbucket.CreatePipelineVariableInput{
		Key: "ENV", Value: "staging",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != "staging" {
		t.Errorf("unexpected variable: %+v", got)
	}
}

func TestWorkspaceVariables_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != workspaceVarsPath+"/uuid-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := client.WorkspaceVariables("testws").Delete(context.Background(), "uuid-1"); err != nil {
		t.Fatal(err)
	}
}
