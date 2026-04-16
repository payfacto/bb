package tui

import "github.com/charmbracelet/lipgloss"

const (
	viewWidth    = 50 // default content width for separators and layout
	maxViewWidth = 80 // maximum rendered view width
)

var (
	colBlue     lipgloss.AdaptiveColor
	colOverlay0 lipgloss.AdaptiveColor
	colSurface1 lipgloss.AdaptiveColor
	colSurface0 lipgloss.AdaptiveColor
	colText     lipgloss.AdaptiveColor
	colRed      lipgloss.AdaptiveColor
	colGreen    lipgloss.AdaptiveColor
	colGreenBg  lipgloss.AdaptiveColor
	colYellow   lipgloss.AdaptiveColor

	breadcrumbStyle     lipgloss.Style
	breadcrumbActive    lipgloss.Style
	breadcrumbSep       lipgloss.Style
	selectedStyle       lipgloss.Style
	normalStyle         lipgloss.Style
	errorStyle          lipgloss.Style
	successStyle        lipgloss.Style
	helpKeyStyle        lipgloss.Style
	helpDescStyle       lipgloss.Style
	helpSepStyle        lipgloss.Style
	filterActiveStyle   lipgloss.Style
	filterInactiveStyle lipgloss.Style
	dialogStyle         lipgloss.Style
	headerStyle         lipgloss.Style
	subtitleStyle       lipgloss.Style
	separatorStyle      lipgloss.Style

	actionSuccessStyle lipgloss.Style
	actionWarnStyle    lipgloss.Style
	actionDangerStyle  lipgloss.Style
)

func init() {
	applyTheme("catppuccin")
}

// applyTheme reassigns all colour and style vars to match the named palette.
// It is safe to call at any point during the TUI lifecycle.
func applyTheme(name string) {
	pal := paletteFor(name)

	colBlue = pal.blue
	colOverlay0 = pal.overlay0
	colSurface1 = pal.surface1
	colSurface0 = pal.surface0
	colText = pal.text
	colRed = pal.red
	colGreen = pal.green
	colGreenBg = pal.greenBg
	colYellow = pal.yellow

	breadcrumbStyle = lipgloss.NewStyle().Foreground(colOverlay0)
	breadcrumbActive = lipgloss.NewStyle().Foreground(colBlue)
	breadcrumbSep = lipgloss.NewStyle().Foreground(colSurface1).SetString(" > ")
	selectedStyle = lipgloss.NewStyle().
		Background(colSurface0).
		Foreground(colText).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colBlue).
		PaddingLeft(1)
	normalStyle = lipgloss.NewStyle().PaddingLeft(2)
	errorStyle = lipgloss.NewStyle().Foreground(colRed)
	successStyle = lipgloss.NewStyle().Foreground(colGreen)
	helpKeyStyle = lipgloss.NewStyle().Foreground(colBlue)
	helpDescStyle = lipgloss.NewStyle().Foreground(colOverlay0)
	helpSepStyle = lipgloss.NewStyle().Foreground(colSurface1).SetString("  ")
	filterActiveStyle = lipgloss.NewStyle().Background(colGreenBg).Foreground(colGreen).Padding(0, 1)
	filterInactiveStyle = lipgloss.NewStyle().Foreground(colOverlay0).Padding(0, 1)
	dialogStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBlue).
		Padding(1, 2).
		Width(40)
	headerStyle = lipgloss.NewStyle().Foreground(colBlue).Bold(true)
	subtitleStyle = lipgloss.NewStyle().Foreground(colOverlay0)
	separatorStyle = lipgloss.NewStyle().Foreground(colSurface1)

	actionSuccessStyle = lipgloss.NewStyle().Foreground(colGreen)
	actionWarnStyle = lipgloss.NewStyle().Foreground(colYellow)
	actionDangerStyle = lipgloss.NewStyle().Foreground(colRed)
}
