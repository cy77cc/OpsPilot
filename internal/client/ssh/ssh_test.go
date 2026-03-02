package client

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

func TestBuildAuthMethods_WithEncryptedPrivateKeyAndPassphrase(t *testing.T) {
	privateKey, passphrase := generateEncryptedPrivateKeyForTest(t)

	methods, err := buildAuthMethods("", privateKey, passphrase)
	if err != nil {
		t.Fatalf("buildAuthMethods returned error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

func TestBuildAuthMethods_PrivateKeyPreferredOverPassword(t *testing.T) {
	privateKey, passphrase := generateEncryptedPrivateKeyForTest(t)

	methods, err := buildAuthMethods("pwd", privateKey, passphrase)
	if err != nil {
		t.Fatalf("buildAuthMethods returned error: %v", err)
	}
	if len(methods) != 2 {
		t.Fatalf("expected 2 auth methods, got %d", len(methods))
	}
}

// TestBuildAuthMethods_PasswordOnly tests password-only authentication.
func TestBuildAuthMethods_PasswordOnly(t *testing.T) {
	methods, err := buildAuthMethods("testpassword", "", "")
	if err != nil {
		t.Fatalf("buildAuthMethods returned error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

// TestBuildAuthMethods_NoAuth tests error when no auth methods available.
func TestBuildAuthMethods_NoAuth(t *testing.T) {
	_, err := buildAuthMethods("", "", "")
	if err == nil {
		t.Fatal("expected error when no auth methods provided")
	}
}

// TestBuildAuthMethods_PlainPrivateKey tests unencrypted private key.
func TestBuildAuthMethods_PlainPrivateKey(t *testing.T) {
	privateKey := generatePlainPrivateKeyForTest(t)

	methods, err := buildAuthMethods("", privateKey, "")
	if err != nil {
		t.Fatalf("buildAuthMethods returned error: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 auth method, got %d", len(methods))
	}
}

// TestBuildAuthMethods_WrongPassphrase tests wrong passphrase error.
func TestBuildAuthMethods_WrongPassphrase(t *testing.T) {
	privateKey, _ := generateEncryptedPrivateKeyForTest(t)

	_, err := buildAuthMethods("", privateKey, "wrong-passphrase")
	if err == nil {
		t.Fatal("expected error for wrong passphrase")
	}
}

func generateEncryptedPrivateKeyForTest(t *testing.T) (string, string) {
	t.Helper()

	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key failed: %v", err)
	}
	passphrase := "test-passphrase"
	block, err := gossh.MarshalPrivateKeyWithPassphrase(raw, "", []byte(passphrase))
	if err != nil {
		t.Fatalf("marshal encrypted private key failed: %v", err)
	}
	return string(pem.EncodeToMemory(block)), passphrase
}

func generatePlainPrivateKeyForTest(t *testing.T) string {
	t.Helper()

	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key failed: %v", err)
	}
	block, err := gossh.MarshalPrivateKey(raw, "")
	if err != nil {
		t.Fatalf("marshal private key failed: %v", err)
	}
	return string(pem.EncodeToMemory(block))
}
