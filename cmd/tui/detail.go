package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
)

type ActionItem struct {
	Label    string
	Info     string
	OnSelect func() tea.Cmd
	Key      *key.Binding
	Confirm  *ConfirmConfig
}

type ConfirmConfig struct {
	Message string
	OnYes   func() tea.Cmd
}

type DetailConfig struct {
	Title     string
	Content   string
	Actions   []ActionItem
	Shortcuts []key.Binding
}

type detailModel struct {
	cfg    DetailConfig
	cursor int
}

func newDetailView(cfg DetailConfig) *detailModel {
	return &detailModel{cfg: cfg}
}

func (m *detailModel) Init() tea.Cmd { return nil }

func (m *detailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, globalKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, globalKeys.Down):
			if m.cursor < len(m.cfg.Actions)-1 {
				m.cursor++
			}
		case key.Matches(msg, globalKeys.Enter):
			return m.executeAction(m.cursor)
		case key.Matches(msg, globalKeys.Back):
			return m, popView
		default:
			for i, action := range m.cfg.Actions {
				if action.Key != nil && key.Matches(msg, *action.Key) {
					return m.executeAction(i)
				}
			}
		}
	}
	return m, nil
}

func (m *detailModel) executeAction(idx int) (tea.Model, tea.Cmd) {
	if idx >= len(m.cfg.Actions) {
		return m, nil
	}
	action := m.cfg.Actions[idx]
	if action.Confirm != nil {
		return m, showConfirm(*action.Confirm)
	}
	if action.OnSelect != nil {
		return m, action.OnSelect()
	}
	return m, nil
}

func (m *detailModel) View() string {
	var sb strings.Builder
	sb.WriteString(m.cfg.Content)
	sb.WriteString("\n")
	sb.WriteString(separatorStyle.Render(strings.Repeat("─", 50)))
	sb.WriteString("\n\n")
	if len(m.cfg.Actions) > 0 {
		sb.WriteString(subtitleStyle.Render("ACTIONS"))
		sb.WriteString("\n")
		for i, action := range m.cfg.Actions {
			label := action.Label
			if action.Info != "" {
				label += " " + subtitleStyle.Render(action.Info)
			}
			if i == m.cursor {
				sb.WriteString(selectedStyle.Render(label))
			} else {
				sb.WriteString(normalStyle.Render(label))
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m *detailModel) Title() string { return m.cfg.Title }
func (m *detailModel) ShortHelp() []key.Binding {
	bindings := []key.Binding{globalKeys.Up, globalKeys.Down, globalKeys.Enter, globalKeys.Back}
	for _, action := range m.cfg.Actions {
		if action.Key != nil {
			bindings = append(bindings, *action.Key)
		}
	}
	return bindings
}

type showConfirmMsg struct{ cfg ConfirmConfig }

func showConfirm(cfg ConfirmConfig) tea.Cmd {
	return func() tea.Msg { return showConfirmMsg{cfg: cfg} }
}
