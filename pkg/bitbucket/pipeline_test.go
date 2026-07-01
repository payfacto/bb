package bitbucket_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	got, err := client.Pipelines("testws", "testrepo").List(context.Background(), "")
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

func TestPipelines_GetByBuildNumber(t *testing.T) {
	pipeline := bitbucket.Pipeline{UUID: "{abc-123}", BuildNumber: 42}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		// A build number is addressed as a plain integer path segment, with no
		// braces - unlike the UUID form, which requires them.
		if !strings.HasSuffix(r.URL.Path, "/pipelines/42") {
			t.Errorf("expected path to end with /pipelines/42, got %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, pipeline)
	}))
	got, err := client.Pipelines("testws", "testrepo").GetByBuildNumber(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{abc-123}" {
		t.Errorf("expected uuid {abc-123}, got %s", got.UUID)
	}
	if got.BuildNumber != 42 {
		t.Errorf("expected build 42, got %d", got.BuildNumber)
	}
}

func TestPipelines_Latest_NoBranch(t *testing.T) {
	pipelines := []bitbucket.Pipeline{
		{UUID: "{p2}", BuildNumber: 2, Target: bitbucket.PipelineTarget{RefName: "main"}},
		{UUID: "{p1}", BuildNumber: 1, Target: bitbucket.PipelineTarget{RefName: "dev"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sort") != "-created_on" {
			t.Errorf("expected sort=-created_on, got %s", r.URL.Query().Get("sort"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": pipelines})
	}))
	got, err := client.Pipelines("testws", "testrepo").Latest(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if got.BuildNumber != 2 {
		t.Errorf("expected latest build 2, got %d", got.BuildNumber)
	}
}

func TestPipelines_Latest_Branch(t *testing.T) {
	pipelines := []bitbucket.Pipeline{
		{UUID: "{p3}", BuildNumber: 3, Target: bitbucket.PipelineTarget{RefName: "main"}},
		{UUID: "{p2}", BuildNumber: 2, Target: bitbucket.PipelineTarget{RefName: "feature"}},
		{UUID: "{p1}", BuildNumber: 1, Target: bitbucket.PipelineTarget{RefName: "feature"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mustEncodeJSON(t, w, map[string]any{"values": pipelines})
	}))
	got, err := client.Pipelines("testws", "testrepo").Latest(context.Background(), "feature")
	if err != nil {
		t.Fatal(err)
	}
	if got.BuildNumber != 2 {
		t.Errorf("expected latest feature build 2, got %d", got.BuildNumber)
	}
}

func TestPipelines_Latest_NotFound(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mustEncodeJSON(t, w, map[string]any{"values": []any{}})
	}))
	_, err := client.Pipelines("testws", "testrepo").Latest(context.Background(), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *bitbucket.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
		t.Errorf("expected 404 APIError, got %v", err)
	}
}

// watchPipelineHandler serves the pipeline-get and steps requests Watch makes.
// getState is called per pipeline-get to advance the pipeline's state across
// polls; steps are returned verbatim.
func watchPipelineHandler(t *testing.T, buildNumber int, steps []bitbucket.PipelineStep, getState func(poll int32) bitbucket.PipelineState) http.HandlerFunc {
	var polls atomic.Int32
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/steps/") {
			mustEncodeJSON(t, w, map[string]any{"values": steps})
			return
		}
		n := polls.Add(1)
		mustEncodeJSON(t, w, bitbucket.Pipeline{UUID: "{p1}", BuildNumber: buildNumber, State: getState(n)})
	}
}

