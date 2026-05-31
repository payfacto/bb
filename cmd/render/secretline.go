package render

import "strings"

// SecretLine builds the ANSI-free terminal redraw string for a single-line
// secret prompt. The rendered line is prompt+shown. prevCells is the rune width
// of the previously rendered line; when the new line is shorter, SecretLine
// pads with spaces to erase the leftover characters and then emits the same
// number of backspaces so the cursor ends at the end of the visible text.
// It returns the string to write and the new cell count to pass as prevCells
// on the next call. It emits no ANSI escape sequences (only '\r', ' ', '\b'),
// so it is safe on terminals without VT processing (e.g. legacy Windows cmd).
func SecretLine(prompt, shown string, prevCells int) (string, int) {
	line := prompt + shown
	cells := len([]rune(line))
	pad := 0
	if prevCells > cells {
		pad = prevCells - cells
	}
	out := "\r" + line + strings.Repeat(" ", pad) + strings.Repeat("\b", pad)
	return out, cells
}
