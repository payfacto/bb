package render

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestCodeSearchResultsString_GrepStyle(t *testing.T) {
	results := []bitbucket.CodeSearchResult{
		{
			File: bitbucket.CodeSearchFile{
				Path:   "src/foo.go",
				Commit: &bitbucket.CodeSearchCommit{Repository: &bitbucket.CodeSearchRepoRef{FullName: "ws/repo"}},
			},
			ContentMatches: []bitbucket.CodeSearchContentMatch{
				{Lines: []bitbucket.CodeSearchLine{
					{Line: 10, Segments: []bitbucket.CodeSearchSegment{
						{Text: "func "}, {Text: "parseConfig", Match: true}, {Text: "() {"},
					}},
				}},
			},
		},
	}
	out := CodeSearchResultsString(results)
	if !strings.Contains(out, "ws/repo/src/foo.go:10:") {
		t.Errorf("missing grep-style location header in:\n%s", out)
	}
	if !strings.Contains(out, "func parseConfig() {") {
		t.Errorf("missing reconstructed matched line in:\n%s", out)
	}
}

func TestCodeSearchResultsString_Empty(t *testing.T) {
	if got := CodeSearchResultsString(nil); got != "No code matches found.\n" {
		t.Errorf("empty render = %q", got)
	}
}
