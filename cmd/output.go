package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	gcf "github.com/blackwell-systems/gcf-go"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/payfacto/bb/internal/config"
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

// renderError writes a CLIError to stderr in the active format. The JSON path
// preserves the exact historical envelope so JSON consumers see no change.
func renderError(e *CLIError) {
	switch format {
	case "text":
		fmt.Fprintf(os.Stderr, "error: %s: %s\n", e.Code, e.Message)
	case "gcf":
		fmt.Fprint(os.Stderr, gcf.EncodeGeneric(e))
	default: // json
		emitErrorJSON(e)
	}
}

// resolveFormat reads the effective format from cobra flags, BB_FORMAT, and
// cfg, applies the non-TTY guard, and stores the result in the package-level
// `format` var used by renderValue/renderError. Call once in PersistentPreRunE
// AFTER config is loaded.
func resolveFormat(cmd *cobra.Command, cfg *config.Config) error {
	resolved, err := resolveFormatFrom(formatInputs{
		cfgFormat:   cfg.Format,
		envFormat:   os.Getenv("BB_FORMAT"),
		flagFormat:  format,
		flagChanged: cmd.Flags().Changed("format"),
		isTTY:       term.IsTerminal(int(os.Stdout.Fd())),
	})
	if err != nil {
		return err
	}
	format = resolved
	return nil
}

// renderValue writes v to stdout in the active format. For "text" it delegates
// to textFn (the per-command human renderer backed by cmd/render).
func renderValue(v any, textFn func()) error {
	switch format {
	case "text":
		textFn()
		return nil
	case "gcf":
		fmt.Fprint(os.Stdout, gcf.EncodeGeneric(v))
		return nil
	default: // json
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
}
