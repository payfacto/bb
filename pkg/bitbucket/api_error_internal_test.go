package bitbucket

import (
	"errors"
	"testing"
)

func TestParseHTTPError_IncludesDetailAndValidationArguments(t *testing.T) {
	body := `{"error": {"message": "Bad request",
		"detail": "The request body contains invalid properties",
		"data": {"key": "variable-service.request.validation-error",
			"arguments": {"hostname-not-allowed": "SSH for this hostname is already configured by Bitbucket"}}}}`

	var apiErr *APIError
	if err := parseHTTPError(400, []byte(body)); !errors.As(err, &apiErr) {
		t.Fatalf("want an APIError, got %v", err)
	}
	want := "Bad request: The request body contains invalid properties (hostname-not-allowed: SSH for this hostname is already configured by Bitbucket)"
	if apiErr.Message != want {
		t.Errorf("message:\n got %q\nwant %q", apiErr.Message, want)
	}
}

func TestParseHTTPError_MessageOnlyIsUnchanged(t *testing.T) {
	var apiErr *APIError
	if err := parseHTTPError(404, []byte(`{"error": {"message": "Not found"}}`)); !errors.As(err, &apiErr) {
		t.Fatalf("want an APIError, got %v", err)
	}
	if apiErr.Message != "Not found" {
		t.Errorf("message: got %q", apiErr.Message)
	}
}

func TestParseHTTPError_ArgumentsAreSortedAndNonStringsIgnored(t *testing.T) {
	body := `{"error": {"message": "Bad request", "data": {"arguments": {"b": "second", "a": "first", "n": 3}}}}`
	var apiErr *APIError
	if err := parseHTTPError(400, []byte(body)); !errors.As(err, &apiErr) {
		t.Fatalf("want an APIError, got %v", err)
	}
	if want := "Bad request (a: first; b: second)"; apiErr.Message != want {
		t.Errorf("message: got %q, want %q", apiErr.Message, want)
	}
}

func TestParseHTTPError_DetailRepeatingTheMessageIsNotDuplicated(t *testing.T) {
	var apiErr *APIError
	if err := parseHTTPError(400, []byte(`{"error": {"message": "Bad request", "detail": "Bad request"}}`)); !errors.As(err, &apiErr) {
		t.Fatalf("want an APIError, got %v", err)
	}
	if apiErr.Message != "Bad request" {
		t.Errorf("message: got %q", apiErr.Message)
	}
}
