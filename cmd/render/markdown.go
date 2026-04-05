package render

import "github.com/charmbracelet/glamour"

// RenderMarkdown renders a markdown string using glamour with auto theme
// detection. Falls back to returning the raw input if rendering fails.
func RenderMarkdown(input string) string {
	if input == "" {
		return ""
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return input
	}
	out, err := r.Render(input)
	if err != nil {
		return input
	}
	return out
}
