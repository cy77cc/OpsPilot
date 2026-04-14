package utils

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
)

func TestGenToken_RejectsEmptySecret(t *testing.T) {
	originalSecret := config.CFG.JWT.Secret
	config.CFG.JWT.Secret = ""
	t.Cleanup(func() {
		config.CFG.JWT.Secret = originalSecret
	})

	_, err := GenToken(1, false)
	if err == nil {
		t.Fatal("expected error when JWT secret is empty")
	}
}

func TestParseToken_RejectsEmptySecret(t *testing.T) {
	originalSecret := config.CFG.JWT.Secret
	config.CFG.JWT.Secret = "parse-secret"
	t.Cleanup(func() {
		config.CFG.JWT.Secret = originalSecret
	})

	token, err := GenToken(1, false)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	config.CFG.JWT.Secret = ""
	_, err = ParseToken(token)
	if err == nil {
		t.Fatal("expected parse error when JWT secret is empty")
	}

	codeErr := xcode.FromError(err)
	if codeErr.Code != xcode.TokenInvalid {
		t.Fatalf("expected error code %d, got %d", xcode.TokenInvalid, codeErr.Code)
	}
}
