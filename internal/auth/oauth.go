package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/browser"
)

const (
	bitbucketAuthURL  = "https://bitbucket.org/site/oauth2/authorize"
	bitbucketTokenURL = "https://bitbucket.org/site/oauth2/access_token"
)

// Token holds the OAuth tokens returned by Bitbucket.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// GenerateState generates a cryptographically random 16-byte hex state string for CSRF protection.
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// BuildAuthURL constructs the Bitbucket OAuth 2.0 authorization URL.
func BuildAuthURL(clientID, state, redirectURI string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("response_type", "code")
	params.Set("state", state)
	params.Set("redirect_uri", redirectURI)
	return bitbucketAuthURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for OAuth tokens.
// tokenEndpoint is normally bitbucketTokenURL but can be overridden in tests.
func ExchangeCode(tokenEndpoint, clientID, clientSecret, code, redirectURI string) (*Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest(http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token endpoint HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tok Token
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	return &tok, nil
}

// Login runs the full OAuth 2.0 Authorization Code flow:
//  1. Starts a local callback server on a random port
//  2. Builds the authorization URL and opens it in the browser
//  3. Waits for the callback with the authorization code
//  4. Exchanges the code for tokens
func Login(clientID, clientSecret string) (*Token, error) {
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("oauth login: clientID and clientSecret must not be empty")
	}

	state, err := GenerateState()
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			errCh <- fmt.Errorf("oauth callback: state mismatch (possible CSRF)")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errParam := q.Get("error"); errParam != "" {
			errCh <- fmt.Errorf("oauth callback: provider error %q: %s", errParam, q.Get("error_description"))
			http.Error(w, "Authentication failed", http.StatusBadRequest)
			return
		}
		code := q.Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(w, "<html><body><h2>Authentication successful! You may close this tab.</h2></body></html>")
		codeCh <- code
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer func() {
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutCancel()
		_ = srv.Shutdown(shutCtx)
	}()

	authURL := BuildAuthURL(clientID, state, redirectURI)
	fmt.Printf("Opening your browser to:\n  %s\n\n", authURL)
	fmt.Println("Waiting for authentication... (press Ctrl+C to cancel)")

	if err := browser.OpenURL(authURL); err != nil {
		fmt.Println("Could not open browser automatically. Please visit the URL above manually.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	select {
	case code := <-codeCh:
		return ExchangeCode(bitbucketTokenURL, clientID, clientSecret, code, redirectURI)
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("authentication timed out after 5 minutes")
	}
}
