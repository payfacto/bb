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

func TestCodeSearchResultsString_NoContentMatches(t *testing.T) {
	results := []bitbucket.CodeSearchResult{
		{
			File: bitbucket.CodeSearchFile{
				Path: "cmd/main.go",
			},
			ContentMatches: nil,
		},
	}
	got := CodeSearchResultsString(results)
	if got != "No code matches found.\n" {
		t.Errorf("no-content-matches render = %q, want sentinel", got)
	}
}

func TestCodeSearchResultsString_NameFallback(t *testing.T) {
	results := []bitbucket.CodeSearchResult{
		{
			File: bitbucket.CodeSearchFile{
				Path: "pkg/util.go",
				Commit: &bitbucket.CodeSearchCommit{
					Repository: &bitbucket.CodeSearchRepoRef{
						FullName: "",
						Name:     "repo",
					},
				},
			},
			ContentMatches: []bitbucket.CodeSearchContentMatch{
				{Lines: []bitbucket.CodeSearchLine{
					{Line: 5, Segments: []bitbucket.CodeSearchSegment{
						{Text: "func helper() {}"},
					}},
				}},
			},
		},
	}
	got := CodeSearchResultsString(results)
	if !strings.Contains(got, "repo/pkg/util.go:5:") {
		t.Errorf("name-fallback: missing expected location in:\n%s", got)
	}
}
