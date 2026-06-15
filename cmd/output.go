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
const formatDefault = config.FormatGCF

// validateFormat returns a validation_failed CLIError if f is not one of
// config.OutputFormats (the canonical valid-format list).
func validateFormat(f string) error {
	for _, a := range config.OutputFormats {
		if f == a {
			return nil
		}
	}
	return newCLIError(ErrCodeValidationFailed,
		fmt.Sprintf("invalid format %q (allowed: %s)", f, strings.Join(config.OutputFormats, ", ")),
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
	if !in.isTTY && f == config.FormatText && !in.flagChanged {
		f = config.FormatGCF
	}
	return f, nil
}

// renderError writes a CLIError to stderr in the active format. The JSON path
// preserves the exact historical envelope so JSON consumers see no change.
func renderError(e *CLIError) {
	switch format {
	case config.FormatText:
		fmt.Fprintf(os.Stderr, "error: %s: %s\n", e.Code, e.Message)
	case config.FormatGCF:
		fmt.Fprint(os.Stderr, gcf.EncodeGeneric(gcfErrorView(e)))
	default: // json
		emitErrorJSON(e)
	}
}

// gcfErrorView builds the value encoded for gcf error output. It deliberately
// includes only Code, Message, and (when present) Details — never the struct
// itself. Encoding the *CLIError directly would (a) emit a spurious "## Details"
// section even when Details is nil, and (b) risk gcf collapsing the value via
// its Error() method. Using an explicit map keeps the output clean and ensures
// the unexported cause field can never leak.
func gcfErrorView(e *CLIError) map[string]any {
	m := map[string]any{"code": e.Code, "message": e.Message}
	if len(e.Details) > 0 {
		m["details"] = e.Details
	}
	return m
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
	case config.FormatText:
		textFn()
		return nil
	case config.FormatGCF:
		fmt.Fprint(os.Stdout, gcf.EncodeGeneric(v))
		return nil
	default: // json
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
}
