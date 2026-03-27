package model

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/config"
)

func TestLLMProviderCrypto_RoundTrip(t *testing.T) {
	original := config.CFG.Security.EncryptionKey
	config.CFG.Security.EncryptionKey = "llm-provider-test-key"
	t.Cleanup(func() { config.CFG.Security.EncryptionKey = original })

	plain := "sk-test-secret"
	cipher, err := encryptLLMProviderAPIKey(plain)
	if err != nil {
		t.Fatalf("encrypt api key: %v", err)
	}
	if cipher == plain {
		t.Fatal("expected encrypted api key to differ from plaintext")
	}

	got, err := decryptLLMProviderAPIKey(cipher)
	if err != nil {
		t.Fatalf("decrypt api key: %v", err)
	}
	if got != plain {
		t.Fatalf("expected round-trip plaintext %q, got %q", plain, got)
	}
}

func TestLLMProviderCrypto_RequiresEncryptionKey(t *testing.T) {
	original := config.CFG.Security.EncryptionKey
	config.CFG.Security.EncryptionKey = ""
	t.Cleanup(func() { config.CFG.Security.EncryptionKey = original })

	if _, err := encryptLLMProviderAPIKey("sk-test-secret"); err == nil {
		t.Fatal("expected encrypt to fail without encryption key")
	}
	if _, err := decryptLLMProviderAPIKey("cipher"); err == nil {
		t.Fatal("expected decrypt to fail without encryption key")
	}
}
