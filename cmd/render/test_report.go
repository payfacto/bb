package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	testStepNameW = 30
	testCountW    = 7
	reasonIndent  = "      "
)

// TestReportsString returns a per-step table of test counts. Steps that
// published no test report are listed with "no report".
func TestReportsString(reports []bitbucket.StepTestReport) string {
	if len(reports) == 0 {
		return "No steps found.\n"
	}
	countCols := []string{"TOTAL", "PASSED", "FAILED", "ERROR", "SKIPPED"}

	var sb strings.Builder
	sb.WriteString("  " + LabelStyle.Render(fmt.Sprintf("%-*s", testStepNameW, "STEP")))
	for _, col := range countCols {
		sb.WriteString("  " + LabelStyle.Render(fmt.Sprintf("%*s", testCountW, col)))
	}
	sb.WriteString("\n  " + SepStyle.Render(strings.Repeat("-", testStepNameW)))
	for range countCols {
		sb.WriteString("  " + SepStyle.Render(strings.Repeat("-", testCountW)))
	}
	sb.WriteString("\n")

	for _, r := range reports {
		sb.WriteString(fmt.Sprintf("  %-*s", testStepNameW, truncate(r.StepName, testStepNameW)))
		if r.Report == nil {
			sb.WriteString("  " + DimStyle.Render("no report") + "\n")
			continue
		}
		for _, n := range []int{r.Report.TotalCases, r.Report.SuccessfulCases,
			r.Report.FailedCases, r.Report.ErrorCases, r.Report.SkippedCases} {
			sb.WriteString(fmt.Sprintf("  %*d", testCountW, n))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// TestReports prints the per-step test report table to stdout.
func TestReports(reports []bitbucket.StepTestReport) { fmt.Print(TestReportsString(reports)) }

// TestCasesString returns one line per test case (status, name, duration,
// step), followed by any failure reasons indented beneath it.
func TestCasesString(cases []bitbucket.StepTestCase) string {
	if len(cases) == 0 {
		return "No test cases found.\n"
	}
	var sb strings.Builder
	for _, c := range cases {
		line := fmt.Sprintf("  %s  %s", StateBadge(c.Status), testCaseLabel(c.TestCase))
		if c.Duration != "" {
			line += "  " + DimStyle.Render(string(c.Duration))
		}
		if c.StepName != "" {
			line += "  " + DimStyle.Render("["+c.StepName+"]")
		}
		sb.WriteString(line + "\n")
		for _, reason := range c.Reasons {
			sb.WriteString(indentLines(reasonText(reason), reasonIndent))
		}
	}
	return sb.String()
}

// TestCases prints the test case list to stdout.
func TestCases(cases []bitbucket.StepTestCase) { fmt.Print(TestCasesString(cases)) }

// testCaseLabel prefers the fully qualified name, then Class.name, then name.
func testCaseLabel(tc bitbucket.TestCase) string {
	switch {
	case tc.FullyQualifiedName != "":
		return tc.FullyQualifiedName
	case tc.ClassName != "":
		return tc.ClassName + "." + tc.Name
	default:
		return tc.Name
	}
}

// reasonText returns the failure message followed by the raw output (stack
// trace, logs), omitting whichever Bitbucket did not record.
func reasonText(r bitbucket.TestCaseReason) string {
	var parts []string
	for _, text := range []string{r.Message, r.Output} {
		if trimmed := strings.TrimRight(text, "\n"); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, "\n")
}

// indentLines prefixes every line of s with indent; empty s yields "".
func indentLines(s, indent string) string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return ""
	}
	var sb strings.Builder
	for _, line := range strings.Split(s, "\n") {
		sb.WriteString(indent + line + "\n")
	}
	return sb.String()
}
