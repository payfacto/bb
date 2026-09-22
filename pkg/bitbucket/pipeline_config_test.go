package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestPipelineConfig_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/pipelines_config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"enabled": true, "repository": map[string]any{"full_name": "testws/testrepo"}})
	}))
	got, err := client.PipelineConfig("testws", "testrepo").Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled {
		t.Errorf("expected enabled=true, got %+v", got)
	}
}

func TestPipelineConfig_Enable(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/pipelines_config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["enabled"] != true {
			t.Errorf("expected enabled=true in body, got: %v", body["enabled"])
		}
		mustEncodeJSON(t, w, bitbucket.PipelineConfig{Enabled: true})
	}))
	got, err := client.PipelineConfig("testws", "testrepo").Enable(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled {
		t.Errorf("expected enabled=true, got %+v", got)
	}
}

func TestPipelineConfig_Disable(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/pipelines_config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["enabled"] != false {
			t.Errorf("expected enabled=false in body, got: %v", body["enabled"])
		}
		mustEncodeJSON(t, w, bitbucket.PipelineConfig{Enabled: false})
	}))
	got, err := client.PipelineConfig("testws", "testrepo").Disable(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Enabled {
		t.Errorf("expected enabled=false, got %+v", got)
	}
}
