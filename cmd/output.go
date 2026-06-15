package cmd

import (
	"fmt"
	"strings"
)

// formatDefault is the built-in output format when nothing is configured.
const formatDefault = "gcf"

// allowedFormats is the single source of truth for valid --format values.
// Order matters: it is the order shown in error messages and the wizard.
var allowedFormats = []string{"gcf", "json", "text"}

// validateFormat returns a validation_failed CLIError if f is not allowed.
func validateFormat(f string) error {
	for _, a := range allowedFormats {
		if f == a {
			return nil
		}
	}
	return newCLIError(ErrCodeValidationFailed,
		fmt.Sprintf("invalid format %q (allowed: %s)", f, strings.Join(allowedFormats, ", ")),
		nil)
}

// formatInputs captures everything resolveFormatFrom needs, so the precedence
// logic and non-TTY guard can be unit-tested without a real terminal or cobra.
type formatInputs struct {
	cfgFormat   string // ~/.bbcloud.yaml "format:" ("" = unset)
	envFormat   string // BB_FORMAT ("" = unset)
	flagFormat  string // --format value
	flagChanged bool   // was --format set on THIS invocation
	isTTY       bool   // stdout is a terminal
}

// resolveFormatFrom applies precedence (default < config < env < --format) and
// the non-TTY text guard: when stdout is not a terminal and the resolved format
// is the human "text" format, coerce to gcf UNLESS --format was set on this
// invocation (a per-command explicit choice is always honored).
func resolveFormatFrom(in formatInputs) (string, error) {
	f := formatDefault
	if in.cfgFormat != "" {
		f = in.cfgFormat
	}
	if in.envFormat != "" {
		f = in.envFormat
	}
	if in.flagChanged {
		f = in.flagFormat
	}
	if err := validateFormat(f); err != nil {
		return "", err
	}
	if !in.isTTY && f == "text" && !in.flagChanged {
		f = "gcf"
	}
	return f, nil
}
