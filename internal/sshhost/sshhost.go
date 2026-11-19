// Package sshhost reads SSH host keys: over the network, with a handshake
// that stops as soon as the server has presented its key, or from a
// known_hosts-style "type base64" string.
package sshhost

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// dialTimeout bounds the whole fetch when the caller's context has no
// deadline of its own.
const dialTimeout = 10 * time.Second

// HostKey is an SSH host public key in the pieces a known_hosts entry needs.
type HostKey struct {
	Type              string // e.g. ssh-ed25519
	Base64            string // the key blob, base64-encoded
	SHA256Fingerprint string // e.g. SHA256:+DiY3wvvV6TuJJhbpZisF/zLDA0zPMSvHdkr4UvCOqU
}

func newHostKey(key ssh.PublicKey) HostKey {
	return HostKey{
		Type:              key.Type(),
		Base64:            base64.StdEncoding.EncodeToString(key.Marshal()),
		SHA256Fingerprint: ssh.FingerprintSHA256(key),
	}
}

// hostKeyAlgorithms maps a key type to the signature algorithms that make a
// server present a key of that type. RSA keys sign with several algorithms.
var hostKeyAlgorithms = map[string][]string{
	ssh.KeyAlgoED25519:  {ssh.KeyAlgoED25519},
	ssh.KeyAlgoECDSA256: {ssh.KeyAlgoECDSA256},
	ssh.KeyAlgoECDSA384: {ssh.KeyAlgoECDSA384},
	ssh.KeyAlgoECDSA521: {ssh.KeyAlgoECDSA521},
	ssh.KeyAlgoRSA:      {ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA},
}

// errHostKeyCaptured aborts the handshake once the host key is in hand, so
// no authentication is ever attempted.
var errHostKeyCaptured = errors.New("host key captured")

// Fetch connects to addr ("host:port") and returns the host key the server
// presents. keyType restricts the answer to one key type (for example
// "ssh-ed25519"); empty accepts whichever type the server prefers.
//
// The key is taken on trust from the network: callers should show the
// fingerprint so it can be compared with one the host's owner publishes.
func Fetch(ctx context.Context, addr, keyType string) (HostKey, error) {
	algorithms, err := algorithmsFor(keyType)
	if err != nil {
		return HostKey{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return HostKey{}, fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline) // best effort: bounds a stalled handshake
	}

	var captured ssh.PublicKey
	cfg := &ssh.ClientConfig{
		HostKeyAlgorithms: algorithms,
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			captured = key
			return errHostKeyCaptured
		},
	}
	_, _, _, handshakeErr := ssh.NewClientConn(conn, addr, cfg)
	if captured == nil {
		return HostKey{}, fmt.Errorf("read SSH host key of %s: %w", addr, handshakeErr)
	}
	return newHostKey(captured), nil
}

func algorithmsFor(keyType string) ([]string, error) {
	if keyType == "" {
		return nil, nil // library default: the server's preferred key
	}
	algorithms, ok := hostKeyAlgorithms[keyType]
	if !ok {
		return nil, fmt.Errorf("unsupported key type %q (use ssh-ed25519, ecdsa-sha2-nistp256, ecdsa-sha2-nistp384, ecdsa-sha2-nistp521, or ssh-rsa)", keyType)
	}
	return algorithms, nil
}

// ParseAuthorizedKey reads a public key written as "type base64 [comment]",
// the form of a .pub file and of the key part of a known_hosts line.
func ParseAuthorizedKey(line string) (HostKey, error) {
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(line)))
	if err != nil {
		return HostKey{}, fmt.Errorf("parse public key (want \"type base64\", e.g. \"ssh-ed25519 AAAA...\"): %w", err)
	}
	return newHostKey(key), nil
}

// CheckKeyPair verifies that privatePEM is an unencrypted SSH private key and
// that publicLine ("type base64 [comment]") is its public half. Its errors
// never include key material.
func CheckKeyPair(privatePEM []byte, publicLine string) error {
	signer, err := ssh.ParsePrivateKey(privatePEM)
	if err != nil {
		var passphraseMissing *ssh.PassphraseMissingError
		if errors.As(err, &passphraseMissing) {
			return errors.New("the private key is passphrase-protected; Pipelines needs an unencrypted key")
		}
		// The parser's own message is dropped: it can echo input bytes.
		return errors.New("the private key file is not a valid SSH private key (PEM or OpenSSH format)")
	}
	public, _, _, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(publicLine)))
	if err != nil {
		return errors.New("the public key file is not a valid SSH public key (want \"type base64\", e.g. \"ssh-ed25519 AAAA...\")")
	}
	if !bytes.Equal(signer.PublicKey().Marshal(), public.Marshal()) {
		return errors.New("the private and public key files do not belong together")
	}
	return nil
}
