package model

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLLMProviderLogic_CreateEncryptsAPIKeyAtRest(t *testing.T) {
	setLLMProviderTestEncryptionKey(t)
	db := newLLMProviderLogicTestDB(t)
	l := NewLLMProviderLogic(db)

	view, err := l.Create(context.Background(), LLMProviderCreateRequest{
		Name:      "Qwen Plus",
		Provider:  "qwen",
		Model:     "qwen-plus",
		BaseURL:   "https://example.com/v1",
		APIKey:    "sk-test-secret",
		IsDefault: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if view == nil || !view.IsDefault {
		t.Fatalf("expected created provider to be default, got %#v", view)
	}
	if view.APIKeyMasked == "" {
		t.Fatal("expected response to include masked api key")
	}

	var stored LLMProviderRecord
	if err := db.First(&stored, view.ID).Error; err != nil {
		t.Fatalf("reload stored provider: %v", err)
	}
	if stored.APIKey == "sk-test-secret" {
		t.Fatal("expected api key to be encrypted at rest")
	}

	plain, err := decryptLLMProviderAPIKey(stored.APIKey)
	if err != nil {
		t.Fatalf("decrypt stored api key: %v", err)
	}
	if plain != "sk-test-secret" {
		t.Fatalf("expected decrypted api key to match plaintext, got %q", plain)
	}

	forUse, err := l.GetProviderForUse(context.Background(), view.ID)
	if err != nil {
		t.Fatalf("get provider for use: %v", err)
	}
	if forUse.APIKey != "sk-test-secret" {
		t.Fatalf("expected plaintext api key for use, got %q", forUse.APIKey)
	}
}

func TestLLMProviderLogic_GetMissingProviderReturnsNotFoundCode(t *testing.T) {
	setLLMProviderTestEncryptionKey(t)
	db := newLLMProviderLogicTestDB(t)
	l := NewLLMProviderLogic(db)

	_, err := l.Get(context.Background(), 404)
	if err == nil {
		t.Fatal("expected missing provider lookup to fail")
	}
	if got := xcode.FromError(err).Code; got != xcode.LLMProviderNotFound {
		t.Fatalf("expected code %d, got %d", xcode.LLMProviderNotFound, got)
	}
}

func TestLLMProviderLogic_GetDisabledProviderReturnsDisabledCode(t *testing.T) {
	setLLMProviderTestEncryptionKey(t)
	db := newLLMProviderLogicTestDB(t)
	l := NewLLMProviderLogic(db)

	rec := &LLMProviderRecord{
		Name:          "Disabled",
		Provider:      "qwen",
		Model:         "disabled",
		BaseURL:       "https://example.com/v1",
		APIKey:        mustEncryptForTest(t, "sk-disabled"),
		ConfigVersion: 1,
	}
	if err := db.Create(rec).Error; err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	if err := db.Model(&LLMProviderRecord{}).Where("id = ?", rec.ID).Update("is_enabled", false).Error; err != nil {
		t.Fatalf("disable provider: %v", err)
	}

	_, err := l.GetProviderForUse(context.Background(), rec.ID)
	if err == nil {
		t.Fatal("expected disabled provider lookup to fail")
	}
	if got := xcode.FromError(err).Code; got != xcode.LLMProviderDisabled {
		t.Fatalf("expected code %d, got %d", xcode.LLMProviderDisabled, got)
	}
}

func TestLLMProviderLogic_DeleteDefaultProviderReturnsInUseCode(t *testing.T) {
	setLLMProviderTestEncryptionKey(t)
	db := newLLMProviderLogicTestDB(t)
	l := NewLLMProviderLogic(db)

	view, err := l.Create(context.Background(), LLMProviderCreateRequest{
		Name:      "Default",
		Provider:  "qwen",
		Model:     "default",
		BaseURL:   "https://example.com/v1",
		APIKey:    "sk-default",
		IsDefault: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("create default provider: %v", err)
	}

	if err := l.Delete(context.Background(), view.ID); err == nil {
		t.Fatal("expected deleting default provider to fail")
	} else if got := xcode.FromError(err).Code; got != xcode.LLMProviderInUse {
		t.Fatalf("expected code %d, got %d", xcode.LLMProviderInUse, got)
	}
}

func TestLLMProviderLogic_PreviewImportRejectsDuplicateProviders(t *testing.T) {
	setLLMProviderTestEncryptionKey(t)
	db := newLLMProviderLogicTestDB(t)
	l := NewLLMProviderLogic(db)

	_, err := l.PreviewImport(context.Background(), LLMProviderImportRequest{
		Providers: []LLMProviderCreateRequest{
			{
				Name:     "One",
				Provider: "qwen",
				Model:    "model-a",
				BaseURL:  "https://example.com/v1",
				APIKey:   "sk-a",
			},
			{
				Name:     "Two",
				Provider: "qwen",
				Model:    "model-a",
				BaseURL:  "https://example.com/v1",
				APIKey:   "sk-b",
			},
		},
	})
	if err == nil {
		t.Fatal("expected duplicate import preview to fail")
	}
	if got := xcode.FromError(err).Code; got != xcode.LLMImportValidationFail {
		t.Fatalf("expected code %d, got %d", xcode.LLMImportValidationFail, got)
	}
}

func newLLMProviderLogicTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&LLMProviderRecord{}); err != nil {
		t.Fatalf("auto migrate llm provider record: %v", err)
	}
	return db
}

func setLLMProviderTestEncryptionKey(t *testing.T) {
	t.Helper()

	original := config.CFG.Security.EncryptionKey
	config.CFG.Security.EncryptionKey = "llm-provider-test-key"
	t.Cleanup(func() {
		config.CFG.Security.EncryptionKey = original
	})
}

func mustEncryptForTest(t *testing.T, plain string) string {
	t.Helper()

	cipher, err := encryptLLMProviderAPIKey(plain)
	if err != nil {
		t.Fatalf("encrypt api key: %v", err)
	}
	return cipher
}

func boolPtr(v bool) *bool { return &v }
