package tui

import "github.com/charmbracelet/lipgloss"

const (
	viewWidth    = 50 // default content width for separators and layout
	maxViewWidth = 80 // maximum rendered view width
)

// adaptive returns an AdaptiveColor that uses the dark value on dark terminals
// and the light value on light terminals.
func adaptive(dark, light string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Dark: dark, Light: light}
}

var (
	// Colours — Catppuccin Mocha (dark) / Catppuccin Latte (light)
	colBlue      = adaptive("#89b4fa", "#1e66f5")
	colOverlay0  = adaptive("#6c7086", "#9ca0b0")
	colSurface1  = adaptive("#45475a", "#ccd0da")
	colSurface0  = adaptive("#313244", "#dce0e8")
	colText      = adaptive("#cdd6f4", "#4c4f69")
	colRed       = adaptive("#f38ba8", "#d20f39")
	colGreen     = adaptive("#a6e3a1", "#40a02b")
	colGreenBg   = adaptive("#1e3a2a", "#e4f0e8")
	colYellow    = adaptive("#f9e2af", "#df8e1d")

	breadcrumbStyle     = lipgloss.NewStyle().Foreground(colOverlay0)
	breadcrumbActive    = lipgloss.NewStyle().Foreground(colBlue)
	breadcrumbSep       = lipgloss.NewStyle().Foreground(colSurface1).SetString(" > ")
	selectedStyle       = lipgloss.NewStyle().Background(colSurface0).Foreground(colText).BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colBlue).PaddingLeft(1)
	normalStyle         = lipgloss.NewStyle().PaddingLeft(2)
	errorStyle          = lipgloss.NewStyle().Foreground(colRed)
	successStyle        = lipgloss.NewStyle().Foreground(colGreen)
	helpKeyStyle        = lipgloss.NewStyle().Foreground(colBlue)
	helpDescStyle       = lipgloss.NewStyle().Foreground(colOverlay0)
	helpSepStyle        = lipgloss.NewStyle().Foreground(colSurface1).SetString("  ")
	filterActiveStyle   = lipgloss.NewStyle().Background(colGreenBg).Foreground(colGreen).Padding(0, 1)
	filterInactiveStyle = lipgloss.NewStyle().Foreground(colOverlay0).Padding(0, 1)
	dialogStyle         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colBlue).Padding(1, 2).Width(40)
	headerStyle         = lipgloss.NewStyle().Foreground(colBlue).Bold(true)
	subtitleStyle       = lipgloss.NewStyle().Foreground(colOverlay0)
	separatorStyle      = lipgloss.NewStyle().Foreground(colSurface1)

	// Action intent colours
	actionSuccessStyle = lipgloss.NewStyle().Foreground(colGreen)
	actionWarnStyle    = lipgloss.NewStyle().Foreground(colYellow)
	actionDangerStyle  = lipgloss.NewStyle().Foreground(colRed)
)
