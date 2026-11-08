package bitbucket

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// TestCaseOptions configures StepTestCases. FailedOnly keeps only failed or
// errored cases; WithReasons fetches the failure reasons of each failed case.
type TestCaseOptions struct {
	FailedOnly  bool
	WithReasons bool
}

// IsFailedTestStatus reports whether a test case status denotes a failure.
// The status vocabulary is undocumented, so FAILED, FAILURE, and ERROR are all
// accepted, case-insensitively.
func IsFailedTestStatus(status string) bool {
	switch strings.ToUpper(status) {
	case "FAILED", "FAILURE", "ERROR":
		return true
	}
	return false
}

func (r *PipelineResource) testReportPath(pipelineUUID, stepUUID string) string {
	return fmt.Sprintf("%s%s/steps/%s/test_reports",
		r.basePath(), url.PathEscape(pipelineUUID), url.PathEscape(stepUUID))
}

// TestReport returns the test report summary of one pipeline step. It returns
// (nil, nil) when the step published no test report: Bitbucket answers that
// case with HTTP 404, which is a normal outcome rather than a failure.
func (r *PipelineResource) TestReport(ctx context.Context, pipelineUUID, stepUUID string) (*TestReportSummary, error) {
	data, err := r.client.do(ctx, "GET", r.testReportPath(pipelineUUID, stepUUID), nil, nil)
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	report, err := decode[TestReportSummary](data)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// TestCases returns every test case Bitbucket lists for a step, following
// pagination. A step without a test report (HTTP 404) yields an empty slice.
// In practice Bitbucket may list only non-passing cases here.
func (r *PipelineResource) TestCases(ctx context.Context, pipelineUUID, stepUUID string) ([]TestCase, error) {
	path := r.testReportPath(pipelineUUID, stepUUID) + "/test_cases"
	cases, err := fetchAllPages[TestCase](ctx, r.client, path, nil)
	if isNotFound(err) {
		return []TestCase{}, nil
	}
	return cases, err
}

// TestCaseReasons returns the failure reasons (message and output) recorded
// for one test case. A test case with no recorded reasons (HTTP 404) yields an
// empty slice. The response shape is undocumented, so a paged envelope, a bare
// array, and a single object are all accepted.
func (r *PipelineResource) TestCaseReasons(ctx context.Context, pipelineUUID, stepUUID, testCaseUUID string) ([]TestCaseReason, error) {
	path := fmt.Sprintf("%s/test_cases/%s/test_case_reasons",
		r.testReportPath(pipelineUUID, stepUUID), url.PathEscape(testCaseUUID))
	data, err := r.client.do(ctx, "GET", path, nil, nil)
	if isNotFound(err) {
		return []TestCaseReason{}, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeTestCaseReasons(data)
}

// decodeTestCaseReasons decodes a reasons payload that may be a paged
// {"values": [...]} envelope, a bare JSON array, or a single reason object.
func decodeTestCaseReasons(data []byte) ([]TestCaseReason, error) {
	trimmed := bytes.TrimSpace(data)
	if bytes.HasPrefix(trimmed, []byte("[")) {
		return decode[[]TestCaseReason](trimmed)
	}
	envelope, err := decode[struct {
		Values *[]TestCaseReason `json:"values"`
	}](trimmed)
	if err != nil {
		return nil, err
	}
	if envelope.Values != nil {
		return *envelope.Values, nil
	}
	reason, err := decode[TestCaseReason](trimmed)
	if err != nil {
		return nil, err
	}
	return []TestCaseReason{reason}, nil
}

// StepTestReports returns one entry per step, in step order, carrying the
// step's test report summary or a nil Report when it published none. Steps
// that never ran are not queried.
func (r *PipelineResource) StepTestReports(ctx context.Context, pipelineUUID string, steps []PipelineStep) ([]StepTestReport, error) {
	reports := make([]StepTestReport, 0, len(steps))
	for _, step := range steps {
		entry := StepTestReport{StepUUID: step.UUID, StepName: step.Name}
		if stepRan(step) {
			report, err := r.TestReport(ctx, pipelineUUID, step.UUID)
			if err != nil {
				return nil, err
			}
			entry.Report = report
		}
		reports = append(reports, entry)
	}
	return reports, nil
}

// StepTestCases returns the test cases of every given step, in step order,
// each annotated with its step. Steps that never ran are skipped and steps
// without a test report contribute nothing.
func (r *PipelineResource) StepTestCases(ctx context.Context, pipelineUUID string, steps []PipelineStep, opts TestCaseOptions) ([]StepTestCase, error) {
	all := []StepTestCase{}
	for _, step := range steps {
		if !stepRan(step) {
			continue
		}
		cases, err := r.stepTestCases(ctx, pipelineUUID, step, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, cases...)
	}
	return all, nil
}

// stepTestCases fetches and annotates the test cases of a single step.
func (r *PipelineResource) stepTestCases(ctx context.Context, pipelineUUID string, step PipelineStep, opts TestCaseOptions) ([]StepTestCase, error) {
	cases, err := r.TestCases(ctx, pipelineUUID, step.UUID)
	if err != nil {
		return nil, err
	}
	out := make([]StepTestCase, 0, len(cases))
	for _, tc := range cases {
		failed := IsFailedTestStatus(tc.Status)
		if opts.FailedOnly && !failed {
			continue
		}
		entry := StepTestCase{StepUUID: step.UUID, StepName: step.Name, TestCase: tc}
		if opts.WithReasons && failed {
			entry.Reasons, err = r.TestCaseReasons(ctx, pipelineUUID, step.UUID, tc.UUID)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

// stepRan reports whether a step executed and so could have published a test
// report. Only steps reported as never run are excluded; any other state is
// queried, letting a 404 decide.
func stepRan(step PipelineStep) bool {
	return step.State.Result == nil || step.State.Result.Name != resultNotRun
}

// isNotFound reports whether err is an HTTP 404 from the Bitbucket API.
func isNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}
