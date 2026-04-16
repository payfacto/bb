package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// actionsKey opens the actions popup from any detail view.
var actionsKey = key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "actions"))

// popupColW is the inner content width for each column in the actions popup.
const popupColW = 22

// popupTwoCols switches to a two-column popup layout when there are more
// than this many actions.
const popupTwoCols = 5

type ActionItem struct {
	Label    string
	Info     string
	Style    *lipgloss.Style // optional; tints the label when not selected
	OnSelect func() tea.Cmd
	Key      *key.Binding
	Confirm  *ConfirmConfig
}

type ConfirmConfig struct {
	Message string
	OnYes   func() tea.Cmd
}

type DetailConfig struct {
	Title        string
	Content      string
	ContentFetch func() string // if set, called in Init() off the main goroutine
	Actions      []ActionItem
}

type detailModel struct {
	cfg       DetailConfig
	cursor    int
	width     int // terminal width, clamped to maxViewWidth
	height    int // terminal height
	vp        viewport.Model
	vpReady   bool
	spinner   spinner.Model
	loading   bool
	popupOpen bool
}

type detailContentMsg struct{ content string }

func newDetailView(cfg DetailConfig) *detailModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	loading := cfg.ContentFetch != nil
	return &detailModel{cfg: cfg, spinner: sp, loading: loading}
}

func (m *detailModel) Init() tea.Cmd {
	if m.cfg.ContentFetch != nil {
		fetch := m.cfg.ContentFetch
		return tea.Batch(
			m.spinner.Tick,
			func() tea.Msg { return detailContentMsg{content: fetch()} },
		)
	}
	return nil
}

func (m *detailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case detailContentMsg:
		m.loading = false
		m.cfg.Content = msg.content
		if m.vpReady {
			m.vp.SetContent(wrapAtWidth(m.cfg.Content, m.width))
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.WindowSizeMsg:
		w := msg.Width
		if w == 0 || w > maxViewWidth {
			w = maxViewWidth
		}
		m.width = w
		m.height = msg.Height
		contentH := msg.Height - 4 // 4 = breadcrumb+sep+help bar
		if contentH < 3 {
			contentH = 3
		}
		m.vp = viewport.New(w, contentH)
		m.vp.SetContent(wrapAtWidth(m.cfg.Content, w))
		m.vpReady = true
		return m, nil

	case tea.KeyMsg:
		if m.popupOpen {
			return m.updatePopup(msg)
		}
		return m.updateContent(msg)
	}
	return m, nil
}

func (m *detailModel) updatePopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, globalKeys.Back), key.Matches(msg, actionsKey):
		m.popupOpen = false
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
	}
	return m, nil
}

func (m *detailModel) updateContent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, actionsKey):
		if len(m.cfg.Actions) > 0 {
			m.popupOpen = true
			m.cursor = 0
		}
	case key.Matches(msg, globalKeys.PgUp), key.Matches(msg, globalKeys.PgDn):
		if m.vpReady {
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
	case key.Matches(msg, globalKeys.Up), key.Matches(msg, globalKeys.Down):
		if m.vpReady {
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
	case key.Matches(msg, globalKeys.Back):
		return m, popView
	default:
		// Action key shortcuts (e.g. 'd' for diff).
		for i, action := range m.cfg.Actions {
			if action.Key != nil && key.Matches(msg, *action.Key) {
				return m.executeAction(i)
			}
		}
		if m.vpReady {
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
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
	if m.popupOpen {
		return m.renderActionsPopup()
	}

	var sb strings.Builder
	if m.loading {
		sb.WriteString("\n  " + m.spinner.View() + " Loading...\n")
	} else if m.vpReady {
		sb.WriteString(m.vp.View())
		if pct := m.vp.ScrollPercent(); pct >= 0 {
			var scrollHint string
			switch {
			case m.vp.AtTop() && m.vp.AtBottom():
				// fits on screen — no hint needed
			case m.vp.AtTop():
				scrollHint = subtitleStyle.Render(fmt.Sprintf("↓ %d%%", int(pct*100)))
			case m.vp.AtBottom():
				scrollHint = subtitleStyle.Render("↑ end")
			default:
				scrollHint = subtitleStyle.Render(fmt.Sprintf("↑↓ %d%%", int(pct*100)))
			}
			if scrollHint != "" {
				sb.WriteString("\n")
				sb.WriteString(scrollHint)
			}
		}
	} else {
		sb.WriteString(m.cfg.Content)
	}
	sb.WriteString("\n")
	return sb.String()
}

func (m *detailModel) renderActionsPopup() string {
	actions := m.cfg.Actions
	twoCols := len(actions) > popupTwoCols

	var rows []string
	if twoCols {
		half := (len(actions) + 1) / 2
		for i := 0; i < half; i++ {
			left := m.renderActionCell(i, popupColW)
			var right string
			if i+half < len(actions) {
				right = m.renderActionCell(i+half, popupColW)
			} else {
				right = normalStyle.Render(strings.Repeat(" ", popupColW))
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
				left,
				separatorStyle.Render(" │ "),
				right,
			))
		}
	} else {
		for i := range actions {
			rows = append(rows, m.renderActionCell(i, popupColW))
		}
	}

	title := headerStyle.Render("ACTIONS")
	body := title + "\n" + strings.Join(rows, "\n")
	popup := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBlue).
		Padding(0, 1).
		Render(body)

	h := m.height - 4
	if h < 1 {
		h = 12
	}
	return lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center, popup)
}

func (m *detailModel) renderActionCell(idx, colW int) string {
	if idx >= len(m.cfg.Actions) {
		return normalStyle.Render(strings.Repeat(" ", colW))
	}
	action := m.cfg.Actions[idx]
	label := truncateStr(action.Label, colW)
	if idx == m.cursor {
		return selectedStyle.Render(fmt.Sprintf("%-*s", colW, label))
	}
	if action.Style != nil {
		return (*action.Style).PaddingLeft(1).Render(label)
	}
	return normalStyle.Render(label)
}

func (m *detailModel) Title() string { return m.cfg.Title }

func (m *detailModel) ShortHelp() []key.Binding {
	if m.popupOpen {
		return []key.Binding{globalKeys.Up, globalKeys.Down, globalKeys.Enter, globalKeys.Back}
	}
	bindings := []key.Binding{globalKeys.PgUp, globalKeys.PgDn, globalKeys.Up, globalKeys.Down}
	if len(m.cfg.Actions) > 0 {
		bindings = append(bindings, actionsKey)
	}
	bindings = append(bindings, globalKeys.Back, globalKeys.Quit)
	for _, action := range m.cfg.Actions {
		if action.Key != nil {
			bindings = append(bindings, *action.Key)
		}
	}
	return bindings
}

// wrapAtWidth pre-wraps s so that lines longer than width are soft-wrapped
// rather than clipped by the viewport. lipgloss handles ANSI sequences correctly.
func wrapAtWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	return lipgloss.NewStyle().Width(width).Render(s)
}

type showConfirmMsg struct{ cfg ConfirmConfig }

func showConfirm(cfg ConfirmConfig) tea.Cmd {
	return func() tea.Msg { return showConfirmMsg{cfg: cfg} }
}
