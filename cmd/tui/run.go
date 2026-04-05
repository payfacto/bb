package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)

// Run starts the TUI application. If client is nil (no credentials), the setup
// wizard is shown first.
func Run(client *bitbucket.Client, cfg *config.Config) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("TUI requires a terminal — use 'bb <command>' for non-interactive use")
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	var root View
	if client == nil {
		// No credentials — show setup wizard first
		root = newSetupView(config.DefaultPath(), cfg)
	} else {
		items := buildMenuItems(client, cfg)
		root = newMenuModel(cfg.Workspace, cfg.Repo, items)
	}

	app := newApp(root)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
