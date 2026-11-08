package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	reportIDWidth       = 38
	reportTypeWidth     = 10
	reportResultWidth   = 8
	annotationSevWidth  = 8
	annotationTypeWidth = 13
	annotationLocWidth  = 40
)

// reportID returns the identifier users pass back to --report-id: the
// creator-assigned external_id when present, otherwise the UUID.
func reportID(r bitbucket.Report) string {
	if r.ExternalID != "" {
		return r.ExternalID
	}
	return r.UUID
}

// ReportListString returns the formatted text for a list of Code Insights reports.
func ReportListString(reports []bitbucket.Report) string {
	if len(reports) == 0 {
		return "No reports found.\n"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-*s", reportIDWidth, "ID")),
		LabelStyle.Render(fmt.Sprintf("%-*s", reportTypeWidth, "TYPE")),
		LabelStyle.Render(fmt.Sprintf("%-*s", reportResultWidth, "RESULT")),
		LabelStyle.Render("TITLE")))
	sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", reportIDWidth)),
		SepStyle.Render(strings.Repeat("─", reportTypeWidth)),
		SepStyle.Render(strings.Repeat("─", reportResultWidth)),
		SepStyle.Render(strings.Repeat("─", 40))))
	for _, r := range reports {
		sb.WriteString(fmt.Sprintf("  %s  %-*s  %s  %s\n",
			IDStyle.Render(fmt.Sprintf("%-*s", reportIDWidth, truncate(reportID(r), reportIDWidth))),
			reportTypeWidth, r.ReportType,
			StateBadge(r.Result),
			r.Title))
	}
	return sb.String()
}

// ReportList prints the formatted report list to stdout.
func ReportList(reports []bitbucket.Report) { fmt.Print(ReportListString(reports)) }

// ReportDetailString returns the full detail string for a single report,
// including its data fields.
func ReportDetailString(r bitbucket.Report) string {
	var sb strings.Builder
	field := func(label, value string) {
		if value != "" {
			sb.WriteString(fmt.Sprintf("  %s  %s\n", LabelStyle.Render(fmt.Sprintf("%-12s", label)), value))
		}
	}
	field("Title:", IDStyle.Render(r.Title))
	field("External ID:", r.ExternalID)
	field("UUID:", DimStyle.Render(r.UUID))
	field("Reporter:", r.Reporter)
	field("Type:", r.ReportType)
	if r.Result != "" {
		field("Result:", StateBadge(r.Result))
	}
	field("Link:", r.Link)
	field("Created:", r.CreatedOn)
	field("Updated:", r.UpdatedOn)
	if r.Details != "" {
		sb.WriteString("\n  " + r.Details + "\n")
	}
	if len(r.Data) > 0 {
		sb.WriteString("\n")
		for _, d := range r.Data {
			sb.WriteString(fmt.Sprintf("  %s  %v\n", LabelStyle.Render(fmt.Sprintf("%-24s", d.Title+":")), d.Value))
		}
	}
	return sb.String()
}

// ReportDetail prints the formatted report detail to stdout.
func ReportDetail(r bitbucket.Report) { fmt.Print(ReportDetailString(r)) }

// annotationLocation renders "path:line", "path", or "-" for report-wide findings.
func annotationLocation(a bitbucket.ReportAnnotation) string {
	switch {
	case a.Path == "":
		return "-"
	case a.Line > 0:
		return fmt.Sprintf("%s:%d", a.Path, a.Line)
	default:
		return a.Path
	}
}

// ReportAnnotationListString returns the formatted text for a report's annotations.
func ReportAnnotationListString(annotations []bitbucket.ReportAnnotation) string {
	if len(annotations) == 0 {
		return "No annotations found.\n"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-*s", annotationSevWidth, "SEVERITY")),
		LabelStyle.Render(fmt.Sprintf("%-*s", annotationTypeWidth, "TYPE")),
		LabelStyle.Render(fmt.Sprintf("%-*s", annotationLocWidth, "LOCATION")),
		LabelStyle.Render("SUMMARY")))
	sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", annotationSevWidth)),
		SepStyle.Render(strings.Repeat("─", annotationTypeWidth)),
		SepStyle.Render(strings.Repeat("─", annotationLocWidth)),
		SepStyle.Render(strings.Repeat("─", 40))))
	for _, a := range annotations {
		sb.WriteString(fmt.Sprintf("  %-*s  %-*s  %-*s  %s\n",
			annotationSevWidth, a.Severity,
			annotationTypeWidth, a.AnnotationType,
			annotationLocWidth, truncate(annotationLocation(a), annotationLocWidth),
			a.Summary))
	}
	return sb.String()
}

// ReportAnnotationList prints the formatted annotation list to stdout.
func ReportAnnotationList(annotations []bitbucket.ReportAnnotation) {
	fmt.Print(ReportAnnotationListString(annotations))
}
