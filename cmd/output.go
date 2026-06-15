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
