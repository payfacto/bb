package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestPipelines_List(t *testing.T) {
	pipelines := []bitbucket.Pipeline{
		{
			UUID:        "{abc-123}",
			BuildNumber: 42,
			State:       bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "SUCCESSFUL"}},
			Target:      bitbucket.PipelineTarget{RefType: "branch", RefName: "main"},
			CreatedOn:   "2024-01-15T10:00:00+00:00",
		},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("sort") != "-created_on" {
			t.Errorf("expected sort=-created_on, got %s", r.URL.Query().Get("sort"))
		}
		if r.URL.Query().Get("pagelen") != "25" {
			t.Errorf("expected pagelen=25, got %s", r.URL.Query().Get("pagelen"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": pipelines})
	}))
	got, err := client.Pipelines("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].BuildNumber != 42 {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestPipelines_Get(t *testing.T) {
	pipeline := bitbucket.Pipeline{UUID: "{abc-123}", BuildNumber: 42}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "{abc-123}") {
			t.Errorf("expected UUID in path, got %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, pipeline)
	}))
	got, err := client.Pipelines("testws", "testrepo").Get(context.Background(), "{abc-123}")
	if err != nil {
		t.Fatal(err)
	}
	if got.BuildNumber != 42 {
		t.Errorf("expected build 42, got %d", got.BuildNumber)
	}
}

func TestPipelines_Trigger(t *testing.T) {
	pipeline := bitbucket.Pipeline{UUID: "{new-uuid}", BuildNumber: 43}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		target, _ := body["target"].(map[string]any)
		if target["ref_name"] != "main" {
			t.Errorf("expected ref_name=main, got %v", target["ref_name"])
		}
		if target["ref_type"] != "branch" {
			t.Errorf("expected ref_type=branch, got %v", target["ref_type"])
		}
		if target["type"] != "pipeline_ref_target" {
			t.Errorf("expected type=pipeline_ref_target, got %v", target["type"])
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, pipeline)
	}))
	got, err := client.Pipelines("testws", "testrepo").Trigger(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if got.BuildNumber != 43 {
		t.Errorf("expected build 43, got %d", got.BuildNumber)
	}
}

func TestPipelines_Stop(t *testing.T) {
	stopped := false
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/stopPipeline") {
			t.Errorf("expected path to end with /stopPipeline, got %s", r.URL.Path)
		}
		stopped = true
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Pipelines("testws", "testrepo").Stop(context.Background(), "{abc-123}")
	if err != nil {
		t.Fatal(err)
	}
	if !stopped {
		t.Error("stop handler was not called")
	}
}

func TestPipelines_Steps(t *testing.T) {
	steps := []bitbucket.PipelineStep{
		{UUID: "{step-1}", Name: "build", State: bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "SUCCESSFUL"}}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/steps/") {
			t.Errorf("expected path to end with /steps/, got %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"values": steps})
	}))
	got, err := client.Pipelines("testws", "testrepo").Steps(context.Background(), "{abc-123}")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "build" {
		t.Errorf("unexpected steps: %+v", got)
	}
}

func TestPipelines_Log(t *testing.T) {
	logText := "Step 1: Building...\nStep 2: Done.\n"
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/log") {
			t.Errorf("expected path to end with /log, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		if _, err := w.Write([]byte(logText)); err != nil {
			t.Fatal(err)
		}
	}))
	got, err := client.Pipelines("testws", "testrepo").Log(context.Background(), "{abc-123}", "{step-1}")
	if err != nil {
		t.Fatal(err)
	}
	if got != logText {
		t.Errorf("expected %q, got %q", logText, got)
	}
}
