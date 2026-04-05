package tui

import "github.com/charmbracelet/lipgloss"

var (
	breadcrumbStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	breadcrumbActive    = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa"))
	breadcrumbSep       = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).SetString(" > ")
	selectedStyle       = lipgloss.NewStyle().Background(lipgloss.Color("#313244")).BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#89b4fa")).PaddingLeft(1)
	normalStyle         = lipgloss.NewStyle().PaddingLeft(2)
	errorStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	successStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1"))
	helpKeyStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa"))
	helpDescStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	helpSepStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).SetString("  ")
	filterActiveStyle   = lipgloss.NewStyle().Background(lipgloss.Color("#1e3a2a")).Foreground(lipgloss.Color("#a6e3a1")).Padding(0, 1)
	filterInactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).Padding(0, 1)
	dialogStyle         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#89b4fa")).Padding(1, 2).Width(40)
	headerStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa")).Bold(true)
	subtitleStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	separatorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a"))
)
