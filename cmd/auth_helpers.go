package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

func newBearerRequest(method, url, bearerToken string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Accept", "application/json")
	return req, nil
}

func decodeJSON(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

// maskToken shows first 4 and last 4 characters with asterisks in between.
func maskToken(tok string) string {
	if len(tok) <= 8 {
		return "********"
	}
	stars := make([]byte, len(tok)-8)
	for i := range stars {
		stars[i] = '*'
	}
	return tok[:4] + string(stars) + tok[len(tok)-4:]
}

// fetchUsername calls GET /2.0/user with a Bearer token and returns the account's username.
func fetchUsername(bearerToken string) (string, error) {
	req, err := newBearerRequest("GET", "https://api.bitbucket.org/2.0/user", bearerToken)
	if err != nil {
		return "", err
	}
	resp, err := defaultHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		Username string `json:"username"`
	}
	if err := decodeJSON(resp.Body, &result); err != nil {
		return "", err
	}
	return result.Username, nil
}
