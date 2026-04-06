package tui

import "github.com/charmbracelet/bubbles/key"

type globalKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	PgUp    key.Binding
	PgDn    key.Binding
	Enter   key.Binding
	Back    key.Binding
	Filter  key.Binding
	Refresh key.Binding
	Quit    key.Binding
	Help    key.Binding
}

var globalKeys = globalKeyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	PgUp:    key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "scroll up")),
	PgDn:    key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn", "scroll down")),
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Quit:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
}

var tabKey = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "filter"))
