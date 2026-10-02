package auth

import (
	"errors"
	"os"
	"runtime"

	"github.com/payfacto/bb/internal/config"
)

// Env describes the machine facts that decide whether the browser-based OAuth
// flow can work here.
type Env struct {
	// HasDisplay is true when a local browser can plausibly be opened.
	HasDisplay bool
	// InSSH is true inside an SSH session, where the loopback OAuth callback
	// lands on the remote host rather than the user's browser.
	InSSH bool
	// KeyringOK is true when the OS keyring is usable. OAuth needs it to keep
	// the refresh token and consumer secret for unattended token refresh.
	KeyringOK bool
}

// Recommendation is the auth method to preselect and, when it is not OAuth,
// why OAuth was not chosen.
type Recommendation struct {
	Method string
	Reason string
}

// Recommend picks the auth method to preselect. OAuth is always the first
// choice; it is only passed over when this machine cannot complete it.
func Recommend(e Env) Recommendation {
	switch {
	case e.InSSH:
		return Recommendation{config.AuthTypeAPIToken, "SSH session: the OAuth browser callback cannot reach this host"}
	case !e.HasDisplay:
		return Recommendation{config.AuthTypeAPIToken, "no display available to open the OAuth browser login"}
	case !e.KeyringOK:
		return Recommendation{config.AuthTypeAPIToken, "OS keyring unavailable: OAuth needs it to store refresh credentials"}
	}
	return Recommendation{Method: config.AuthTypeOAuth}
}

// DetectEnv inspects the current process environment.
func DetectEnv() Env {
	return Env{
		HasDisplay: hasDisplay(),
		InSSH:      os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "",
		KeyringOK:  keyringAvailable(),
	}
}

func hasDisplay() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// keyringAvailable probes the keyring with a lookup that is expected to miss.
// "Not found" still means the keyring answered; only ErrNoKeyring means it is
// unusable.
func keyringAvailable() bool {
	_, err := get(keyringSvc, "bb-keyring-probe")
	return !errors.Is(err, ErrNoKeyring)
}
