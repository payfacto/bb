package bitbucket_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	trPipelineUUID = "{pipe-1}"
	trStepWithTest = "{step-a}"
	trStepNoReport = "{step-b}"
	trStepNotRun   = "{step-c}"
)

func completedStep(uuid, name, result string) bitbucket.PipelineStep {
	return bitbucket.PipelineStep{UUID: uuid, Name: name, State: bitbucket.PipelineState{
		Name: "COMPLETED", Result: &bitbucket.PipelineResult{Name: result}}}
}

func trSteps() []bitbucket.PipelineStep {
	return []bitbucket.PipelineStep{
		completedStep(trStepWithTest, "Build and test", "FAILED"),
		completedStep(trStepNoReport, "Lint", "SUCCESSFUL"),
		completedStep(trStepNotRun, "Deploy", "NOT_RUN"),
	}
}

// testReportServer serves a step's test report for trStepWithTest only; every
// other step 404s like the real API does for steps that published no report.
// It fails the test if a NOT_RUN step is queried (wasted request).
func testReportServer(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, trStepNotRun) {
			t.Errorf("NOT_RUN step must not be queried: %s", path)
		}
		if !strings.Contains(path, trStepWithTest) {
			http.Error(w, `{"error":{"message":"Not found"}}`, http.StatusNotFound)
			return
		}
		switch {
		case strings.HasSuffix(path, "/test_case_reasons"):
			mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{
				{"message": "expected 1 but was 2", "output": "at FooTest.java:12"},
			}})
		case strings.HasSuffix(path, "/test_reports/test_cases"):
			mustEncodeJSON(t, w, map[string]any{"values": []map[string]any{
				{"uuid": "{tc-1}", "name": "testAdd", "class_name": "FooTest", "fully_qualified_name": "com.x.FooTest.testAdd", "status": "FAILED", "duration": "PT0.5S"},
				{"uuid": "{tc-2}", "name": "testSub", "class_name": "FooTest", "fully_qualified_name": "com.x.FooTest.testSub", "status": "SUCCESSFUL", "duration": 12},
			}})
		case strings.HasSuffix(path, "/test_reports"):
			mustEncodeJSON(t, w, map[string]any{
				"uuid": "{rep-1}", "step_uuid": trStepWithTest,
				"number_of_test_cases": 2, "number_of_successful_test_cases": 1,
				"number_of_failed_test_cases": 1, "number_of_error_test_cases": 0,
				"number_of_skipped_test_cases": 0,
			})
		default:
			t.Errorf("unexpected path %s", path)
		}
	})
}

func TestPipelines_TestReport_found(t *testing.T) {
	client := newTestClient(t, testReportServer(t))
	got, err := client.Pipelines("ws", "repo").TestReport(context.Background(), trPipelineUUID, trStepWithTest)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.TotalCases != 2 || got.FailedCases != 1 || got.SuccessfulCases != 1 {
		t.Errorf("unexpected report: %+v", got)
	}
}

func TestPipelines_TestReport_notFoundIsNoReport(t *testing.T) {
	client := newTestClient(t, testReportServer(t))
	got, err := client.Pipelines("ws", "repo").TestReport(context.Background(), trPipelineUUID, trStepNoReport)
	if err != nil {
		t.Fatalf("404 must mean no report, not an error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil report, got %+v", got)
	}
}

func TestPipelines_TestReport_otherErrorsPropagate(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	if _, err := client.Pipelines("ws", "repo").TestReport(context.Background(), trPipelineUUID, trStepWithTest); err == nil {
		t.Fatal("expected a 500 to surface as an error")
	}
}

func TestPipelines_StepTestReports_marksStepsWithoutReport(t *testing.T) {
	client := newTestClient(t, testReportServer(t))
	got, err := client.Pipelines("ws", "repo").StepTestReports(context.Background(), trPipelineUUID, trSteps())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected one entry per step, got %d", len(got))
	}
	if got[0].StepName != "Build and test" || got[0].Report == nil || got[0].Report.FailedCases != 1 {
		t.Errorf("step with report: %+v", got[0])
	}
	if got[1].Report != nil || got[2].Report != nil {
		t.Errorf("steps without report must have nil Report: %+v %+v", got[1], got[2])
	}
}

func TestPipelines_StepTestCases_all(t *testing.T) {
	client := newTestClient(t, testReportServer(t))
	got, err := client.Pipelines("ws", "repo").StepTestCases(context.Background(), trPipelineUUID, trSteps(), bitbucket.TestCaseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 cases, got %d: %+v", len(got), got)
	}
	if got[0].StepName != "Build and test" || got[0].FullyQualifiedName != "com.x.FooTest.testAdd" {
		t.Errorf("unexpected first case: %+v", got[0])
	}
	if got[0].Duration != "PT0.5S" || got[1].Duration != "12" {
		t.Errorf("duration must accept string or number, got %q / %q", got[0].Duration, got[1].Duration)
	}
	if got[0].Reasons != nil {
		t.Errorf("reasons must not be fetched without WithReasons: %+v", got[0].Reasons)
	}
}

func TestPipelines_StepTestCases_failedWithReasons(t *testing.T) {
	client := newTestClient(t, testReportServer(t))
	got, err := client.Pipelines("ws", "repo").StepTestCases(context.Background(), trPipelineUUID, trSteps(),
		bitbucket.TestCaseOptions{FailedOnly: true, WithReasons: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Status != "FAILED" {
		t.Fatalf("expected only the failed case, got %+v", got)
	}
	if len(got[0].Reasons) != 1 || got[0].Reasons[0].Message != "expected 1 but was 2" {
		t.Errorf("unexpected reasons: %+v", got[0].Reasons)
	}
}

func TestPipelines_TestCaseReasons_acceptsBareObjectAndArray(t *testing.T) {
	bodies := map[string]string{
		"object": `{"message":"m1","output":"o1"}`,
		"array":  `[{"message":"m1","output":"o1"}]`,
		"paged":  `{"values":[{"message":"m1","output":"o1"}]}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(body))
			}))
			got, err := client.Pipelines("ws", "repo").TestCaseReasons(context.Background(), trPipelineUUID, trStepWithTest, "{tc-1}")
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].Message != "m1" || got[0].Output != "o1" {
				t.Errorf("unexpected reasons: %+v", got)
			}
		})
	}
}

func TestIsFailedTestStatus(t *testing.T) {
	for status, want := range map[string]bool{
		"FAILED": true, "failed": true, "ERROR": true, "FAILURE": true,
		"SUCCESSFUL": false, "SUCCESS": false, "SKIPPED": false, "": false,
	} {
		if got := bitbucket.IsFailedTestStatus(status); got != want {
			t.Errorf("IsFailedTestStatus(%q) = %v, want %v", status, got, want)
		}
	}
}
