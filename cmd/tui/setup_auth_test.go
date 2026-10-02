package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	keyring "github.com/zalando/go-keyring"

	"github.com/payfacto/bb/internal/auth"
	"github.com/payfacto/bb/internal/config"
)

func init() {
	// In-memory keyring: wizard tests must never touch the OS keyring.
	keyring.MockInit()
}

func stubRecommendation(t *testing.T, rec auth.Recommendation) {
	t.Helper()
	prev := detectAuthRecommendation
	detectAuthRecommendation = func() auth.Recommendation { return rec }
	t.Cleanup(func() { detectAuthRecommendation = prev })
}

func newSetupForTest(t *testing.T, rec auth.Recommendation, existing *config.Config) *setupModel {
	t.Helper()
	stubRecommendation(t, rec)
	if existing == nil {
		existing = &config.Config{}
	}
	return newSetupView(filepath.Join(t.TempDir(), "cfg.yaml"), existing)
}

func TestSetupDefaultsToOAuthWhenRecommended(t *testing.T) {
	m := newSetupForTest(t, auth.Recommendation{Method: config.AuthTypeOAuth}, nil)
	if got := m.method(); got != config.AuthTypeOAuth {
		t.Errorf("method = %q, want oauth", got)
	}
	if m.focus != setupFieldMethod {
		t.Errorf("initial focus = %d, want the method selector (%d)", m.focus, setupFieldMethod)
	}
}

func TestSetupPreselectsAPITokenAndExplainsWhy(t *testing.T) {
	rec := auth.Recommendation{Method: config.AuthTypeAPIToken, Reason: "SSH session: no browser here"}
	m := newSetupForTest(t, rec, nil)
	if got := m.method(); got != config.AuthTypeAPIToken {
		t.Errorf("method = %q, want apitoken", got)
	}
	if view := m.View(); !strings.Contains(view, "SSH session: no browser here") {
		t.Errorf("view does not explain why API token was preselected:\n%s", view)
	}
}

func TestSetupOAuthShowsClientFieldsNotTokenFields(t *testing.T) {
	m := newSetupForTest(t, auth.Recommendation{Method: config.AuthTypeOAuth}, nil)
	order := m.navOrder()
	for _, want := range []int{setupFieldMethod, setupFieldWorkspace, setupFieldClientID, setupFieldClientSecret} {
		if !slices.Contains(order, want) {
			t.Errorf("OAuth nav order %v is missing slot %d", order, want)
		}
	}
	for _, unwanted := range []int{setupFieldUsername, setupFieldPassword} {
		if slices.Contains(order, unwanted) {
			t.Errorf("OAuth nav order %v should not include slot %d", order, unwanted)
		}
	}
	if order[0] != setupFieldMethod {
		t.Errorf("auth method must come first, order = %v", order)
	}
}

func TestSetupAPITokenShowsTokenFieldsNotClientFields(t *testing.T) {
	m := newSetupForTest(t, auth.Recommendation{Method: config.AuthTypeAPIToken}, nil)
	order := m.navOrder()
	for _, want := range []int{setupFieldUsername, setupFieldPassword} {
		if !slices.Contains(order, want) {
			t.Errorf("API token nav order %v is missing slot %d", order, want)
		}
	}
	for _, unwanted := range []int{setupFieldClientID, setupFieldClientSecret} {
		if slices.Contains(order, unwanted) {
			t.Errorf("API token nav order %v should not include slot %d", order, unwanted)
		}
	}
}

