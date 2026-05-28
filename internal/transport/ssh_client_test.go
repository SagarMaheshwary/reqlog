package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"golang.org/x/crypto/ssh"
)

type fakeDialer struct {
	client *ssh.Client
	err    error
	called bool
}

func (f *fakeDialer) dial(network, addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
	f.called = true

	if f.err != nil {
		return nil, f.err
	}

	return f.client, nil
}

func writeTestPrivateKey(t *testing.T) string {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(priv)

	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "key")

	if err := os.WriteFile(path, pem.EncodeToMemory(pemBlock), 0600); err != nil {
		t.Fatalf("write key failed: %v", err)
	}

	return path
}

func TestNewSSHClient_Success(t *testing.T) {
	keyPath := writeTestPrivateKey(t)

	host := config.Host{
		Host:    "example.com",
		Port:    22,
		User:    "root",
		Key:     keyPath,
		Timeout: 5 * time.Second,
	}

	dialer := &fakeDialer{
		client: &ssh.Client{},
	}

	client, err := NewSSHClient(host, dialer.dial)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("expected client, got nil")
	}

	if !dialer.called {
		t.Fatal("expected dialer to be called")
	}
}

func TestNewSSHClient_DialError(t *testing.T) {
	keyPath := writeTestPrivateKey(t)

	host := config.Host{
		Host: "example.com",
		Port: 22,
		User: "root",
		Key:  keyPath,
	}

	dialer := &fakeDialer{
		err: fmt.Errorf("dial failed"),
	}

	_, err := NewSSHClient(host, dialer.dial)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewSSHClient_InvalidKeyFile(t *testing.T) {
	host := config.Host{
		Host: "example.com",
		Port: 22,
		User: "root",
		Key:  "/non/existent/key",
	}

	dialer := &fakeDialer{}

	_, err := NewSSHClient(host, dialer.dial)
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
}

func TestNewSSHClient_InvalidPrivateKey(t *testing.T) {
	dir := t.TempDir()
	badKey := filepath.Join(dir, "bad_key")

	if err := os.WriteFile(badKey, []byte("invalid-key"), 0600); err != nil {
		t.Fatal(err)
	}

	host := config.Host{
		Host: "example.com",
		Port: 22,
		User: "root",
		Key:  badKey,
	}

	dialer := &fakeDialer{}

	_, err := NewSSHClient(host, dialer.dial)
	if err == nil {
		t.Fatal("expected invalid key parse error")
	}
}
