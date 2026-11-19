package sshhost

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// startSSHServer runs a throwaway SSH server on localhost that presents an
// ed25519 host key, and returns its address and that key.
func startSSHServer(t *testing.T) (addr string, hostKey ssh.PublicKey) {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed by the test cleanup
			}
			go func() {
				defer conn.Close()
				// The handshake is all the client needs; errors are expected
				// because the client hangs up once it has the host key.
				_, _, _, _ = ssh.NewServerConn(conn, cfg)
			}()
		}
	}()
	return listener.Addr().String(), signer.PublicKey()
}

func TestFetch_ReturnsTheKeyTheServerPresents(t *testing.T) {
	addr, want := startSSHServer(t)

	got, err := Fetch(context.Background(), addr, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Type != "ssh-ed25519" {
		t.Errorf("type: got %q, want ssh-ed25519", got.Type)
	}
	if wantKey := base64.StdEncoding.EncodeToString(want.Marshal()); got.Base64 != wantKey {
		t.Errorf("key: got %q, want %q", got.Base64, wantKey)
	}
	if got.SHA256Fingerprint != ssh.FingerprintSHA256(want) {
		t.Errorf("fingerprint: got %q, want %q", got.SHA256Fingerprint, ssh.FingerprintSHA256(want))
	}
}

func TestFetch_KeyTypeTheServerLacksIsAnError(t *testing.T) {
	addr, _ := startSSHServer(t)

	if _, err := Fetch(context.Background(), addr, "ecdsa-sha2-nistp256"); err == nil {
		t.Fatal("want an error when the server has no key of the requested type")
	}
}

func TestFetch_UnknownKeyTypeIsRejectedBeforeDialling(t *testing.T) {
	_, err := Fetch(context.Background(), "127.0.0.1:1", "ssh-dss")
	if err == nil || !strings.Contains(err.Error(), "unsupported key type") {
		t.Fatalf("want an unsupported key type error, got %v", err)
	}
}

func TestFetch_UnreachableHostFailsWithinTheContextDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close() // nothing listens here any more

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := Fetch(ctx, addr, ""); err == nil {
		t.Fatal("want an error for an unreachable host")
	}
}

func TestParseAuthorizedKey(t *testing.T) {
	_, want := startSSHServer(t)
	line := string(ssh.MarshalAuthorizedKey(want)) // "ssh-ed25519 AAAA...\n"

	for _, input := range []string{line, strings.TrimSpace(line) + " some comment", "  " + line} {
		got, err := ParseAuthorizedKey(input)
		if err != nil {
			t.Fatalf("ParseAuthorizedKey(%q): %v", input, err)
		}
		if got.Type != "ssh-ed25519" || got.Base64 != base64.StdEncoding.EncodeToString(want.Marshal()) {
			t.Errorf("ParseAuthorizedKey(%q) = %+v", input, got)
		}
	}
}

func TestParseAuthorizedKey_Invalid(t *testing.T) {
	for _, input := range []string{"", "not a key", "ssh-ed25519 !!!notbase64"} {
		if _, err := ParseAuthorizedKey(input); err == nil {
			t.Errorf("ParseAuthorizedKey(%q): want an error", input)
		}
	}
}

// newKeyPairPEM returns an unencrypted OpenSSH private key and its public
// key line.
func newKeyPairPEM(t *testing.T) (privatePEM []byte, publicLine string) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(private, "")
	if err != nil {
		t.Fatal(err)
	}
	sshPublic, err := ssh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(block), string(ssh.MarshalAuthorizedKey(sshPublic))
}

func TestCheckKeyPair(t *testing.T) {
	privatePEM, publicLine := newKeyPairPEM(t)
	_, otherPublic := newKeyPairPEM(t)

	if err := CheckKeyPair(privatePEM, publicLine); err != nil {
		t.Errorf("matching pair: unexpected error: %v", err)
	}
	if err := CheckKeyPair(privatePEM, otherPublic); err == nil || !strings.Contains(err.Error(), "do not belong together") {
		t.Errorf("mismatched pair: got %v", err)
	}
	if err := CheckKeyPair([]byte("not a key"), publicLine); err == nil {
		t.Error("garbage private key: want an error")
	}
	if err := CheckKeyPair(privatePEM, "not a key"); err == nil {
		t.Error("garbage public key: want an error")
	}
}

func TestCheckKeyPair_PassphraseProtectedKeyIsRejected(t *testing.T) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKeyWithPassphrase(private, "", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	_, publicLine := newKeyPairPEM(t)

	err = CheckKeyPair(pem.EncodeToMemory(block), publicLine)
	if err == nil || !strings.Contains(err.Error(), "passphrase") {
		t.Fatalf("want a passphrase error, got %v", err)
	}
}

func TestCheckKeyPair_ErrorsNeverQuoteThePrivateKey(t *testing.T) {
	_, publicLine := newKeyPairPEM(t)
	secret := "-----BEGIN OPENSSH PRIVATE KEY-----\nTOPSECRETMATERIAL\n-----END OPENSSH PRIVATE KEY-----\n"

	err := CheckKeyPair([]byte(secret), publicLine)
	if err == nil {
		t.Fatal("want an error for a corrupt private key")
	}
	if strings.Contains(err.Error(), "TOPSECRETMATERIAL") {
		t.Errorf("error quotes private key material: %v", err)
	}
}
