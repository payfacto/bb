package cmd

import (
	"fmt"
	"os"
)

// resolveTextBody picks the effective text body from a pair of mutually
// exclusive flags: an inline string (e.g. --description "...") and a file
// path (e.g. --description-file /tmp/body.md). The file form exists so AI
// agents and CI scripts can pass multi-paragraph markdown without wrestling
// with shell quoting, heredocs, or stdin-as-JSON contracts.
//
// Precedence rules:
//   - both set → validation_failed (ambiguous; refuse rather than guess)
//   - file set → read the file and return its contents
//   - inline set → return it verbatim
//   - neither set → return ""
//
// flagName / fileFlagName are used in error messages so callers don't have
// to repeat the canonical flag spelling.
func resolveTextBody(inline, filePath, flagName, fileFlagName string) (string, error) {
	if inline != "" && filePath != "" {
		return "", newCLIError(ErrCodeValidationFailed,
			fmt.Sprintf("--%s and --%s are mutually exclusive", flagName, fileFlagName), nil)
	}
	if filePath == "" {
		return inline, nil
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return "", newCLIError(ErrCodeValidationFailed,
			fmt.Sprintf("--%s: %s", fileFlagName, err.Error()), err)
	}
	return string(b), nil
}
