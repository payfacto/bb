package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type listItem struct {
	id       string
	title    string
	subtitle string
	data     any
}

type ListConfig struct {
	Title     string
	Fetch     func(ctx context.Context, filter string) ([]listItem, error)
	OnSelect  func(item listItem) tea.Cmd
	Filters   []string
	Shortcuts []key.Binding
}

type listModel struct {
	cfg       ListConfig
	items     []listItem
	filtered  []listItem
	cursor    int
	loading   bool
	err       error
	spinner   spinner.Model
	filter    int
	searching bool
	search    textinput.Model
}

func newListView(cfg ListConfig) *listModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 100
	return &listModel{cfg: cfg, loading: true, spinner: sp, search: ti}
}

type fetchResultMsg struct {
	items []listItem
	err   error
}

func (m *listModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchItems())
}

func (m *listModel) fetchItems() tea.Cmd {
	filter := ""
	if len(m.cfg.Filters) > 0 {
		filter = m.cfg.Filters[m.filter]
	}
	fetch := m.cfg.Fetch
	return func() tea.Msg {
		items, err := fetch(context.Background(), filter)
		return fetchResultMsg{items: items, err: err}
	}
}

func (m *listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchResultMsg:
		m.loading = false
		m.err = msg.err
		m.items = msg.items
		m.filtered = msg.items
		m.cursor = 0
		return m, nil
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case tea.KeyMsg:
		if m.searching {
			return m.updateSearch(msg)
		}
		return m.updateNavigation(msg)
	}
	return m, nil
}

func (m *listModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.searching = false
		m.filtered = m.items
		m.cursor = 0
		return m, nil
	case tea.KeyEnter:
		m.searching = false
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.applySearch()
	return m, cmd
}

func (m *listModel) applySearch() {
	query := strings.ToLower(m.search.Value())
	if query == "" {
		m.filtered = m.items
		m.cursor = 0
		return
	}
	var result []listItem
	for _, item := range m.items {
		text := strings.ToLower(item.id + " " + item.title + " " + item.subtitle)
		if strings.Contains(text, query) {
			result = append(result, item)
		}
	}
	m.filtered = result
	m.cursor = 0
}

func (m *listModel) updateNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, globalKeys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, globalKeys.Down):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case key.Matches(msg, globalKeys.Enter):
		if m.cursor < len(m.filtered) && m.cfg.OnSelect != nil {
			return m, m.cfg.OnSelect(m.filtered[m.cursor])
		}
	case key.Matches(msg, globalKeys.Back):
		return m, popView
	case key.Matches(msg, globalKeys.Filter):
		m.searching = true
		m.search.SetValue("")
		m.search.Focus()
		return m, textinput.Blink
	case key.Matches(msg, globalKeys.Refresh):
		m.loading = true
		m.err = nil
		return m, tea.Batch(m.spinner.Tick, m.fetchItems())
	case key.Matches(msg, tabKey):
		if len(m.cfg.Filters) > 0 {
			m.filter = (m.filter + 1) % len(m.cfg.Filters)
			m.loading = true
			m.cursor = 0
			return m, tea.Batch(m.spinner.Tick, m.fetchItems())
		}
	}
	return m, nil
}

func (m *listModel) View() string {
	var sb strings.Builder
	if len(m.cfg.Filters) > 0 {
		for i, f := range m.cfg.Filters {
			if i == m.filter {
				sb.WriteString(filterActiveStyle.Render(f))
			} else {
				sb.WriteString(filterInactiveStyle.Render(f))
			}
			if i < len(m.cfg.Filters)-1 {
				sb.WriteString(separatorStyle.Render(" | "))
			}
		}
		sb.WriteString("\n")
		sb.WriteString(separatorStyle.Render(strings.Repeat("─", viewWidth)))
		sb.WriteString("\n")
	}
	if m.searching {
		sb.WriteString("/ " + m.search.View())
		sb.WriteString("\n")
	}
	if m.loading {
		sb.WriteString(fmt.Sprintf("\n  %s Loading %s...\n", m.spinner.View(), m.cfg.Title))
		return sb.String()
	}
	if m.err != nil {
		sb.WriteString("\n")
		sb.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		sb.WriteString("\n")
		sb.WriteString(subtitleStyle.Render("  Press r to retry"))
		sb.WriteString("\n")
		return sb.String()
	}
	if len(m.filtered) == 0 {
		sb.WriteString(fmt.Sprintf("\n  No %s found.\n", strings.ToLower(m.cfg.Title)))
		return sb.String()
	}
	sb.WriteString("\n")
	for i, item := range m.filtered {
		var line string
		if item.id != "" && item.id != item.title {
			line = fmt.Sprintf("%-6s %s", item.id, item.title)
		} else {
			line = item.title
		}
		if item.subtitle != "" {
			line += "  " + subtitleStyle.Render(item.subtitle)
		}
		if i == m.cursor {
			sb.WriteString(selectedStyle.Render(line))
		} else {
			sb.WriteString(normalStyle.Render(line))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (m *listModel) Title() string { return m.cfg.Title }
func (m *listModel) ShortHelp() []key.Binding {
	bindings := []key.Binding{globalKeys.Up, globalKeys.Down, globalKeys.Enter}
	if len(m.cfg.Filters) > 0 {
		bindings = append(bindings, tabKey)
	}
	bindings = append(bindings, globalKeys.Filter, globalKeys.Refresh, globalKeys.Back, globalKeys.Quit)
	bindings = append(bindings, m.cfg.Shortcuts...)
	return bindings
}
