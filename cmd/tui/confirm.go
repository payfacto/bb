package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/key"
)

type confirmModel struct {
	message string
	onYes   func() tea.Cmd
}

func (m *confirmModel) Init() tea.Cmd { return nil }

func (m *confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			if m.onYes != nil {
				return m, m.onYes()
			}
			return m, popView
		case "n", "N", "esc":
			return m, popView
		}
	}
	return m, nil
}

func (m *confirmModel) View() string {
	content := m.message + "\n\n" + helpKeyStyle.Render("[Y]") + helpDescStyle.Render("es  ") +
		helpKeyStyle.Render("[N]") + helpDescStyle.Render("o")
	return "\n\n" + dialogStyle.Render(content) + "\n"
}

func (m *confirmModel) Title() string { return "Confirm" }
func (m *confirmModel) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm")),
		key.NewBinding(key.WithKeys("n"), key.WithHelp("n/esc", "cancel")),
	}
}

type actionResultMsg struct {
	success bool
	message string
}

type clearStatusMsg struct{}
