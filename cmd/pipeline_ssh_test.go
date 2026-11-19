package cmd

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/payfacto/bb/internal/sshhost"
)

const testKeyLine = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"
const testKeyBase64 = "AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"

func TestResolveKnownHostKey_ExplicitKeySkipsTheNetwork(t *testing.T) {
	fetch := func(context.Context, string, string) (sshhost.HostKey, error) {
		t.Fatal("the host must not be contacted when --public-key is given")
		return sshhost.HostKey{}, nil
	}
	got, err := resolveKnownHostKey(knownHostAddOptions{hostname: "github.com", port: 22, publicKey: testKeyLine}, fetch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Type != "ssh-ed25519" || got.Base64 != testKeyBase64 {
		t.Errorf("unexpected key: %+v", got)
	}
}

func TestResolveKnownHostKey_FetchesFromHostAndPort(t *testing.T) {
	var gotAddr, gotType string
	fetch := func(_ context.Context, addr, keyType string) (sshhost.HostKey, error) {
		gotAddr, gotType = addr, keyType
		return sshhost.HostKey{Type: "ssh-rsa", Base64: "AAAA"}, nil
	}
	got, err := resolveKnownHostKey(knownHostAddOptions{hostname: "git.example.com", port: 7999, keyType: "ssh-rsa"}, fetch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAddr != "git.example.com:7999" || gotType != "ssh-rsa" {
		t.Errorf("fetch called with addr=%q keyType=%q", gotAddr, gotType)
	}
	if got.Base64 != "AAAA" {
		t.Errorf("unexpected key: %+v", got)
	}
}

func TestResolveKnownHostKey_Errors(t *testing.T) {
	boom := errors.New("boom")
	failing := func(context.Context, string, string) (sshhost.HostKey, error) { return sshhost.HostKey{}, boom }

	if _, err := resolveKnownHostKey(knownHostAddOptions{hostname: "h", port: 22}, failing); !errors.Is(err, boom) {
		t.Errorf("fetch failure: got %v", err)
	}
	if _, err := resolveKnownHostKey(knownHostAddOptions{hostname: "h", port: 22, publicKey: "garbage"}, failing); err == nil {
		t.Error("invalid --public-key: want an error")
	}
	_, err := resolveKnownHostKey(knownHostAddOptions{hostname: "h", port: 22, publicKey: testKeyLine, keyType: "ssh-rsa"}, failing)
	var cliErr *CLIError
	if !errors.As(err, &cliErr) || cliErr.Code != ErrCodeValidationFailed {
		t.Errorf("--public-key with a different --key-type: got %v", err)
	}
}

func TestReadKeyPairFiles_MissingFileNamesTheFlag(t *testing.T) {
	_, _, err := readKeyPairFiles(t.TempDir()+"/absent", t.TempDir()+"/absent.pub")
	var cliErr *CLIError
	if !errors.As(err, &cliErr) || cliErr.Code != ErrCodeValidationFailed {
		t.Fatalf("want validation_failed, got %v", err)
	}
	if !strings.Contains(cliErr.Message, "--private-key-file") {
		t.Errorf("message should name the flag: %q", cliErr.Message)
	}
}

func TestReadKeyPairFiles_InvalidKeyIsRejectedWithoutEchoingIt(t *testing.T) {
	dir := t.TempDir()
	privatePath, publicPath := dir+"/id", dir+"/id.pub"
	if err := os.WriteFile(privatePath, []byte("TOPSECRETMATERIAL"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicPath, []byte(testKeyLine+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := readKeyPairFiles(privatePath, publicPath)
	if err == nil {
		t.Fatal("want an error for an invalid private key")
	}
	if strings.Contains(err.Error(), "TOPSECRETMATERIAL") {
		t.Errorf("error echoes the private key: %v", err)
	}
}
