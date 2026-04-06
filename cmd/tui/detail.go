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
	cfg     DetailConfig
	cursor  int
	vp      viewport.Model
	vpReady bool
	spinner spinner.Model
	loading bool
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
			m.vp.SetContent(m.cfg.Content)
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
		// Reserve lines: 1 separator + 2 "ACTIONS" header + len(actions) + 2 padding
		actionsH := 0
		if len(m.cfg.Actions) > 0 {
			actionsH = len(m.cfg.Actions) + 3
		}
		contentH := msg.Height - 4 - actionsH // 4 = breadcrumb+sep+help bar
		if contentH < 3 {
			contentH = 3
		}
		m.vp = viewport.New(msg.Width, contentH)
		m.vp.SetContent(m.cfg.Content)
		m.vpReady = true
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, globalKeys.Up):
			if len(m.cfg.Actions) > 0 {
				if m.cursor > 0 {
					m.cursor--
				}
			} else if m.vpReady {
				var cmd tea.Cmd
				m.vp, cmd = m.vp.Update(msg)
				return m, cmd
			}
		case key.Matches(msg, globalKeys.Down):
			if len(m.cfg.Actions) > 0 {
				if m.cursor < len(m.cfg.Actions)-1 {
					m.cursor++
				}
			} else if m.vpReady {
				var cmd tea.Cmd
				m.vp, cmd = m.vp.Update(msg)
				return m, cmd
			}
		case key.Matches(msg, globalKeys.PgUp), key.Matches(msg, globalKeys.PgDn):
			if m.vpReady {
				var cmd tea.Cmd
				m.vp, cmd = m.vp.Update(msg)
				return m, cmd
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
			if m.vpReady {
				var cmd tea.Cmd
				m.vp, cmd = m.vp.Update(msg)
				return m, cmd
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
				scrollHint = subtitleStyle.Render(fmt.Sprintf("↓ scroll down (%d%%)", int(pct*100)))
			case m.vp.AtBottom():
				scrollHint = subtitleStyle.Render("↑ scroll up  (end)")
			default:
				scrollHint = subtitleStyle.Render(fmt.Sprintf("↑↓ %d%% — ctrl+u/d to scroll", int(pct*100)))
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
	sb.WriteString(separatorStyle.Render(strings.Repeat("─", viewWidth)))
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
			} else if action.Style != nil {
				sb.WriteString(action.Style.Copy().PaddingLeft(2).Render(label))
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
	bindings := []key.Binding{globalKeys.Up, globalKeys.Down, globalKeys.Enter, globalKeys.Back, globalKeys.Quit}
	if m.vpReady && len(m.cfg.Actions) > 0 {
		bindings = append([]key.Binding{globalKeys.PgUp, globalKeys.PgDn}, bindings...)
	}
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
