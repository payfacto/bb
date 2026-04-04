package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfactopay/bb/pkg/bitbucket"
)

func TestWebhooks_List(t *testing.T) {
	hooks := []bitbucket.Webhook{
		{UUID: "{abc123}", Description: "CI hook", URL: "https://ci.example.com/hook", Active: true, Events: []string{"repo:push"}},
		{UUID: "{def456}", Description: "Notify", URL: "https://notify.example.com/hook", Active: false, Events: []string{"pullrequest:created", "pullrequest:merged"}},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/hooks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		json.NewEncoder(w).Encode(map[string]any{"values": hooks})
	}))
	got, err := client.Webhooks("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 webhooks, got %d", len(got))
	}
	if got[0].UUID != "{abc123}" || got[0].Description != "CI hook" || !got[0].Active {
		t.Errorf("unexpected first webhook: %+v", got[0])
	}
	if len(got[0].Events) != 1 || got[0].Events[0] != "repo:push" {
		t.Errorf("unexpected events: %v", got[0].Events)
	}
	if got[1].UUID != "{def456}" || got[1].Active {
		t.Errorf("unexpected second webhook: %+v", got[1])
	}
}

func TestWebhooks_Create(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/hooks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["url"] != "https://example.com/hook" {
			t.Errorf("expected url=https://example.com/hook, got %v", body["url"])
		}
		if body["active"] != true {
			t.Errorf("expected active=true, got %v", body["active"])
		}
		events, ok := body["events"].([]any)
		if !ok || len(events) != 2 {
			t.Errorf("expected 2 events, got %v", body["events"])
		}
		if body["description"] != "My hook" {
			t.Errorf("expected description=My hook, got %v", body["description"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bitbucket.Webhook{
			UUID:        "{new-hook-uuid}",
			Description: "My hook",
			URL:         "https://example.com/hook",
			Active:      true,
			Events:      []string{"repo:push", "pullrequest:created"},
		})
	}))
	input := bitbucket.CreateWebhookInput{
		Description: "My hook",
		URL:         "https://example.com/hook",
		Active:      true,
		Events:      []string{"repo:push", "pullrequest:created"},
	}
	got, err := client.Webhooks("testws", "testrepo").Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{new-hook-uuid}" || got.Description != "My hook" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestWebhooks_CreateNoDescription(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, hasDesc := body["description"]; hasDesc {
			t.Errorf("expected no 'description' field in body, got %v", body["description"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bitbucket.Webhook{
			UUID:   "{minimal-hook}",
			URL:    "https://example.com/hook",
			Active: true,
			Events: []string{"repo:push"},
		})
	}))
	input := bitbucket.CreateWebhookInput{
		URL:    "https://example.com/hook",
		Active: true,
		Events: []string{"repo:push"},
	}
	got, err := client.Webhooks("testws", "testrepo").Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != "{minimal-hook}" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestWebhooks_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		// r.URL.Path contains the decoded path — braces are unescaped by httptest
		if r.URL.Path != "/repositories/testws/testrepo/hooks/{abc123}" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Webhooks("testws", "testrepo").Delete(context.Background(), "{abc123}")
	if err != nil {
		t.Fatal(err)
	}
}
