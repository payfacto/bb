package tui

import "github.com/charmbracelet/lipgloss"

const (
	viewWidth    = 50 // default content width for separators and layout
	maxViewWidth = 80 // maximum rendered view width
)

var (
	colBlue     lipgloss.Color
	colOverlay0 lipgloss.Color
	colSurface1 lipgloss.Color
	colSurface0 lipgloss.Color
	colText     lipgloss.Color
	colRed      lipgloss.Color
	colRedBg    lipgloss.Color
	colGreen    lipgloss.Color
	colGreenBg  lipgloss.Color
	colYellow   lipgloss.Color
	colYellowBg lipgloss.Color

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
// Colors are resolved to lipgloss.Color (concrete hex strings) using the
// palette's own isDark flag, bypassing unreliable terminal background detection.
// It is safe to call at any point during the TUI lifecycle.
func applyTheme(name string) {
	pal := paletteFor(name)
	r := func(p colorPair) lipgloss.Color { return resolve(pal.isDark, p) }

	colBlue = r(pal.blue)
	colOverlay0 = r(pal.overlay0)
	colSurface1 = r(pal.surface1)
	colSurface0 = r(pal.surface0)
	colText = r(pal.text)
	colRed = r(pal.red)
	colRedBg = r(pal.redBg)
	colGreen = r(pal.green)
	colGreenBg = r(pal.greenBg)
	colYellow = r(pal.yellow)
	colYellowBg = r(pal.yellowBg)

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
