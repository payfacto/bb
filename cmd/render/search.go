package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// CodeSearchResultsString returns grep-style text for code search results:
// one line per matched line, formatted as "<repo>/<path>:<line>: <content>".
func CodeSearchResultsString(results []bitbucket.CodeSearchResult) string {
	if len(results) == 0 {
		return "No code matches found.\n"
	}
	var sb strings.Builder
	for _, res := range results {
		loc := res.File.Path
		if c := res.File.Commit; c != nil && c.Repository != nil {
			repo := c.Repository.FullName
			if repo == "" {
				repo = c.Repository.Name
			}
			if repo != "" {
				loc = repo + "/" + res.File.Path
			}
		}
		for _, cm := range res.ContentMatches {
			for _, ln := range cm.Lines {
				var line strings.Builder
				for _, seg := range ln.Segments {
					line.WriteString(seg.Text)
				}
				text := strings.TrimRight(line.String(), "\r\n")
				sb.WriteString(fmt.Sprintf("%s %s\n",
					IDStyle.Render(fmt.Sprintf("%s:%d:", loc, ln.Line)),
					truncate(text, 200)))
			}
		}
	}
	return sb.String()
}

// CodeSearchResults prints the grep-style code search output to stdout.
func CodeSearchResults(results []bitbucket.CodeSearchResult) { fmt.Print(CodeSearchResultsString(results)) }
