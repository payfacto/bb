package auth

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/pkg/browser"
)

// stubBrowser replaces openURL with a stub that completes the flow with a
// provider error, so no token exchange is attempted.
func stubBrowser(t *testing.T) {
	t.Helper()
	prev := openURL
	t.Cleanup(func() { openURL = prev })
	openURL = func(u string) error {
		parsed, err := url.Parse(u)
		if err != nil {
			return err
		}
		q := parsed.Query()
		resp, err := http.Get(q.Get("redirect_uri") + "?error=access_denied&state=" + q.Get("state"))
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}
}

// freePort returns a loopback port that was free a moment ago.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	prev := os.Stdout
	os.Stdout = w
	fn()
	os.Stdout = prev
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestLoginQuietWritesNothingToStdout(t *testing.T) {
	stubBrowser(t)
	var err error
	out := captureStdout(t, func() { _, err = LoginQuiet("id", "secret", freePort(t)) })
	if err == nil || !strings.Contains(err.Error(), "access_denied") {
		t.Fatalf("expected provider error, got %v", err)
	}
	if out != "" {
		t.Errorf("LoginQuiet wrote to stdout: %q", out)
	}
}

func TestLoginPrintsProgress(t *testing.T) {
	stubBrowser(t)
	out := captureStdout(t, func() { _, _ = Login("id", "secret", freePort(t)) })
	if !strings.Contains(out, "Opening your browser") {
		t.Errorf("Login should print progress, got %q", out)
	}
}

func TestLoginQuietFailsFastWithURLWhenBrowserCannotOpen(t *testing.T) {
	prev := openURL
	t.Cleanup(func() { openURL = prev })
	openURL = func(string) error { return errors.New("no display") }

	_, err := LoginQuiet("id", "secret", freePort(t))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"no display", "https://bitbucket.org/site/oauth2/authorize", "client_id=id"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err, want)
		}
	}
}

func TestOpenSilentlyDiscardsBrowserOutputAndRestores(t *testing.T) {
	prevOut, prevErr := browser.Stdout, browser.Stderr
	var inOut, inErr io.Writer
	_ = openSilently(func(string) error {
		inOut, inErr = browser.Stdout, browser.Stderr
		return nil
	}, "http://x")
	if inOut != io.Discard || inErr != io.Discard {
		t.Errorf("browser output not discarded during open: %v %v", inOut, inErr)
	}
	if browser.Stdout != prevOut || browser.Stderr != prevErr {
		t.Error("browser writers not restored")
	}
}
