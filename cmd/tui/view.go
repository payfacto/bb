package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type View interface {
	tea.Model
	Title() string
	ShortHelp() []key.Binding
}
