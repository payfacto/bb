package render

import "github.com/charmbracelet/lipgloss"

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
	badgeOpen      = lipgloss.NewStyle().Background(lipgloss.Color("#1e3a2a")).Foreground(lipgloss.Color("#a6e3a1")).Padding(0, 1)
	badgeMerged    = lipgloss.NewStyle().Background(lipgloss.Color("#3a2e1e")).Foreground(lipgloss.Color("#f9e2af")).Padding(0, 1)
	badgeDeclined  = lipgloss.NewStyle().Background(lipgloss.Color("#3a1e1e")).Foreground(lipgloss.Color("#f38ba8")).Padding(0, 1)
	badgeOther     = lipgloss.NewStyle().Background(lipgloss.Color("#1e2e3a")).Foreground(lipgloss.Color("#89dceb")).Padding(0, 1)
)

// StateBadge returns a colored badge string for a PR or build state.
func StateBadge(state string) string {
	switch state {
	case "OPEN":
		return badgeOpen.Render(state)
	case "MERGED":
		return badgeMerged.Render(state)
	case "DECLINED":
		return badgeDeclined.Render(state)
	default:
		return badgeOther.Render(state)
	}
}
