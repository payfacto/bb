package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)

// Stable error codes consumed by agent clients. Do not rename without
// coordinating with API consumers — these are part of bb's public contract.
const (
	ErrCodeConfigMissing    = "config_missing"
	ErrCodeAuthFailed       = "auth_failed"
	ErrCodeNotFound         = "not_found"
	ErrCodeValidationFailed = "validation_failed"
	ErrCodeConflict         = "conflict"
	ErrCodeRateLimited      = "rate_limited"
	ErrCodeAPIError         = "api_error"
	ErrCodeInternalError    = "internal_error"
)

// CLIError is the wire-format error surfaced to stderr when bb fails.
// Construct via newCLIError so the Code field stays bound to the constants
// above.
type CLIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	cause   error
}

func (e *CLIError) Error() string { return e.Message }
func (e *CLIError) Unwrap() error { return e.cause }

func newCLIError(code, message string, cause error) *CLIError {
	return &CLIError{Code: code, Message: message, cause: cause}
}

// requiredFlagPattern matches Cobra's required-flag validation error so we
// can surface it as validation_failed. Cobra formats it as:
//
//	required flag(s) "name" not set
//	required flag(s) "a", "b" not set
//
// Captured group is the comma-separated quoted list of flag names.
var requiredFlagPattern = regexp.MustCompile(`required flag\(s\) (.+) not set`)

// mapError converts any error returned from a RunE into a structured CLIError.
// Already-typed *CLIError values pass through unchanged. *bitbucket.APIError is
// mapped by HTTP status. config sentinels are detected via errors.Is. Cobra's
// required-flag errors are recognised via regex (Cobra doesn't expose a typed
// error). Everything else becomes internal_error.
func mapError(err error) *CLIError {
	if err == nil {
		return nil
	}

	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr
	}

	var apiErr *bitbucket.APIError
	if errors.As(err, &apiErr) {
		return mapAPIError(apiErr)
	}

	// Config sentinels — stable contract via errors.Is.
	switch {
	case errors.Is(err, config.ErrNoWorkspace),
		errors.Is(err, config.ErrNoRepo),
		errors.Is(err, config.ErrNoUsername):
		return &CLIError{Code: ErrCodeConfigMissing, Message: err.Error(), cause: err}
	case errors.Is(err, config.ErrNoCredentials):
		return &CLIError{Code: ErrCodeAuthFailed, Message: err.Error(), cause: err}
	}

	// Cobra's required-flag check returns a plain *errors.errorString; pattern
	// match on the message and extract the flag names into details.
	msg := err.Error()
	if m := requiredFlagPattern.FindStringSubmatch(msg); m != nil {
		return &CLIError{
			Code:    ErrCodeValidationFailed,
			Message: msg,
			Details: map[string]any{"missing_flags": parseQuotedNames(m[1])},
			cause:   err,
		}
	}

	return &CLIError{Code: ErrCodeInternalError, Message: msg, cause: err}
}

// parseQuotedNames extracts bare names from a comma-separated list of
// double-quoted identifiers, e.g. `"a", "b"` -> ["a", "b"]. Used to surface
// the flags Cobra is complaining about.
func parseQuotedNames(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// codeForHTTPStatus maps an HTTP status code to a stable CLI error code.
// Specific 4xx statuses get dedicated codes (auth_failed, not_found, ...);
// other 4xx fall back to validation_failed; 5xx maps to api_error.
func codeForHTTPStatus(status int) string {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrCodeAuthFailed
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusConflict:
		return ErrCodeConflict
	case http.StatusUnprocessableEntity:
		return ErrCodeValidationFailed
	case http.StatusTooManyRequests:
		return ErrCodeRateLimited
	}
	if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
		return ErrCodeValidationFailed
	}
	return ErrCodeAPIError
}

func mapAPIError(apiErr *bitbucket.APIError) *CLIError {
	code := codeForHTTPStatus(apiErr.Status)
	details := map[string]any{"http_status": apiErr.Status}
	// Bitbucket 4xx responses sometimes echo request fragments (PR titles,
	// branch names, accidentally-submitted secrets). Default to redacted; opt
	// in to raw body via BB_DEBUG=1 for local troubleshooting.
	if os.Getenv("BB_DEBUG") == "1" {
		details["response_body"] = apiErr.Body
	} else {
		details["response_body_redacted"] = true
	}
	return &CLIError{
		Code:    code,
		Message: apiErr.Error(),
		Details: details,
		cause:   apiErr,
	}
}

// emitError writes a single JSON object to stderr in the shape
//
//	{"error": {"code": "...", "message": "...", "details": {...}}}
//
// followed by a newline. stdout is left untouched so successful payloads on
// stdout and error payloads on stderr never interleave.
func emitError(e *CLIError) {
	payload := map[string]any{"error": e}
	b, err := json.Marshal(payload)
	if err != nil {
		// Marshal failure should be impossible (CLIError fields are JSON-safe);
		// fall back to a hand-crafted envelope so callers never see plain text.
		fmt.Fprintf(os.Stderr, `{"error":{"code":%q,"message":%q}}`+"\n",
			ErrCodeInternalError, e.Message)
		return
	}
	fmt.Fprintln(os.Stderr, string(b))
}