func TestSetupArrowKeysSwitchMethod(t *testing.T) {
	m := newSetupForTest(t, auth.Recommendation{Method: config.AuthTypeOAuth}, nil)
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := m.method(); got != config.AuthTypeAPIToken {
		t.Errorf("after right arrow method = %q, want apitoken", got)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if got := m.method(); got != config.AuthTypeOAuth {
		t.Errorf("after left arrow method = %q, want oauth", got)
	}
}

func TestSetupTabSkipsHiddenFields(t *testing.T) {
	m := newSetupForTest(t, auth.Recommendation{Method: config.AuthTypeOAuth}, nil)
	seen := map[int]bool{}
	for i := 0; i < len(m.navOrder()); i++ {
		seen[m.focus] = true
		m.nextField()
	}
	if seen[setupFieldPassword] || seen[setupFieldUsername] {
		t.Errorf("tab landed on a hidden API-token field: %v", seen)
	}
	if !seen[setupFieldClientID] || !seen[setupFieldClientSecret] {
		t.Errorf("tab never reached the OAuth fields: %v", seen)
	}
}

func TestSetupPrefillsClientIDAndSecretFromConfig(t *testing.T) {
	existing := &config.Config{OAuthClientID: "preseeded-id", OAuthClientSecret: "env-secret"}
	m := newSetupForTest(t, auth.Recommendation{Method: config.AuthTypeOAuth}, existing)
	if got := m.fields[setupFieldClientID].Value(); got != "preseeded-id" {
		t.Errorf("client ID field = %q, want preseeded-id", got)
	}
	if got := m.fields[setupFieldClientSecret].Value(); got != "env-secret" {
		t.Errorf("client secret field = %q, want env-secret", got)
	}
}

func stubOAuthLogin(t *testing.T) {
	t.Helper()
	prevLogin, prevUser := oauthLogin, fetchOAuthUsername
	oauthLogin = func(clientID, secret string, port int) (*auth.Token, error) {
		return &auth.Token{AccessToken: "access-tok", RefreshToken: "refresh-tok"}, nil
	}
	fetchOAuthUsername = func(bearer string) (string, error) { return "jay", nil }
	t.Cleanup(func() { oauthLogin, fetchOAuthUsername = prevLogin, prevUser })
}

func TestSetupOAuthSaveLogsInStoresCredentialsAndWritesConfig(t *testing.T) {
	stubOAuthLogin(t)
	m := newSetupForTest(t, auth.Recommendation{Method: config.AuthTypeOAuth}, nil)
	m.fields[setupFieldWorkspace].SetValue("ws")
	m.fields[setupFieldClientID].SetValue("cid")
	m.fields[setupFieldClientSecret].SetValue("csecret")

	msg, ok := m.save()().(saveResultMsg)
	if !ok {
		t.Fatal("save did not return a saveResultMsg")
	}
	if msg.err != nil {
		t.Fatalf("save error: %v", msg.err)
	}
	if msg.client == nil || msg.cfg == nil {
		t.Fatal("save returned no client/config")
	}
	if msg.cfg.AuthType != config.AuthTypeOAuth || msg.cfg.Username != "jay" {
		t.Errorf("cfg = %+v, want oauth auth for user jay", msg.cfg)
	}

	if got, _ := auth.GetToken("jay"); got != "access-tok" {
		t.Errorf("stored access token = %q, want access-tok", got)
	}
	if got, _ := auth.GetClientSecret("jay"); got != "csecret" {
		t.Errorf("stored client secret = %q, want csecret", got)
	}

	data, err := os.ReadFile(m.cfgPath)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	saved := string(data)
	if !strings.Contains(saved, "oauth_client_id: cid") || !strings.Contains(saved, "auth_type: oauth") {
		t.Errorf("saved config missing oauth settings:\n%s", saved)
	}
	if strings.Contains(saved, "csecret") || strings.Contains(saved, "access-tok") {
		t.Errorf("saved config leaked a credential:\n%s", saved)
	}
}

func TestSetupOAuthSaveRequiresClientIDAndSecret(t *testing.T) {
	stubOAuthLogin(t)
	m := newSetupForTest(t, auth.Recommendation{Method: config.AuthTypeOAuth}, nil)
	m.fields[setupFieldWorkspace].SetValue("ws")

	msg := m.save()().(saveResultMsg)
	if msg.err == nil {
		t.Fatal("expected an error when client ID and secret are empty")
	}
}

func TestSetupOAuthLoginFailureSurfacesError(t *testing.T) {
	stubOAuthLogin(t)
	oauthLogin = func(string, string, int) (*auth.Token, error) { return nil, os.ErrDeadlineExceeded }
	m := newSetupForTest(t, auth.Recommendation{Method: config.AuthTypeOAuth}, nil)
	m.fields[setupFieldWorkspace].SetValue("ws")
	m.fields[setupFieldClientID].SetValue("cid")
	m.fields[setupFieldClientSecret].SetValue("csecret")

	if msg := m.save()().(saveResultMsg); msg.err == nil {
		t.Fatal("expected the login failure to be returned")
	}
}

func TestSetupViewMentionsCLIAlternatives(t *testing.T) {
	m := newSetupForTest(t, auth.Recommendation{Method: config.AuthTypeOAuth}, nil)
	view := m.View()
	for _, want := range []string{"bb auth login", "bb setup"} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not mention %q:\n%s", want, view)
		}
	}
}
