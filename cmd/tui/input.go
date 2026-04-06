package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const inputCharLimit = 200

// inputModel is a minimal single-field text input view.
// onSubmit is called with the entered value when the user presses Enter.
type inputModel struct {
	title    string
	input    textinput.Model
	onSubmit func(value string) tea.Cmd
}

func newInputView(title, placeholder string, onSubmit func(string) tea.Cmd) *inputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = inputCharLimit
	ti.Focus()
	return &inputModel{title: title, input: ti, onSubmit: onSubmit}
}

func (m *inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			val := m.input.Value()
			if val == "" {
				return m, nil
			}
			return m, tea.Sequence(popView, m.onSubmit(val))
		case tea.KeyEsc:
			return m, popView
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *inputModel) View() string {
	return headerStyle.Render(m.title) + "\n\n" + m.input.View() + "\n\n" +
		helpKeyStyle.Render("enter") + "  confirm   " +
		helpKeyStyle.Render("esc") + "  cancel"
}

func (m *inputModel) Title() string { return m.title }

func (m *inputModel) ShortHelp() []key.Binding { return nil }
