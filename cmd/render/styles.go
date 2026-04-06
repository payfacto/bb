package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Typography
var (
	IDStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa")).Bold(true)
	LabelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	DimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	BranchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#fab387"))
	SepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a"))
)

// State badge styles
var (
	badgeOpen     = lipgloss.NewStyle().Background(lipgloss.Color("#1e3a2a")).Foreground(lipgloss.Color("#a6e3a1")).Padding(0, 1)
	badgeMerged   = lipgloss.NewStyle().Background(lipgloss.Color("#3a2e1e")).Foreground(lipgloss.Color("#f9e2af")).Padding(0, 1)
	badgeDeclined = lipgloss.NewStyle().Background(lipgloss.Color("#3a1e1e")).Foreground(lipgloss.Color("#f38ba8")).Padding(0, 1)
	badgeOther    = lipgloss.NewStyle().Background(lipgloss.Color("#1e2e3a")).Foreground(lipgloss.Color("#89dceb")).Padding(0, 1)
)

// StateBadge returns a colored badge string for a PR, pipeline, task, or issue state.
func StateBadge(state string) string {
	switch strings.ToUpper(state) {
	case "OPEN", "NEW", "ACTIVE", "SUCCESSFUL", "RESOLVED":
		return badgeOpen.Render(state)
	case "COMPLETED", "MERGED", "PENDING", "PAUSED", "UNRESOLVED", "ON HOLD", "STOPPED", "NOT_RUN":
		return badgeMerged.Render(state)
	case "IN_PROGRESS", "RUNNING":
		return badgeOther.Render(state)
	case "UNLOCKED":
		return badgeOpen.Render(state)
	case "LOCKED":
		return badgeDeclined.Render(state)
	case "DECLINED", "FAILED", "ERROR", "INACTIVE", "INVALID", "WONTFIX", "DUPLICATE":
		return badgeDeclined.Render(state)
	default:
		return badgeOther.Render(state)
	}
}
