package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func sampleReport() bitbucket.Report {
	return bitbucket.Report{
		UUID:       "{r-1}",
		ExternalID: "sonar-001",
		Title:      "SonarCloud Quality Gate",
		Reporter:   "SonarCloud",
		ReportType: "COVERAGE",
		Result:     "PASSED",
		Link:       "https://sonarcloud.io/x",
		Data:       []bitbucket.ReportData{{Type: "PERCENTAGE", Title: "Coverage", Value: 87.5}},
	}
}

func TestReportListString_empty(t *testing.T) {
	if out := render.ReportListString(nil); !strings.Contains(out, "No reports found.") {
		t.Errorf("expected empty message, got: %q", out)
	}
}

func TestReportListString_row(t *testing.T) {
	out := render.ReportListString([]bitbucket.Report{sampleReport()})
	for _, want := range []string{"sonar-001", "COVERAGE", "PASSED", "SonarCloud Quality Gate"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}

func TestReportListString_fallsBackToUUID(t *testing.T) {
	rep := sampleReport()
	rep.ExternalID = ""
	if out := render.ReportListString([]bitbucket.Report{rep}); !strings.Contains(out, "{r-1}") {
		t.Errorf("expected uuid when external_id is empty, got:\n%s", out)
	}
}

func TestReportDetailString_fields(t *testing.T) {
	out := render.ReportDetailString(sampleReport())
	for _, want := range []string{"SonarCloud Quality Gate", "sonar-001", "{r-1}", "SonarCloud", "COVERAGE", "PASSED",
		"https://sonarcloud.io/x", "Coverage", "87.5"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}

func TestReportAnnotationListString(t *testing.T) {
	if out := render.ReportAnnotationListString(nil); !strings.Contains(out, "No annotations found.") {
		t.Errorf("expected empty message, got: %q", out)
	}
	out := render.ReportAnnotationListString([]bitbucket.ReportAnnotation{
		{Severity: "HIGH", AnnotationType: "BUG", Path: "main.go", Line: 12, Summary: "nil dereference"},
		{Severity: "LOW", AnnotationType: "CODE_SMELL", Summary: "report-wide finding"},
	})
	for _, want := range []string{"HIGH", "BUG", "main.go:12", "nil dereference", "CODE_SMELL", "report-wide finding"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}
