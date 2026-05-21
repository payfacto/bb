package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/payfacto/bb/internal/config"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestMapError_Sentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"workspace", fmt.Errorf("%w — hint", config.ErrNoWorkspace), ErrCodeConfigMissing},
		{"repo", fmt.Errorf("%w — hint", config.ErrNoRepo), ErrCodeConfigMissing},
		{"username", fmt.Errorf("%w — hint", config.ErrNoUsername), ErrCodeConfigMissing},
		{"credentials", fmt.Errorf("%w — hint", config.ErrNoCredentials), ErrCodeAuthFailed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mapError(c.err)
			if got.Code != c.want {
				t.Errorf("Code = %q, want %q", got.Code, c.want)
			}
		})
	}
}

func TestMapError_APIError(t *testing.T) {
	cases := []struct {
		status int
		want   string
	}{
		{401, ErrCodeAuthFailed},
		{403, ErrCodeAuthFailed},
		{404, ErrCodeNotFound},
		{409, ErrCodeConflict},
		{422, ErrCodeValidationFailed},
		{429, ErrCodeRateLimited},
		{400, ErrCodeValidationFailed}, // generic 4xx
		{418, ErrCodeValidationFailed},
		{500, ErrCodeAPIError},
		{502, ErrCodeAPIError},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("status_%d", c.status), func(t *testing.T) {
			got := mapError(&bitbucket.APIError{Status: c.status, Body: "x"})
			if got.Code != c.want {
				t.Errorf("Code = %q, want %q", got.Code, c.want)
			}
			if got.Details["http_status"] != c.status {
				t.Errorf("Details.http_status = %v, want %d", got.Details["http_status"], c.status)
			}
		})
	}
}

func TestMapError_APIError_RedactionDefault(t *testing.T) {
	t.Setenv("BB_DEBUG", "")
	got := mapError(&bitbucket.APIError{Status: 404, Body: "sensitive"})
	if _, present := got.Details["response_body"]; present {
		t.Errorf("response_body should not be present without BB_DEBUG=1; details=%v", got.Details)
	}
	if got.Details["response_body_redacted"] != true {
		t.Errorf("response_body_redacted should be true; details=%v", got.Details)
	}
}

func TestMapError_APIError_RedactionDebugOptIn(t *testing.T) {
	t.Setenv("BB_DEBUG", "1")
	got := mapError(&bitbucket.APIError{Status: 404, Body: "sensitive"})
	if got.Details["response_body"] != "sensitive" {
		t.Errorf("response_body = %v, want \"sensitive\"", got.Details["response_body"])
	}
}

func TestMapError_RequiredFlag(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want []string
	}{
		{"single", `required flag(s) "pr-id" not set`, []string{"pr-id"}},
		{"two", `required flag(s) "name", "from" not set`, []string{"name", "from"}},
		{"three", `required flag(s) "a", "b", "c" not set`, []string{"a", "b", "c"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mapError(errors.New(c.msg))
			if got.Code != ErrCodeValidationFailed {
				t.Errorf("Code = %q, want %q", got.Code, ErrCodeValidationFailed)
			}
			missing, ok := got.Details["missing_flags"].([]string)
			if !ok {
				t.Fatalf("missing_flags = %T, want []string", got.Details["missing_flags"])
			}
			if !reflect.DeepEqual(missing, c.want) {
				t.Errorf("missing_flags = %v, want %v", missing, c.want)
			}
		})
	}
}

func TestMapError_PassthroughCLIError(t *testing.T) {
	original := &CLIError{Code: ErrCodeRateLimited, Message: "slow down"}
	got := mapError(original)
	if got != original {
		t.Errorf("expected pass-through of *CLIError, got new instance")
	}
}

func TestMapError_FallbackInternalError(t *testing.T) {
	got := mapError(errors.New("something inscrutable"))
	if got.Code != ErrCodeInternalError {
		t.Errorf("Code = %q, want %q", got.Code, ErrCodeInternalError)
	}
}

func TestMapError_Nil(t *testing.T) {
	if got := mapError(nil); got != nil {
		t.Errorf("mapError(nil) = %v, want nil", got)
	}
}

func TestParseQuotedNames(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`"a"`, []string{"a"}},
		{`"a", "b"`, []string{"a", "b"}},
		{`"a","b"`, []string{"a", "b"}},
		{`  "a"  ,  "b"  `, []string{"a", "b"}},
		{``, []string{}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := parseQuotedNames(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("parseQuotedNames(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
