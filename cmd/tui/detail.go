package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	// Disabled greys the action out in the popup and blocks activation.
	// Use for actions that are not applicable to the current resource state
	// (e.g. Approve/Merge/Decline on a closed PR).
	Disabled bool
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
		// Action key shortcuts (e.g. 'd' for diff). Disabled actions are ignored.
		for i, action := range m.cfg.Actions {
			if action.Disabled {
				continue
			}
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
	if action.Disabled {
		// Silently ignore — the popup already shows the action as greyed out.
		return m, nil
	}
	if action.Confirm != nil {
		return m, showConfirm(*action.Confirm)
	}
	if action.OnSelect != nil {
		return m, action.OnSelect()
	}
	return m, nil
}

func (m *detailModel) View() string {
	bg := m.renderContent()
	if m.popupOpen {
		popup := m.renderActionsPopup()
		return overlayCenter(bg, popup, m.width, m.contentHeight())
	}
	return bg
}

// renderContent returns the detail content (viewport or loading state)
// without the popup overlay.
func (m *detailModel) renderContent() string {
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

// contentHeight returns the number of rows available to the detail content area.
func (m *detailModel) contentHeight() int {
	h := m.height - 4 // 4 = breadcrumb+sep+help bar
	if h < 3 {
		h = 3
	}
	return h
}

func (m *detailModel) renderActionsPopup() string {
	actions := m.cfg.Actions
	twoCols := len(actions) > popupTwoCols

	var rows []string
	sepStyle := lipgloss.NewStyle().Background(colSurface0).Foreground(colSurface1)
	if twoCols {
		half := (len(actions) + 1) / 2
		for i := 0; i < half; i++ {
			left := m.renderActionCell(i, popupColW)
			var right string
			if i+half < len(actions) {
				right = m.renderActionCell(i+half, popupColW)
			} else {
				right = lipgloss.NewStyle().Width(popupColW).Background(colSurface0).Render("")
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
				left,
				sepStyle.Render(" │ "),
				right,
			))
		}
	} else {
		for i := range actions {
			rows = append(rows, m.renderActionCell(i, popupColW))
		}
	}

	titleW := lipgloss.Width(rows[0])
	title := lipgloss.NewStyle().
		Width(titleW).
		Background(colSurface0).
		Foreground(colBlue).
		Bold(true).
		Render("ACTIONS")
	body := title + "\n" + strings.Join(rows, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBlue).
		Background(colSurface0).
		Padding(0, 1).
		Render(body)
}

func (m *detailModel) renderActionCell(idx, colW int) string {
	// colW is the total cell width including the 1-char left pad.
	// All cells use the same base style so the column widths never shift.
	// Background is always colSurface0 (the popup surface) so padding blends with
	// the popup rather than showing through to the detail content underneath.
	contentW := colW - 1
	base := lipgloss.NewStyle().Width(contentW).PaddingLeft(1).Background(colSurface0)

	if idx >= len(m.cfg.Actions) {
		return base.Render("")
	}
	action := m.cfg.Actions[idx]
	label := truncateStr(action.Label, contentW)
	if idx == m.cursor {
		// Selected row uses the brighter surface1 so it stands out against the popup body.
		// Disabled selected rows stay muted so users can still see they're inactive.
		if action.Disabled {
			return base.Background(colSurface1).Foreground(colOverlay0).Faint(true).Render(label)
		}
		return base.Background(colSurface1).Foreground(colText).Bold(true).Render(label)
	}
	if action.Disabled {
		return base.Foreground(colOverlay0).Faint(true).Render(label)
	}
	if action.Style != nil {
		return base.Foreground((*action.Style).GetForeground()).Render(label)
	}
	return base.Foreground(colText).Render(label)
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

// overlayCenter composites fg on top of bg, centered within a canvas of
// (canvasW x canvasH) cells. Returns the composited string with fg's lines
// replacing the corresponding cells of bg at the centered position. ANSI
// sequences in both layers are preserved. The canvas is padded with spaces
// so the popup can land on rows/cols that bg does not occupy.
func overlayCenter(bg, fg string, canvasW, canvasH int) string {
	fgLines := strings.Split(fg, "\n")
	fgH := len(fgLines)
	fgW := 0
	for _, l := range fgLines {
		if w := ansi.StringWidth(l); w > fgW {
			fgW = w
		}
	}
	x := (canvasW - fgW) / 2
	if x < 0 {
		x = 0
	}
	y := (canvasH - fgH) / 2
	if y < 0 {
		y = 0
	}
	return overlayAt(bg, fg, x, y, canvasW, canvasH)
}

// overlayAt pastes fg on top of bg at (x, y). bg is padded so it has at
// least canvasH rows, each canvasW cells wide, before compositing.
func overlayAt(bg, fg string, x, y, canvasW, canvasH int) string {
	bgLines := strings.Split(bg, "\n")
	// Trim a trailing empty line produced by a final "\n" so padding lines up.
	if n := len(bgLines); n > 0 && bgLines[n-1] == "" {
		bgLines = bgLines[:n-1]
	}
	// Pad bg height.
	for len(bgLines) < canvasH {
		bgLines = append(bgLines, "")
	}
	// Pad each bg line to canvasW cells so overlay position is valid.
	for i, line := range bgLines {
		w := ansi.StringWidth(line)
		if w < canvasW {
			bgLines[i] = line + strings.Repeat(" ", canvasW-w)
		}
	}

	fgLines := strings.Split(fg, "\n")
	for i, fgLine := range fgLines {
		row := y + i
		if row < 0 || row >= len(bgLines) {
			continue
		}
		fgW := ansi.StringWidth(fgLine)
		left := ansi.Truncate(bgLines[row], x, "")
		// Pad left if bg line is shorter than x.
		if lw := ansi.StringWidth(left); lw < x {
			left += strings.Repeat(" ", x-lw)
		}
		right := ansi.TruncateLeft(bgLines[row], x+fgW, "")
		bgLines[row] = left + fgLine + ansi.ResetStyle + right
	}
	return strings.Join(bgLines, "\n")
}

type showConfirmMsg struct{ cfg ConfirmConfig }

func showConfirm(cfg ConfirmConfig) tea.Cmd {
	return func() tea.Msg { return showConfirmMsg{cfg: cfg} }
}
