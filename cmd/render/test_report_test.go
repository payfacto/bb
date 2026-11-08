package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestTestReportsString_rows(t *testing.T) {
	reports := []bitbucket.StepTestReport{
		{StepName: "Build and test", Report: &bitbucket.TestReportSummary{
			TotalCases: 10, SuccessfulCases: 7, FailedCases: 2, ErrorCases: 1}},
		{StepName: "Deploy"},
	}
	out := render.TestReportsString(reports)
	for _, want := range []string{"Build and test", "10", "7", "2", "Deploy", "no report"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}

func TestTestReportsString_empty(t *testing.T) {
	if out := render.TestReportsString(nil); !strings.Contains(out, "No steps found.") {
		t.Errorf("expected empty message, got: %q", out)
	}
}

func TestTestCasesString_showsReasons(t *testing.T) {
	cases := []bitbucket.StepTestCase{{
		StepName: "Build and test",
		TestCase: bitbucket.TestCase{Name: "testAdd", FullyQualifiedName: "com.x.FooTest.testAdd", Status: "FAILED", Duration: "PT0.5S"},
		Reasons:  []bitbucket.TestCaseReason{{Message: "expected 1 but was 2"}},
	}}
	out := render.TestCasesString(cases)
	for _, want := range []string{"FAILED", "com.x.FooTest.testAdd", "PT0.5S", "Build and test", "expected 1 but was 2"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q, got:\n%s", want, out)
		}
	}
}

func TestTestCasesString_fallsBackToName(t *testing.T) {
	cases := []bitbucket.StepTestCase{{TestCase: bitbucket.TestCase{Name: "testSub", ClassName: "FooTest", Status: "ERROR"}}}
	if out := render.TestCasesString(cases); !strings.Contains(out, "FooTest.testSub") {
		t.Errorf("expected class.name fallback, got:\n%s", out)
	}
}

func TestTestCasesString_empty(t *testing.T) {
	if out := render.TestCasesString(nil); !strings.Contains(out, "No test cases found.") {
		t.Errorf("expected empty message, got: %q", out)
	}
}
