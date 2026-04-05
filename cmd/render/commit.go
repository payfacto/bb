package render

import (
	"fmt"
	"strings"

	"github.com/payfacto/bb/pkg/bitbucket"
)

// CommitListString returns the formatted text for a list of commits.
func CommitListString(commits []bitbucket.Commit) string {
	if len(commits) == 0 {
		return "No commits found.\n"
	}

	header := fmt.Sprintf("  %s  %s  %s  %s\n",
		LabelStyle.Render(fmt.Sprintf("%-8s", "HASH")),
		LabelStyle.Render(fmt.Sprintf("%-10s", "DATE")),
		LabelStyle.Render(fmt.Sprintf("%-30s", "AUTHOR")),
		LabelStyle.Render("MESSAGE"))
	divider := fmt.Sprintf("  %s  %s  %s  %s\n",
		SepStyle.Render(strings.Repeat("─", 8)),
		SepStyle.Render(strings.Repeat("─", 10)),
		SepStyle.Render(strings.Repeat("─", 30)),
		SepStyle.Render(strings.Repeat("─", 40)))

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(divider)

	for _, c := range commits {
		hash := IDStyle.Render(c.Hash[:shortHashLen])
		date := ""
		if len(c.Date) >= datePrefixLen {
			date = c.Date[:datePrefixLen]
		}
		msg := c.Message
		if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
			msg = msg[:idx]
		}
		sb.WriteString(fmt.Sprintf("  %s  %s  %-30s  %s\n",
			hash, DimStyle.Render(date), truncate(c.Author.Raw, 30), truncate(msg, 72)))
	}

	return sb.String()
}

// CommitList prints the formatted commit list to stdout.
func CommitList(commits []bitbucket.Commit) { fmt.Print(CommitListString(commits)) }

// CommitDetailString returns the formatted text for a single commit.
func CommitDetailString(c bitbucket.Commit) string {
	const labelW = 8

	label := func(s string) string {
		return LabelStyle.Render(fmt.Sprintf("  %-*s", labelW, s))
	}

	parents := make([]string, len(c.Parents))
	for i, p := range c.Parents {
		if len(p.Hash) >= shortHashLen {
			parents[i] = p.Hash[:shortHashLen]
		} else {
			parents[i] = p.Hash
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Hash"), IDStyle.Render(c.Hash[:shortHashLen])))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Date"), DimStyle.Render(c.Date)))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Author"), c.Author.Raw))
	sb.WriteString(fmt.Sprintf("%s  %s\n", label("Message"), c.Message))
	if len(parents) > 0 {
		sb.WriteString(fmt.Sprintf("%s  %s\n", label("Parents"), strings.Join(parents, ", ")))
	}

	return sb.String()
}

// CommitDetail prints the formatted commit detail to stdout.
func CommitDetail(c bitbucket.Commit) { fmt.Print(CommitDetailString(c)) }
