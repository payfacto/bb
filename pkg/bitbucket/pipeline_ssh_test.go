package bitbucket_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/payfacto/bb/pkg/bitbucket"
)

const sshBase = "/repositories/ws/repo/pipelines_config/ssh"

func TestPipelineSSHKey_Get(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != sshBase+"/key_pair" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"type": "pipeline_ssh_key_pair", "public_key": "ssh-ed25519 AAAA"}`))
	}))
	got, err := c.PipelineSSHKey("ws", "repo").Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PublicKey != "ssh-ed25519 AAAA" {
		t.Errorf("public key: got %q", got.PublicKey)
	}
}

func TestPipelineSSHKey_Set_SendsBothKeysAndNeverReturnsThePrivateOne(t *testing.T) {
	var body map[string]any
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != sshBase+"/key_pair" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		// Even if the API echoed the private key, the result must not carry it.
		_, _ = w.Write([]byte(`{"private_key": "PRIVATE-MATERIAL", "public_key": "ssh-ed25519 AAAA"}`))
	}))
	got, err := c.PipelineSSHKey("ws", "repo").Set(context.Background(), "PRIVATE-MATERIAL", "ssh-ed25519 AAAA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["private_key"] != "PRIVATE-MATERIAL" || body["public_key"] != "ssh-ed25519 AAAA" {
		t.Errorf("unexpected request body keys: %v", keysOf(body))
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "PRIVATE-MATERIAL") {
		t.Errorf("result leaks the private key: %s", encoded)
	}
	if got.PublicKey != "ssh-ed25519 AAAA" {
		t.Errorf("public key: got %q", got.PublicKey)
	}
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestPipelineSSHKey_Delete(t *testing.T) {
	called := false
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete || r.URL.Path != sshBase+"/key_pair" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := c.PipelineSSHKey("ws", "repo").Delete(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("server was not called")
	}
}

func TestPipelineKnownHosts_List(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != sshBase+"/known_hosts" {
			t.Errorf("path: got %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"values": [{"uuid": "{h-1}", "hostname": "github.com",
			"public_key": {"key_type": "ssh-ed25519", "key": "AAAA", "sha256_fingerprint": "SHA256:abc"}}]}`))
	}))
	got, err := c.PipelineKnownHosts("ws", "repo").List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Hostname != "github.com" || got[0].PublicKey.KeyType != "ssh-ed25519" ||
		got[0].PublicKey.SHA256Fingerprint != "SHA256:abc" {
		t.Errorf("unexpected hosts: %+v", got)
	}
}

func TestPipelineKnownHosts_Add(t *testing.T) {
	var raw []byte
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != sshBase+"/known_hosts" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		raw, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"uuid": "{h-2}", "hostname": "github.com", "public_key": {"key_type": "ssh-ed25519", "key": "AAAA"}}`))
	}))
	got, err := c.PipelineKnownHosts("ws", "repo").Add(context.Background(), bitbucket.AddKnownHostInput{
		Hostname:  "github.com",
		PublicKey: bitbucket.KnownHostKey{KeyType: "ssh-ed25519", Key: "AAAA"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UUID != "{h-2}" {
		t.Errorf("uuid: got %q", got.UUID)
	}
	want := `{"hostname":"github.com","public_key":{"key_type":"ssh-ed25519","key":"AAAA"}}`
	if string(raw) != want {
		t.Errorf("request body:\n got %s\nwant %s", raw, want)
	}
}

func TestPipelineKnownHosts_Delete(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method: got %s", r.Method)
		}
		if want := sshBase + "/known_hosts/%7Bh-1%7D"; r.URL.EscapedPath() != want {
			t.Errorf("path: got %s, want %s", r.URL.EscapedPath(), want)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err := c.PipelineKnownHosts("ws", "repo").Delete(context.Background(), "{h-1}"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineSSHKey_Set_ErrorNeverEchoesThePrivateKey(t *testing.T) {
	const privateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmU\nQyNTUxOQAAACDjKqp5Fc65tEnRulDqKi\n-----END OPENSSH PRIVATE KEY-----\n"
	// A hostile or careless error response that echoes the key every way it
	// can: whole in detail, one line in a validation argument, and
	// JSON-escaped in an unparsed part of the body.
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		mustEncodeJSON(t, w, map[string]any{"error": map[string]any{
			"message": "Bad request",
			"detail":  "invalid key: " + privateKey,
			"data": map[string]any{
				"arguments": map[string]any{"private_key": "bad line QyNTUxOQAAACDjKqp5Fc65tEnRulDqKi here"},
				"echo":      privateKey,
			},
		}})
	}))
	_, err := c.PipelineSSHKey("ws", "repo").Set(context.Background(), privateKey, "ssh-ed25519 AAAA")

	var apiErr *bitbucket.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want an APIError, got %v", err)
	}
	for _, fragment := range []string{"b3BlbnNzaC1rZXktdjEAAAAABG5vbmU", "QyNTUxOQAAACDjKqp5Fc65tEnRulDqKi"} {
		for name, text := range map[string]string{"Message": apiErr.Message, "Body": apiErr.Body, "Error()": apiErr.Error()} {
			if strings.Contains(text, fragment) {
				t.Errorf("%s leaks private key material %q: %s", name, fragment, text)
			}
		}
	}
	if !strings.Contains(apiErr.Message, "Bad request") || !strings.Contains(apiErr.Message, "[REDACTED]") {
		t.Errorf("message should keep the reason and mark the redaction: %q", apiErr.Message)
	}
	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("status: got %d", apiErr.Status)
	}
}
