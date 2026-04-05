package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
)

type textViewModel struct {
	title   string
	content string
	vp      viewport.Model
	ready   bool
}

func newTextView(title, content string) *textViewModel {
	return &textViewModel{title: title, content: content}
}

func (m *textViewModel) Init() tea.Cmd { return nil }

func (m *textViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.vp = viewport.New(msg.Width, msg.Height-4)
		m.vp.SetContent(m.content)
		m.ready = true
		return m, nil
	case tea.KeyMsg:
		if key.Matches(msg, globalKeys.Back) {
			return m, popView
		}
	}
	if m.ready {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *textViewModel) View() string {
	if !m.ready {
		return "\n  Loading...\n"
	}
	return m.vp.View()
}

func (m *textViewModel) Title() string { return m.title }
func (m *textViewModel) ShortHelp() []key.Binding {
	return []key.Binding{globalKeys.Up, globalKeys.Down, globalKeys.Back}
}
