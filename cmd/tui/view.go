package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type View interface {
	tea.Model
	Title() string
	ShortHelp() []key.Binding
}

// textCapturer is implemented by views that consume free-form text input and
// therefore must receive every key press, including ones bound to global
// single-key shortcuts like 'q'.
type textCapturer interface {
	CapturesText() bool
}

// topCapturesText reports whether v is currently capturing free-form text.
func topCapturesText(v View) bool {
	tc, ok := v.(textCapturer)
	return ok && tc.CapturesText()
}
