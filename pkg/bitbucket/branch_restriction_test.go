package bitbucket_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestBranchRestrictions_List(t *testing.T) {
	val1 := 2
	restrictions := []bitbucket.BranchRestriction{
		{ID: 1, Kind: "require_approvals_to_merge", BranchMatchKind: "glob", Pattern: "main", Value: &val1},
		{ID: 2, Kind: "force", BranchMatchKind: "glob", Pattern: "main"},
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/branch-restrictions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("pagelen") != "50" {
			t.Errorf("expected pagelen=50, got %s", r.URL.Query().Get("pagelen"))
		}
		mustEncodeJSON(t, w, map[string]any{"values": restrictions})
	}))
	got, err := client.Restrictions("testws", "testrepo").List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 restrictions, got %d", len(got))
	}
	if got[0].ID != 1 || got[0].Kind != "require_approvals_to_merge" || got[0].Pattern != "main" {
		t.Errorf("unexpected first restriction: %+v", got[0])
	}
	if got[0].Value == nil || *got[0].Value != 2 {
		t.Errorf("expected Value=2, got %v", got[0].Value)
	}
	if got[1].ID != 2 || got[1].Kind != "force" {
		t.Errorf("unexpected second restriction: %+v", got[1])
	}
	if got[1].Value != nil {
		t.Errorf("expected nil Value for restriction 2, got %v", got[1].Value)
	}
}

func TestBranchRestrictions_Create(t *testing.T) {
	val := 3
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/branch-restrictions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["kind"] != "require_approvals_to_merge" {
			t.Errorf("expected kind=require_approvals_to_merge, got %v", body["kind"])
		}
		if body["branch_match_kind"] != "glob" {
			t.Errorf("expected branch_match_kind=glob, got %v", body["branch_match_kind"])
		}
		if body["pattern"] != "main" {
			t.Errorf("expected pattern=main, got %v", body["pattern"])
		}
		if body["value"] != float64(3) {
			t.Errorf("expected value=3, got %v", body["value"])
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.BranchRestriction{
			ID:              10,
			Kind:            "require_approvals_to_merge",
			BranchMatchKind: "glob",
			Pattern:         "main",
			Value:           &val,
		})
	}))
	input := bitbucket.CreateBranchRestrictionInput{
		Kind:            "require_approvals_to_merge",
		BranchMatchKind: "glob",
		Pattern:         "main",
		Value:           &val,
	}
	got, err := client.Restrictions("testws", "testrepo").Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 10 || got.Kind != "require_approvals_to_merge" {
		t.Errorf("unexpected result: %+v", got)
	}
	if got.Value == nil || *got.Value != 3 {
		t.Errorf("expected Value=3, got %v", got.Value)
	}
}

func TestBranchRestrictions_CreateNoValue(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if _, hasValue := body["value"]; hasValue {
			t.Errorf("expected no 'value' field in body, got %v", body["value"])
		}
		w.WriteHeader(http.StatusCreated)
		mustEncodeJSON(t, w, bitbucket.BranchRestriction{
			ID:              11,
			Kind:            "force",
			BranchMatchKind: "glob",
			Pattern:         "main",
		})
	}))
	input := bitbucket.CreateBranchRestrictionInput{
		Kind:            "force",
		BranchMatchKind: "glob",
		Pattern:         "main",
	}
	got, err := client.Restrictions("testws", "testrepo").Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 11 || got.Kind != "force" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestBranchRestrictions_Delete(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/repositories/testws/testrepo/branch-restrictions/5" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.Restrictions("testws", "testrepo").Delete(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
}
