package cmd

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/payfacto/bb/cmd/render"
)

// readSecretRevealing reads a secret from in (which the caller has confirmed is
// a terminal), echoing it in cleartext while the user types or pastes so they
// can confirm the value landed, then masking the whole field to '*' when they
// press Enter. It returns the entered secret without a trailing newline. On
// Ctrl-C it restores the terminal and exits with code 130. If raw mode cannot
// be enabled it falls back to a plain hidden read via term.ReadPassword.
//
// It reads one byte at a time directly from in (no buffered read-ahead), so it
// never consumes input past the terminating newline; any stdin reads the caller
// performs afterward are unaffected. It runs entirely in the calling goroutine
// — no background reader — so there is no goroutine leak or stdin race.
func readSecretRevealing(in *os.File, prompt string) string {
	fd := int(in.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback: no reveal, plain hidden read.
		fmt.Print(prompt)
		b, _ := term.ReadPassword(fd)
		fmt.Println()
		return string(b)
	}
	defer term.Restore(fd, oldState)

	out := os.Stdout
	var buf []rune    // accepted runes so far
	var pending []byte // bytes of an in-progress multi-byte UTF-8 rune
	prevCells := 0
	draw := func(shown string) {
		s, cells := render.SecretLine(prompt, shown, prevCells)
		fmt.Fprint(out, s)
		prevCells = cells
	}

	draw("") // show the prompt before any input

	readBuf := make([]byte, 1)
	for {
		n, rerr := in.Read(readBuf)
		if n == 0 || rerr != nil { // EOF/error → submit what we have
			draw(strings.Repeat("*", len(buf)))
			fmt.Fprint(out, "\r\n")
			return string(buf)
		}
		b := readBuf[0]

		// Single-byte control characters are only meaningful when we are not
		// mid-way through assembling a multi-byte rune.
		if len(pending) == 0 {
			switch b {
			case '\r', '\n': // Enter → mask and submit
				draw(strings.Repeat("*", len(buf)))
				fmt.Fprint(out, "\r\n")
				return string(buf)
			case 0x03: // Ctrl-C
				term.Restore(fd, oldState)
				fmt.Fprint(out, "\r\n")
				os.Exit(130)
			case 0x7f, 0x08: // Backspace / Delete
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
				}
				draw(string(buf))
				continue
			}
			if b < 0x20 { // ignore other control bytes
				continue
			}
		}

		pending = append(pending, b)
		if utf8.FullRune(pending) {
			r, _ := utf8.DecodeRune(pending)
			pending = pending[:0]
			if r != utf8.RuneError {
				buf = append(buf, r)
			}
			draw(string(buf)) // cleartext while typing
		}
		// otherwise: wait for the remaining bytes of this rune
	}
}
