package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// loadingTextView shows a spinner while a remote fetch completes, then
// transitions to a scrollable text view when the content is ready.
type loadingTextView struct {
	title   string
	fetch   func() (string, error)
	spinner spinner.Model
	vp      viewport.Model
	width   int
	height  int
	loading bool
	err     error
	content string
}

type fetchContentMsg struct {
	content string
	err     error
}

func newLoadingTextView(title string, fetch func() (string, error)) *loadingTextView {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return &loadingTextView{
		title:   title,
		fetch:   fetch,
		spinner: sp,
		loading: true,
	}
}

func (m *loadingTextView) Init() tea.Cmd {
	fetchFn := m.fetch
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			content, err := fetchFn()
			return fetchContentMsg{content: content, err: err}
		},
	)
}

func (m *loadingTextView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.loading && m.err == nil {
			m.vp = viewport.New(msg.Width, msg.Height-4)
			m.vp.SetContent(m.content)
		}
		return m, nil

	case fetchContentMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.content = msg.content
			m.vp = viewport.New(m.width, m.height-4)
			m.vp.SetContent(m.content)
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, globalKeys.Back) {
			return m, popView
		}
	}

	if !m.loading && m.err == nil {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *loadingTextView) View() string {
	if m.loading {
		return "\n  " + m.spinner.View() + " Loading " + m.title + "...\n"
	}
	if m.err != nil {
		return "\n" + errorStyle.Render("  Error: "+m.err.Error()) + "\n" +
			subtitleStyle.Render("  Press esc to go back") + "\n"
	}
	return m.vp.View()
}

func (m *loadingTextView) Title() string { return m.title }
func (m *loadingTextView) ShortHelp() []key.Binding {
	return []key.Binding{globalKeys.Up, globalKeys.Down, globalKeys.Back, globalKeys.Quit}
}
