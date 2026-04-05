package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func Run(client *bitbucket.Client, cfg *config.Config) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("TUI requires a terminal — use 'bb <command>' for non-interactive use")
	}
	items := buildMenuItems(client, cfg)
	menu := newMenuModel(cfg.Workspace, cfg.Repo, items)
	app := newApp(menu)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
