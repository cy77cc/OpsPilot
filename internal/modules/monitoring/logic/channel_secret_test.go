package logic

import (
	"strings"
	"testing"
)

func TestEncryptAndMaskChannelConfig_RoundTrip(t *testing.T) {
	key := "monitoring-secret-key"
	plain := `{"webhook":"https://x.example/hook/abc123","to":["ops@example.com"]}`
	cipherText, err := encryptChannelConfig(plain, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if cipherText == plain {
		t.Fatalf("expected ciphertext to differ from plaintext")
	}
	masked, err := decryptAndMaskChannelConfig(cipherText, key)
	if err != nil {
		t.Fatalf("mask: %v", err)
	}
	if !strings.Contains(masked, "***") {
		t.Fatalf("expected masked output, got %s", masked)
	}
}
