package render

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// MaybePage prints content to stdout. If stdout is a TTY and the content
// is taller than the terminal, it pipes through $PAGER (default: less -R).
func MaybePage(content string) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Print(content)
		return
	}

	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || strings.Count(content, "\n") <= height {
		fmt.Print(content)
		return
	}

	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -R"
	}

	// Split pager into command + args (e.g. "less -R" → ["less", "-R"])
	parts := strings.Fields(pager)
	cmd := exec.Command(parts[0], parts[1:]...) //nolint:gosec
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Pager failed — fall back to direct print
		fmt.Print(content)
	}
}