func TestPipelines_Watch_PollToSuccess(t *testing.T) {
	handler := watchPipelineHandler(t, 7, nil, func(poll int32) bitbucket.PipelineState {
		if poll >= 2 {
			return bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "SUCCESSFUL"}}
		}
		return bitbucket.PipelineState{Name: "IN_PROGRESS", Stage: &bitbucket.PipelineStage{Name: "RUNNING"}}
	})
	client := newTestClient(t, handler)
	res, err := client.Pipelines("testws", "testrepo").Watch(context.Background(), "{p1}", bitbucket.WatchOptions{Interval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != bitbucket.WatchSuccess {
		t.Errorf("expected success, got %s", res.Status)
	}
}

func TestPipelines_Watch_PollToFailure(t *testing.T) {
	handler := watchPipelineHandler(t, 8, nil, func(poll int32) bitbucket.PipelineState {
		if poll >= 2 {
			return bitbucket.PipelineState{Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: "FAILED"}}
		}
		return bitbucket.PipelineState{Name: "IN_PROGRESS", Stage: &bitbucket.PipelineStage{Name: "RUNNING"}}
	})
	client := newTestClient(t, handler)
	res, err := client.Pipelines("testws", "testrepo").Watch(context.Background(), "{p1}", bitbucket.WatchOptions{Interval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != bitbucket.WatchFailed {
		t.Errorf("expected failed, got %s", res.Status)
	}
}

func TestPipelines_Watch_Blocked(t *testing.T) {
	steps := []bitbucket.PipelineStep{
		{Name: "build", State: bitbucket.PipelineState{Name: "COMPLETED"}},
		{Name: "deploy", State: bitbucket.PipelineState{Name: "PENDING"}},
	}
	handler := watchPipelineHandler(t, 9, steps, func(poll int32) bitbucket.PipelineState {
		return bitbucket.PipelineState{Name: "IN_PROGRESS", Stage: &bitbucket.PipelineStage{Name: "PAUSED"}}
	})
	client := newTestClient(t, handler)
	res, err := client.Pipelines("testws", "testrepo").Watch(context.Background(), "{p1}", bitbucket.WatchOptions{Interval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != bitbucket.WatchBlocked {
		t.Fatalf("expected blocked, got %s", res.Status)
	}
	if res.ManualGate == nil {
		t.Fatal("expected manual gate, got nil")
	}
	if res.ManualGate.Step != "deploy" {
		t.Errorf("expected gate step deploy, got %q", res.ManualGate.Step)
	}
	if !strings.HasSuffix(res.ManualGate.URL, "/pipelines/results/9") {
		t.Errorf("expected gate URL ending /pipelines/results/9, got %q", res.ManualGate.URL)
	}
}

func TestPipelines_Watch_Timeout(t *testing.T) {
	handler := watchPipelineHandler(t, 5, nil, func(poll int32) bitbucket.PipelineState {
		return bitbucket.PipelineState{Name: "IN_PROGRESS", Stage: &bitbucket.PipelineStage{Name: "RUNNING"}}
	})
	client := newTestClient(t, handler)
	res, err := client.Pipelines("testws", "testrepo").Watch(context.Background(), "{p1}",
		bitbucket.WatchOptions{Interval: time.Millisecond, Timeout: 15 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != bitbucket.WatchTimeout {
		t.Errorf("expected timeout, got %s", res.Status)
	}
}

func TestPipelines_Watch_ContextCancel(t *testing.T) {
	handler := watchPipelineHandler(t, 3, nil, func(poll int32) bitbucket.PipelineState {
		return bitbucket.PipelineState{Name: "IN_PROGRESS", Stage: &bitbucket.PipelineStage{Name: "RUNNING"}}
	})
	client := newTestClient(t, handler)
	ctx, cancel := context.WithCancel(context.Background())
	_, err := client.Pipelines("testws", "testrepo").Watch(ctx, "{p1}", bitbucket.WatchOptions{
		Interval: time.Millisecond,
		// Cancel mid-watch, as a Ctrl-C signal would. Watch must stop and return
		// the cancellation rather than polling forever.
		OnPoll: func(bitbucket.Pipeline, []bitbucket.PipelineStep) { cancel() },
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// triggerBody captures the decoded POST pipelines/ body for assertions.
func triggerBody(t *testing.T, opts bitbucket.TriggerOptions) map[string]any {
	t.Helper()
	var body map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.Pipeline{UUID: "{new-uuid}", BuildNumber: 43})
	}))
	got, err := client.Pipelines("testws", "testrepo").Trigger(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if got.BuildNumber != 43 {
		t.Errorf("expected build 43, got %d", got.BuildNumber)
	}
	return body
}

func TestPipelines_Trigger_Branch(t *testing.T) {
	body := triggerBody(t, bitbucket.TriggerOptions{Ref: bitbucket.TriggerRef{Branch: "main"}})
	target, _ := body["target"].(map[string]any)
	if target["type"] != "pipeline_ref_target" {
		t.Errorf("type = %v, want pipeline_ref_target", target["type"])
	}
	if target["ref_type"] != "branch" || target["ref_name"] != "main" {
		t.Errorf("ref = %v/%v, want branch/main", target["ref_type"], target["ref_name"])
	}
	if _, ok := target["selector"]; ok {
		t.Errorf("did not expect a selector, got %v", target["selector"])
	}
	if _, ok := body["variables"]; ok {
		t.Errorf("did not expect variables, got %v", body["variables"])
	}
}

func TestPipelines_Trigger_Tag(t *testing.T) {
	body := triggerBody(t, bitbucket.TriggerOptions{Ref: bitbucket.TriggerRef{Tag: "v1.0"}})
	target, _ := body["target"].(map[string]any)
	if target["type"] != "pipeline_ref_target" {
		t.Errorf("type = %v, want pipeline_ref_target", target["type"])
	}
	if target["ref_type"] != "tag" || target["ref_name"] != "v1.0" {
		t.Errorf("ref = %v/%v, want tag/v1.0", target["ref_type"], target["ref_name"])
	}
}

func TestPipelines_Trigger_Commit(t *testing.T) {
	body := triggerBody(t, bitbucket.TriggerOptions{Ref: bitbucket.TriggerRef{Commit: "abc123"}})
	target, _ := body["target"].(map[string]any)
	if target["type"] != "pipeline_commit_target" {
		t.Errorf("type = %v, want pipeline_commit_target", target["type"])
	}
	if _, ok := target["ref_type"]; ok {
		t.Errorf("commit target should omit ref_type, got %v", target["ref_type"])
	}
	commit, _ := target["commit"].(map[string]any)
	if commit["type"] != "commit" || commit["hash"] != "abc123" {
		t.Errorf("commit = %v, want {commit, abc123}", commit)
	}
}

func TestPipelines_Trigger_CustomWithVariables(t *testing.T) {
	body := triggerBody(t, bitbucket.TriggerOptions{
		Ref:       bitbucket.TriggerRef{Branch: "main"},
		Custom:    "deploy",
		Variables: []bitbucket.TriggerVariable{{Key: "FOO", Value: "bar"}},
	})
	target, _ := body["target"].(map[string]any)
	selector, _ := target["selector"].(map[string]any)
	if selector["type"] != "custom" || selector["pattern"] != "deploy" {
		t.Errorf("selector = %v, want {custom, deploy}", selector)
	}
	vars, _ := body["variables"].([]any)
	if len(vars) != 1 {
		t.Fatalf("expected 1 variable, got %v", body["variables"])
	}
	v0, _ := vars[0].(map[string]any)
	if v0["key"] != "FOO" || v0["value"] != "bar" {
		t.Errorf("variable = %v, want {FOO, bar}", v0)
	}
	if _, ok := v0["secured"]; ok {
		t.Errorf("unsecured variable should omit 'secured', got %v", v0["secured"])
	}
}

func TestPipelines_Trigger_NoRefErrors(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no HTTP request expected when no ref is set")
	}))
	_, err := client.Pipelines("testws", "testrepo").Trigger(context.Background(), bitbucket.TriggerOptions{})
	if err == nil {
		t.Fatal("expected an error when no branch/tag/commit is set")
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
	// Bitbucket Cloud requires the curly braces on pipeline/step UUIDs.
	// Stripping them produces a 404 against the live API — guard against that
	// by asserting the request path preserves the braces verbatim.
	wantPath := "/repositories/testws/testrepo/pipelines/%7Babc-123%7D/steps/%7Bstep-1%7D/log"
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != wantPath {
			t.Errorf("expected escaped path %q, got %q", wantPath, r.URL.EscapedPath())
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
