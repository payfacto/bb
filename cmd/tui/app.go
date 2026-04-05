package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type appModel struct {
	stack       navStack
	width       int
	height      int
	statusMsg   string
	statusIsErr bool
}

func newApp(root View) *appModel {
	a := &appModel{}
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
		return a, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
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
	if maxW == 0 || maxW > 80 {
		maxW = 80
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
	sb.WriteString(a.renderHelpBar())

	return sb.String()
}

func (a *appModel) renderHelpBar() string {
	top := a.stack.Top()
	if top == nil {
		return ""
	}
	bindings := top.ShortHelp()
	parts := make([]string, len(bindings))
	for i, b := range bindings {
		parts[i] = helpKeyStyle.Render(b.Help().Key) + " " + helpDescStyle.Render(b.Help().Desc)
	}
	return strings.Join(parts, helpSepStyle.String())
}
