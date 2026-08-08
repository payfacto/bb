package render

import (
	"regexp"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
)

const markdownWordWrap = 100

// atlassianAttr matches Atlassian-flavoured markdown attribute blocks that
// appear after links in Bitbucket PR descriptions, for example:
//
//	{: data-inline-card='' }
//
// The example is indented so gofmt treats it as a code block and leaves it
// verbatim. Written as prose instead, gofmt's doc-comment printer reads a
// doubled single-quote as a TeX-style closing quote and rewrites it into a
// non-ASCII curly quote.
var atlassianAttr = regexp.MustCompile(`\{:[^}]*\}`)

var (
	mdOnce     sync.Once
	mdRenderer *glamour.TermRenderer
)

// WarmMarkdownRenderer initialises the glamour renderer synchronously.
// Call this before entering alt screen so the terminal background-colour
// query (WithAutoStyle) does not race with bubbletea's input loop, which
// is what caused PR descriptions to hang for several seconds in the TUI.
func WarmMarkdownRenderer() {
	mdOnce.Do(buildRenderer)
}

func buildRenderer() {
	// Build on top of the default dark style but strip the H1 background so
	// headers like "# JIRA Ticket: https://..." do not become unreadable
	// white-on-violet blocks. We also drop the Prefix/Suffix space padding
	// that pairs with the background fill.
	style := styles.DarkStyleConfig
	style.H1.BackgroundColor = nil
	style.H1.Prefix = ""
	style.H1.Suffix = ""
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
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
	input = atlassianAttr.ReplaceAllString(input, "")
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
