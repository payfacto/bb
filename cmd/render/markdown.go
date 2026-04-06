package render

import (
	"sync"

	"github.com/charmbracelet/glamour"
)

const markdownWordWrap = 100

var (
	mdOnce     sync.Once
	mdRenderer *glamour.TermRenderer
)

// WarmMarkdownRenderer initialises the glamour renderer synchronously.
// Call this before entering alt screen so the terminal background-colour
// query (WithAutoStyle) does not race with bubbletea's input loop — which
// is what caused PR descriptions to hang for several seconds in the TUI.
func WarmMarkdownRenderer() {
	mdOnce.Do(buildRenderer)
}

func buildRenderer() {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(markdownWordWrap),
	)
	if err == nil {
		mdRenderer = r
	}
}

// RenderMarkdown renders a markdown string using glamour with auto theme
// detection. Falls back to returning the raw input if rendering fails.
func RenderMarkdown(input string) string {
	if input == "" {
		return ""
	}
	mdOnce.Do(buildRenderer)
	if mdRenderer == nil {
		return input
	}
	out, err := mdRenderer.Render(input)
	if err != nil {
		return input
	}
	return out
}
