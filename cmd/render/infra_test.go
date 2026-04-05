package render_test

import (
	"strings"
	"testing"

	"github.com/payfacto/bb/cmd/render"
	"github.com/payfacto/bb/pkg/bitbucket"
)

func TestDeploymentListString_empty(t *testing.T) {
	if out := render.DeploymentListString(nil); !strings.Contains(out, "No deployments found.") {
		t.Errorf("got: %q", out)
	}
}

func TestDeploymentListString_row(t *testing.T) {
	d := []bitbucket.Deployment{{
		State:       bitbucket.DeploymentState{Name: "COMPLETED", Status: &bitbucket.DeploymentStatus{Name: "SUCCESSFUL"}},
		Environment: bitbucket.DeploymentEnvRef{UUID: "{env-1}"},
		Deployable:  bitbucket.Deployable{Commit: &bitbucket.DeployableCommit{Hash: "abc123def456"}},
		LastUpdateTime: "2026-04-01T10:00:00Z",
	}}
	out := render.DeploymentListString(d)
	if !strings.Contains(out, "COMPLETED") {
		t.Errorf("expected state, got: %q", out)
	}
	if !strings.Contains(out, "abc123de") {
		t.Errorf("expected hash, got: %q", out)
	}
}

func TestEnvListString_empty(t *testing.T) {
	if out := render.EnvListString(nil); !strings.Contains(out, "No environments found.") {
		t.Errorf("got: %q", out)
	}
}

func TestEnvListString_row(t *testing.T) {
	e := []bitbucket.Environment{{
		UUID:            "{env-1}",
		Name:            "Production",
		EnvironmentType: bitbucket.EnvironmentType{Name: "Production"},
		Lock:            bitbucket.EnvironmentLock{Name: "LOCKED"},
	}}
	out := render.EnvListString(e)
	if !strings.Contains(out, "Production") {
		t.Errorf("expected name, got: %q", out)
	}
	if !strings.Contains(out, "LOCKED") {
		t.Errorf("expected lock, got: %q", out)
	}
}

func TestWebhookListString_empty(t *testing.T) {
	if out := render.WebhookListString(nil); !strings.Contains(out, "No webhooks found.") {
		t.Errorf("got: %q", out)
	}
}

func TestWebhookListString_row(t *testing.T) {
	h := []bitbucket.Webhook{{UUID: "{wh-1}", Active: true, URL: "https://example.com/hook"}}
	out := render.WebhookListString(h)
	if !strings.Contains(out, "{wh-1}") {
		t.Errorf("expected UUID, got: %q", out)
	}
	if !strings.Contains(out, "https://example.com/hook") {
		t.Errorf("expected URL, got: %q", out)
	}
}

func TestDeployKeyListString_empty(t *testing.T) {
	if out := render.DeployKeyListString(nil); !strings.Contains(out, "No deploy keys found.") {
		t.Errorf("got: %q", out)
	}
}

func TestDeployKeyListString_row(t *testing.T) {
	k := []bitbucket.DeployKey{{ID: 5, Label: "CI Key", Key: "ssh-rsa AAAA..."}}
	out := render.DeployKeyListString(k)
	if !strings.Contains(out, "5") {
		t.Errorf("expected ID, got: %q", out)
	}
	if !strings.Contains(out, "CI Key") {
		t.Errorf("expected label, got: %q", out)
	}
}

func TestDownloadListString_empty(t *testing.T) {
	if out := render.DownloadListString(nil); !strings.Contains(out, "No downloads found.") {
		t.Errorf("got: %q", out)
	}
}

func TestDownloadListString_row(t *testing.T) {
	d := []bitbucket.Download{{Name: "release-v1.0.0.tar.gz", Size: 1048576}}
	out := render.DownloadListString(d)
	if !strings.Contains(out, "release-v1.0.0") {
		t.Errorf("expected filename, got: %q", out)
	}
}

func TestRestrictionListString_empty(t *testing.T) {
	if out := render.RestrictionListString(nil); !strings.Contains(out, "No branch restrictions found.") {
		t.Errorf("got: %q", out)
	}
}

func TestRestrictionListString_row(t *testing.T) {
	v := 2
	r := []bitbucket.BranchRestriction{{ID: 3, Kind: "require_approvals_to_merge", Pattern: "main", Value: &v}}
	out := render.RestrictionListString(r)
	if !strings.Contains(out, "3") {
		t.Errorf("expected ID, got: %q", out)
	}
	if !strings.Contains(out, "require_approvals") {
		t.Errorf("expected kind, got: %q", out)
	}
}

func TestMemberListString_empty(t *testing.T) {
	if out := render.MemberListString(nil); !strings.Contains(out, "No members found.") {
		t.Errorf("got: %q", out)
	}
}

func TestMemberListString_row(t *testing.T) {
	m := []bitbucket.WorkspaceMember{{User: bitbucket.User{DisplayName: "Alice", Nickname: "alice", AccountID: "123:abc"}}}
	out := render.MemberListString(m)
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected name, got: %q", out)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("expected nickname, got: %q", out)
	}
}

func TestUserMeString_fields(t *testing.T) {
	u := bitbucket.User{DisplayName: "Alice", Nickname: "alice", AccountID: "123:abc"}
	out := render.UserMeString(u)
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected name, got: %q", out)
	}
	if !strings.Contains(out, "@alice") {
		t.Errorf("expected nickname, got: %q", out)
	}
	if !strings.Contains(out, "123:abc") {
		t.Errorf("expected account ID, got: %q", out)
	}
}
