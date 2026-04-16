package tui

import "github.com/charmbracelet/lipgloss"

// colorPair stores a dark and light hex value for a single semantic color slot.
// The correct variant is chosen at applyTheme time based on themePalette.isDark,
// bypassing lipgloss.AdaptiveColor's terminal-detection (unreliable on Windows).
type colorPair struct{ dark, light string }

// themePalette holds the semantic colour slots used across all TUI styles.
type themePalette struct {
	isDark   bool
	blue     colorPair
	overlay0 colorPair
	surface1 colorPair
	surface0 colorPair
	text     colorPair
	red      colorPair
	redBg    colorPair
	green    colorPair
	greenBg  colorPair
	yellow   colorPair
	yellowBg colorPair
}

// themeNames lists every supported theme in display order.
var themeNames = []string{
	"catppuccin",
	"tokyo-night",
	"dracula",
	"gruvbox",
	"nord",
	"rose-pine",
	"one-dark",
	"facto",
}

// themeDisplayNames maps internal theme names to human-readable labels.
var themeDisplayNames = map[string]string{
	"catppuccin":  "Catppuccin",
	"tokyo-night": "Tokyo Night",
	"dracula":     "Dracula",
	"gruvbox":     "Gruvbox",
	"nord":        "Nord",
	"rose-pine":   "Rosé Pine",
	"one-dark":    "One Dark",
	"facto":       "Facto",
}

func col(dark, light string) colorPair { return colorPair{dark, light} }

// resolve returns the lipgloss.Color for a colorPair using the palette's isDark flag.
func resolve(isDark bool, p colorPair) lipgloss.Color {
	if isDark {
		return lipgloss.Color(p.dark)
	}
	return lipgloss.Color(p.light)
}

