package bitbucket_test

import (
	"context"
	"net/http"
	"testing"
)

const reportsPath = "/repositories/testws/testrepo/commit/abc123/reports"

func TestCommits_Reports(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != reportsPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{{
			"uuid":        "{r-1}",
			"external_id": "sonar-001",
			"title":       "SonarCloud",
			"report_type": "COVERAGE",
			"result":      "PASSED",
			"data":        []map[string]any{{"type": "PERCENTAGE", "title": "Coverage", "value": 87.5}},
		}}})
	}))
	got, err := client.Commits("testws", "testrepo").Reports(context.Background(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 report, got %d", len(got))
	}
	rep := got[0]
	if rep.UUID != "{r-1}" || rep.ExternalID != "sonar-001" || rep.ReportType != "COVERAGE" || rep.Result != "PASSED" {
		t.Errorf("unexpected report: %+v", rep)
	}
	if len(rep.Data) != 1 || rep.Data[0].Title != "Coverage" || rep.Data[0].Value != 87.5 {
		t.Errorf("unexpected report data: %+v", rep.Data)
	}
}

func TestCommits_Report(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != reportsPath+"/sonar-001" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"uuid": "{r-1}", "title": "SonarCloud", "reporter": "Sonar"})
	}))
	got, err := client.Commits("testws", "testrepo").Report(context.Background(), "abc123", "sonar-001")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "SonarCloud" || got.Reporter != "Sonar" {
		t.Errorf("unexpected report: %+v", got)
	}
}

func TestCommits_ReportAnnotations(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != reportsPath+"/sonar-001/annotations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{{
			"uuid":            "{a-1}",
			"annotation_type": "BUG",
			"severity":        "HIGH",
			"path":            "main.go",
			"line":            12,
			"summary":         "nil dereference",
		}}})
	}))
	got, err := client.Commits("testws", "testrepo").ReportAnnotations(context.Background(), "abc123", "sonar-001")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(got))
	}
	a := got[0]
	if a.AnnotationType != "BUG" || a.Severity != "HIGH" || a.Path != "main.go" || a.Line != 12 || a.Summary != "nil dereference" {
		t.Errorf("unexpected annotation: %+v", a)
	}
}
