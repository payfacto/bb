package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const schedulesPath = "/repositories/testws/testrepo/pipelines_config/schedules"

func TestPipelineSchedules_List(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != schedulesPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{{
			"uuid":         "{s-1}",
			"enabled":      true,
			"cron_pattern": "0 0 12 * * ? *",
			"target": map[string]any{
				"ref_type": "branch",
				"ref_name": "main",
				"selector": map[string]any{"type": "branches", "pattern": "main"},
			},
		}}})
	}))
	got, err := client.PipelineSchedules("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(got))
	}
	s := got[0]
	if s.UUID != "{s-1}" || !s.Enabled || s.CronPattern != "0 0 12 * * ? *" {
		t.Errorf("unexpected schedule: %+v", s)
	}
	if s.Target.RefName != "main" || s.Target.Selector.Type != "branches" {
		t.Errorf("unexpected target: %+v", s.Target)
	}
}

func TestPipelineSchedules_Get(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != schedulesPath+"/{s-1}" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"uuid": "{s-1}", "enabled": false})
	}))
	got, err := client.PipelineSchedules("testws", "testrepo").Get(context.Background(), "{s-1}")
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{s-1}" || got.Enabled {
		t.Errorf("unexpected schedule: %+v", got)
	}
}

func TestPipelineSchedules_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != schedulesPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			CronPattern string `json:"cron_pattern"`
			Enabled     *bool  `json:"enabled"`
			Target      struct {
				Type     string `json:"type"`
				RefType  string `json:"ref_type"`
				RefName  string `json:"ref_name"`
				Selector struct {
					Type    string `json:"type"`
					Pattern string `json:"pattern"`
				} `json:"selector"`
			} `json:"target"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.CronPattern != "0 0 2 * * ? *" {
			t.Errorf("unexpected cron_pattern: %q", body.CronPattern)
		}
		if body.Enabled == nil || *body.Enabled {
			t.Errorf("expected enabled=false in body, got %v", body.Enabled)
		}
		tgt := body.Target
		if tgt.Type != "pipeline_ref_target" || tgt.RefType != "branch" || tgt.RefName != "main" {
			t.Errorf("unexpected target: %+v", tgt)
		}
		if tgt.Selector.Type != "custom" || tgt.Selector.Pattern != "nightly" {
			t.Errorf("unexpected selector: %+v", tgt.Selector)
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, map[string]any{"uuid": "{s-2}", "cron_pattern": body.CronPattern})
	}))
	disabled := false
	got, err := client.PipelineSchedules("testws", "testrepo").Create(context.Background(), bitbucket.CreatePipelineScheduleInput{
		CronPattern: "0 0 2 * * ? *",
		Enabled:     &disabled,
		Target: bitbucket.PipelineScheduleTarget{
			Type:     bitbucket.PipelineRefTargetType,
			RefType:  "branch",
			RefName:  "main",
			Selector: bitbucket.PipelineSelector{Type: "custom", Pattern: "nightly"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{s-2}" {
		t.Errorf("unexpected uuid: %s", got.UUID)
	}
}

func TestPipelineSchedules_CreateDefaults(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, ok := body["enabled"]; ok {
			t.Errorf("expected enabled omitted so the API default applies, got %v", body["enabled"])
		}
		target, _ := body["target"].(map[string]any)
		if target["type"] != "pipeline_ref_target" {
			t.Errorf("expected target.type defaulted to pipeline_ref_target, got %v", target["type"])
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, map[string]any{"uuid": "{s-3}"})
	}))
	_, err := client.PipelineSchedules("testws", "testrepo").Create(context.Background(), bitbucket.CreatePipelineScheduleInput{
		CronPattern: "0 0 2 * * ? *",
		Target:      bitbucket.PipelineScheduleTarget{RefType: "branch", RefName: "main"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPipelineSchedules_Update(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != schedulesPath+"/{s-1}" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body) != 1 || body["enabled"] != true {
			t.Errorf("expected body {enabled:true}, got %v", body)
		}
		mustEncodeJSON(t, w, map[string]any{"uuid": "{s-1}", "enabled": true})
	}))
	got, err := client.PipelineSchedules("testws", "testrepo").Update(context.Background(), "{s-1}",
		bitbucket.UpdatePipelineScheduleInput{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled {
		t.Errorf("expected enabled schedule, got %+v", got)
	}
}

func TestPipelineSchedules_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != schedulesPath+"/{s-1}" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := client.PipelineSchedules("testws", "testrepo").Delete(context.Background(), "{s-1}"); err != nil {
		t.Fatal(err)
	}
}

func TestPipelineSchedules_Executions(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != schedulesPath+"/{s-1}/executions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{
			{"type": "pipeline_schedule_execution_executed", "pipeline": map[string]any{"uuid": "{p-1}"}},
		}})
	}))
	got, err := client.PipelineSchedules("testws", "testrepo").Executions(context.Background(), "{s-1}")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0]["type"] != "pipeline_schedule_execution_executed" {
		t.Errorf("unexpected executions: %+v", got)
	}
}