var palettes = map[string]themePalette{
	// Catppuccin Mocha (dark) / Catppuccin Latte (light)
	"catppuccin": {
		isDark:   true,
		blue:     col("#89b4fa", "#1e66f5"),
		overlay0: col("#6c7086", "#9ca0b0"),
		surface1: col("#45475a", "#ccd0da"),
		surface0: col("#313244", "#dce0e8"),
		text:     col("#cdd6f4", "#4c4f69"),
		red:      col("#f38ba8", "#d20f39"),
		redBg:    col("#3b1a22", "#fce8ec"),
		green:    col("#a6e3a1", "#40a02b"),
		greenBg:  col("#1e3a2a", "#e4f0e8"),
		yellow:   col("#f9e2af", "#df8e1d"),
		yellowBg: col("#3a2e10", "#fdf3e0"),
	},
	// Tokyo Night Storm (dark) / Tokyo Night Day (light)
	"tokyo-night": {
		isDark:   true,
		blue:     col("#7aa2f7", "#2e7de9"),
		overlay0: col("#565f89", "#9699a3"),
		surface1: col("#3b4261", "#c4c8da"),
		surface0: col("#292e42", "#d5d6db"),
		text:     col("#c0caf5", "#343b58"),
		red:      col("#f7768e", "#8c4351"),
		redBg:    col("#2d1a20", "#f8e8ec"),
		green:    col("#9ece6a", "#485e30"),
		greenBg:  col("#1a2a1a", "#e0e4d0"),
		yellow:   col("#e0af68", "#8f5e15"),
		yellowBg: col("#30270e", "#faf0dc"),
	},
	// Dracula
	"dracula": {
		isDark:   true,
		blue:     col("#bd93f9", "#6272a4"),
		overlay0: col("#6272a4", "#7a8196"),
		surface1: col("#44475a", "#c8cbda"),
		surface0: col("#383a59", "#dfe0e8"),
		text:     col("#f8f8f2", "#282a36"),
		red:      col("#ff5555", "#cc1111"),
		redBg:    col("#2d1010", "#fce8e8"),
		green:    col("#50fa7b", "#218a21"),
		greenBg:  col("#1a2e1a", "#dff0df"),
		yellow:   col("#f1fa8c", "#b5a000"),
		yellowBg: col("#2e2d10", "#fdfde0"),
	},
	// Gruvbox Dark / Gruvbox Light
	"gruvbox": {
		isDark:   true,
		blue:     col("#83a598", "#076678"),
		overlay0: col("#928374", "#7c6f64"),
		surface1: col("#3c3836", "#d5c4a1"),
		surface0: col("#32302f", "#ebdbb2"),
		text:     col("#ebdbb2", "#3c3836"),
		red:      col("#fb4934", "#cc241d"),
		redBg:    col("#2d1010", "#fce8e8"),
		green:    col("#b8bb26", "#79740e"),
		greenBg:  col("#1d2021", "#f2e5bc"),
		yellow:   col("#fabd2f", "#b57614"),
		yellowBg: col("#2e2710", "#fdf4dc"),
	},
	// Nord
	"nord": {
		isDark:   true,
		blue:     col("#81a1c1", "#5e81ac"),
		overlay0: col("#616e88", "#8190a8"),
		surface1: col("#3b4252", "#c8d0e0"),
		surface0: col("#2e3440", "#d8dee9"),
		text:     col("#d8dee9", "#2e3440"),
		red:      col("#bf616a", "#b02020"),
		redBg:    col("#2a1a1c", "#fce8e8"),
		green:    col("#a3be8c", "#4c6a2a"),
		greenBg:  col("#1a2a1a", "#dce8d0"),
		yellow:   col("#ebcb8b", "#8a6000"),
		yellowBg: col("#2e2910", "#fdf4dc"),
	},
	// Rosé Pine / Rosé Pine Dawn
	"rose-pine": {
		isDark:   true,
		blue:     col("#c4a7e7", "#907aa9"),
		overlay0: col("#6e6a86", "#9893a5"),
		surface1: col("#26233a", "#f2e9e1"),
		surface0: col("#1f1d2e", "#faf4ed"),
		text:     col("#e0def4", "#575279"),
		red:      col("#eb6f92", "#b4637a"),
		redBg:    col("#2a1520", "#fce8ed"),
		green:    col("#9ccfd8", "#286983"),
		greenBg:  col("#1a1f2e", "#eef4f0"),
		yellow:   col("#f6c177", "#ea9d34"),
		yellowBg: col("#2e2510", "#fdf5e0"),
	},
	// One Dark Pro
	"one-dark": {
		isDark:   true,
		blue:     col("#61afef", "#4078f2"),
		overlay0: col("#5c6370", "#9da5b4"),
		surface1: col("#3e4451", "#ccd0d8"),
		surface0: col("#2c313c", "#e5e5e6"),
		text:     col("#abb2bf", "#383a42"),
		red:      col("#e06c75", "#e45649"),
		redBg:    col("#2a1518", "#fce8e8"),
		green:    col("#98c379", "#50a14f"),
		greenBg:  col("#1e2a1e", "#e0edd8"),
		yellow:   col("#e5c07b", "#c18401"),
		yellowBg: col("#2e2910", "#fdf4dc"),
	},
	// Facto — PayFacto brand palette (light theme).
	// White bg, darkened gold accent (#7a5c2e), near-black text.
	// Raw gold (#bc955c) is only 2.8:1 on white, so the light accent is
	// darkened to ~4.7:1. Warning amber (#d4a047 / #8a6000) is kept
	// visually distinct from the gold accent to avoid confusion.
	"facto": {
		isDark:   false,
		blue:     col("#bc955c", "#7a5c2e"), // PayFacto Gold / darkened gold
		overlay0: col("#7a7a7a", "#535353"), // medium grey / PayFacto Dark Grey
		surface1: col("#3d3d3d", "#c8c8c8"), // dark separator / light separator
		surface0: col("#1e1e1e", "#e8e0d5"), // near-black select bg / warm beige
		text:     col("#e6e6e6", "#1a1a1a"), // PayFacto Light Grey / near-black
		red:      col("#c0392b", "#c0392b"), // error red (brand has none; derived)
		redBg:    col("#2d1010", "#fce8e8"),
		green:    col("#5cbc95", "#3a9e72"), // PayFacto complementary Green / darkened
		greenBg:  col("#1a2e24", "#d9f0e8"), // dark green tint / light green tint
		yellow:   col("#d4a047", "#8a6000"), // warm amber / dark amber
		yellowBg: col("#2e2710", "#fdf0d8"),
	},
}

// paletteFor returns the palette for name, falling back to catppuccin.
func paletteFor(name string) themePalette {
	if pal, ok := palettes[name]; ok {
		return pal
	}
	return palettes["catppuccin"]
}

// themeIndex returns the index of name in themeNames, or 0 if not found.
func themeIndex(name string) int {
	for i, n := range themeNames {
		if n == name {
			return i
		}
	}
	return 0
}
