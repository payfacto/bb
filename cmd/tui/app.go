package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/payfacto/bb/internal/history"
)

const statusClearDelay = 2 * time.Second // how long status messages remain visible

type appModel struct {
	stack       navStack
	width       int
	height      int
	statusMsg   string
	statusIsErr bool
	hist        *history.History
	cache       *listCache
}

func newApp(root View, hist *history.History, cache *listCache) *appModel {
	a := &appModel{hist: hist, cache: cache}
	a.stack.Push(root)
	return a
}

func (a *appModel) Init() tea.Cmd {
	top := a.stack.Top()
	if top != nil {
		return top.Init()
	}
	return nil
}

func (a *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a.forwardToTop(msg)

	case pushViewMsg:
		a.stack.Push(msg.view)
		initCmd := msg.view.Init()
		// Send window size to the new view
		sizeCmd := func() tea.Msg {
			return tea.WindowSizeMsg{Width: a.width, Height: a.height}
		}
		return a, tea.Batch(initCmd, sizeCmd)

	case rebuildMenuMsg:
		// Setup completed or reconfigured — replace entire stack with new home menu.
		applyTheme(msg.cfg.Theme)
		a.stack = navStack{}
		items := buildMenuItems(msg.client, msg.cfg, a.hist, a.cache)
		menu := newMenuModel(msg.cfg.Workspace, msg.cfg.Repo, items)
		a.stack.Push(menu)
		return a, menu.Init()

	case popViewMsg:
		if a.stack.Len() <= 1 {
			return a, tea.Quit
		}
		a.stack.Pop()
		return a, nil

	case showConfirmMsg:
		confirm := &confirmModel{
			message: msg.cfg.Message,
			onYes:   msg.cfg.OnYes,
		}
		a.stack.Push(confirm)
		return a, confirm.Init()

	case actionResultMsg:
		a.statusMsg = msg.message
		a.statusIsErr = !msg.success
		return a, tea.Tick(statusClearDelay, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		a.statusMsg = ""
		return a, nil

	case tea.KeyMsg:
		// Global quit — not from confirm dialogs
		if key.Matches(msg, globalKeys.Quit) {
			if _, ok := a.stack.Top().(*confirmModel); !ok {
				return a, tea.Quit
			}
		}
		return a.forwardToTop(msg)
	}

	return a.forwardToTop(msg)
}

func (a *appModel) forwardToTop(msg tea.Msg) (tea.Model, tea.Cmd) {
	top := a.stack.Top()
	if top == nil {
		return a, tea.Quit
	}
	updated, cmd := top.Update(msg)
	if v, ok := updated.(View); ok {
		a.stack.views[len(a.stack.views)-1] = v
	}
	return a, cmd
}

func (a *appModel) View() string {
	if a.stack.Len() == 0 {
		return ""
	}

	maxW := a.width
	if maxW == 0 || maxW > maxViewWidth {
		maxW = maxViewWidth
	}

	var sb strings.Builder

	// Breadcrumb
	sb.WriteString(a.stack.Breadcrumb())
	sb.WriteString("\n")
	sb.WriteString(separatorStyle.Render(strings.Repeat("─", maxW)))
	sb.WriteString("\n")

	// Current view
	sb.WriteString(a.stack.Top().View())

	// Status message
	if a.statusMsg != "" {
		sb.WriteString("\n")
		if a.statusIsErr {
			sb.WriteString(errorStyle.Render(a.statusMsg))
		} else {
			sb.WriteString(successStyle.Render(a.statusMsg))
		}
		sb.WriteString("\n")
	}

	// Help bar
	sb.WriteString("\n")
	sb.WriteString(separatorStyle.Render(strings.Repeat("─", maxW)))
	sb.WriteString("\n")
	sb.WriteString(a.renderHelpBar(maxW))

	return sb.String()
}

func (a *appModel) renderHelpBar(maxW int) string {
	top := a.stack.Top()
	if top == nil {
		return ""
	}
	sep := helpSepStyle.String()
	sepW := lipgloss.Width(sep)

	var parts []string
	usedW := 0
	for _, b := range top.ShortHelp() {
		if b.Help().Key == "" {
			continue
		}
		part := helpKeyStyle.Render(b.Help().Key) + " " + helpDescStyle.Render(b.Help().Desc)
		partW := lipgloss.Width(part)
		needed := partW
		if len(parts) > 0 {
			needed += sepW
		}
		if usedW+needed > maxW {
			break
		}
		parts = append(parts, part)
		usedW += needed
	}
	return strings.Join(parts, sep)
}
