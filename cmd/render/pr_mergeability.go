package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/payfacto/bb/pkg/bitbucket"
)

const (
	checkStatusWidth      = 7  // widest status values: "PENDING", "SKIPPED", "UNKNOWN"
	checkTypeWidth        = 29 // widest check type: "current_user_permission_check"
	conflictPathWidth     = 50
	conflictScenarioWidth = 14 // widest scenario: "directory_file"/"file_directory"
	detailSeparator       = "; "
)

var (
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Bold(true)
)

// PRMergeabilityString returns formatted text for a PR's mergeability checks:
// one line per check followed by an overall "mergeable: yes/no" verdict.
func PRMergeabilityString(m bitbucket.PRMergeability) string {
	var sb strings.Builder
	if len(m.Checks) == 0 {
		sb.WriteString("No checks reported.\n")
	}
	for _, c := range m.Checks {
		sb.WriteString(mergeabilityCheckLine(c))
	}
	sb.WriteString(fmt.Sprintf("\n%s %s\n", LabelStyle.Render("mergeable:"), yesNo(m.Mergeable)))
	return sb.String()
}

// PRMergeability prints the formatted mergeability checks to stdout.
func PRMergeability(m bitbucket.PRMergeability) {
	fmt.Print(PRMergeabilityString(m))
}

func mergeabilityCheckLine(c bitbucket.MergeabilityCheck) string {
	line := fmt.Sprintf("  %s  %-*s  %s",
		paddedStatusBadge(c.Status), checkTypeWidth, c.Type, checkDetail(c))
	if c.Blocking {
		line += "  " + failStyle.Render("BLOCKING")
	}
	return strings.TrimRight(line, " ") + "\n"
}

// paddedStatusBadge pads after the badge so the padding is not coloured and
// StateBadge still sees the raw status for its colour lookup.
func paddedStatusBadge(status string) string {
	padding := max(checkStatusWidth-len(status), 0)
	return StateBadge(status) + strings.Repeat(" ", padding)
}

// checkDetail joins every populated per-type field of a check, so new or
// unknown check types still show whatever detail they carry.
func checkDetail(c bitbucket.MergeabilityCheck) string {
	var parts []string
	if c.State != "" {
		parts = append(parts, "state: "+c.State)
	}
	if c.Reason != "" {
		parts = append(parts, "reason: "+c.Reason)
	}
	if name := checkName(c.Check); name != "" {
		parts = append(parts, name)
	}
	if len(c.Observed) > 0 {
		parts = append(parts, "observed "+formatKeyValues(c.Observed))
	}
	if len(c.Requirement) > 0 {
		parts = append(parts, "required "+formatKeyValues(c.Requirement))
	}
	if c.Message != "" {
		parts = append(parts, c.Message)
	}
	if c.MergeQueue != nil {
		parts = append(parts, fmt.Sprintf("queue: %s (%s)", c.MergeQueue.Name, c.MergeQueue.State))
	}
	if c.Queued {
		parts = append(parts, "queued")
	}
	return strings.Join(parts, detailSeparator)
}

func checkName(def *bitbucket.MergeCheckDefinition) string {
	if def == nil {
		return ""
	}
	if def.Kind != "" {
		return def.Kind
	}
	return def.Name
}

// formatKeyValues renders a map as "k1=v1, k2=v2" in key order for stable output.
func formatKeyValues(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = fmt.Sprintf("%s=%v", k, values[k])
	}
	return strings.Join(pairs, ", ")
}

func yesNo(b bool) string {
	if b {
		return successStyle.Render("yes")
	}
	return failStyle.Render("no")
}

// PRConflictsString returns formatted text for a PR's file conflicts.
func PRConflictsString(conflicts []bitbucket.FileConflict) string {
	if len(conflicts) == 0 {
		return "No conflicts.\n"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-*s", conflictPathWidth, "PATH")),
		LabelStyle.Render(fmt.Sprintf("%-*s", conflictScenarioWidth, "SCENARIO")),
		LabelStyle.Render("MESSAGE"),
	))
	for _, c := range conflicts {
		line := fmt.Sprintf("  %-*s  %-*s  %s",
			conflictPathWidth, truncate(c.Path, conflictPathWidth),
			conflictScenarioWidth, c.Scenario,
			DimStyle.Render(c.Message))
		sb.WriteString(strings.TrimRight(line, " ") + "\n")
	}
	return sb.String()
}

// PRConflicts prints the formatted file conflicts to stdout.
func PRConflicts(conflicts []bitbucket.FileConflict) {
	fmt.Print(PRConflictsString(conflicts))
}
