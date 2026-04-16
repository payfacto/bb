package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/internal/history"
	"github.com/payfacto/bb/pkg/bitbucket"
)

// Run starts the TUI application. If client is nil (no credentials), the setup
// wizard is shown first.
func Run(client *bitbucket.Client, cfg *config.Config, ver string) error {
	version = ver
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("TUI requires a terminal — use 'bb <command>' for non-interactive use")
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	// Warm the glamour markdown renderer before entering alt screen.
	// WithAutoStyle queries the terminal's background colour via an OSC escape
	// sequence; doing this inside bubbletea's alt screen causes it to race
	// with the input loop and can block for several seconds.
	render.WarmMarkdownRenderer()

	applyTheme(cfg.Theme)

	histPath := history.HistoryPath(config.DefaultPath())
	hist, histErr := history.Load(histPath)
	cache := newListCache()

	var root View
	if client == nil {
		root = newSetupView(config.DefaultPath(), cfg)
	} else {
		items := buildMenuItems(client, cfg, hist, cache)
		root = newMenuModel(cfg.Workspace, cfg.Repo, items)
	}

	app := newApp(root, hist, cache)
	if histErr != nil {
		app.statusMsg = fmt.Sprintf("could not load history: %v", histErr)
		app.statusIsErr = true
	}
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
