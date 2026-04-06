package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type menuItem struct {
	label       string
	description string
	onSelect    func() View
}

type menuModel struct {
	items    []menuItem
	cursor   int
	ws       string
	repoSlug string
}

func newMenuModel(ws, repo string, items []menuItem) *menuModel {
	return &menuModel{items: items, ws: ws, repoSlug: repo}
}

func (m *menuModel) Init() tea.Cmd { return nil }

func (m *menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, globalKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, globalKeys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, globalKeys.Enter):
			if m.cursor < len(m.items) {
				item := m.items[m.cursor]
				if item.onSelect != nil {
					return m, pushViewCmd(item.onSelect())
				}
			}
		}
	}
	return m, nil
}

func (m *menuModel) View() string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("bb — Bitbucket Cloud CLI"))
	sb.WriteString("\n")
	sb.WriteString(subtitleStyle.Render(fmt.Sprintf("Workspace: %s  Repo: %s", m.ws, m.repoSlug)))
	sb.WriteString("\n")
	sb.WriteString(separatorStyle.Render(strings.Repeat("─", viewWidth)))
	sb.WriteString("\n\n")
	for i, item := range m.items {
		line := fmt.Sprintf("%-20s %s", item.label, subtitleStyle.Render(item.description))
		if i == m.cursor {
			sb.WriteString(selectedStyle.Render(line))
		} else {
			sb.WriteString(normalStyle.Render(line))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (m *menuModel) Title() string { return "Home" }
func (m *menuModel) ShortHelp() []key.Binding {
	return []key.Binding{globalKeys.Up, globalKeys.Down, globalKeys.Enter, globalKeys.Quit}
}

// Messages for navigation between views.

type pushViewMsg struct{ view View }

func pushViewCmd(v View) tea.Cmd {
	return func() tea.Msg { return pushViewMsg{view: v} }
}

type popViewMsg struct{}

func popView() tea.Msg { return popViewMsg{} }
