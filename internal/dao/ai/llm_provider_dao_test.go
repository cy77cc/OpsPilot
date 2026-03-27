package ai

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestLLMProviderDAO_GetDefault(t *testing.T) {
	db := newLLMProviderDAOTestDB(t)
	dao := NewLLMProviderDAO(db)
	ctx := context.Background()

	seedLLMProvider(t, db, &model.AILLMProvider{
		ID:        1,
		Name:      "Disabled Default",
		Provider:  "qwen",
		Model:     "qwen-max",
		BaseURL:   "https://example.invalid/v1",
		APIKey:    "key-1",
		IsDefault: true,
		IsEnabled: false,
		SortOrder: 100,
	})
	seedLLMProvider(t, db, &model.AILLMProvider{
		ID:        2,
		Name:      "Enabled Fallback",
		Provider:  "ark",
		Model:     "doubao-pro",
		BaseURL:   "https://example.invalid/v1",
		APIKey:    "key-2",
		IsEnabled: true,
		SortOrder: 50,
	})

	got, err := dao.GetDefault(ctx)
	if err != nil {
		t.Fatalf("get default provider: %v", err)
	}
	if got != nil {
		t.Fatalf("expected no default provider when the only default is disabled, got id=%d", got.ID)
	}
}

func TestLLMProviderDAO_GetFirstEnabled(t *testing.T) {
	db := newLLMProviderDAOTestDB(t)
	dao := NewLLMProviderDAO(db)
	ctx := context.Background()

	seedLLMProvider(t, db, &model.AILLMProvider{
		ID:        1,
		Name:      "Disabled",
		Provider:  "qwen",
		Model:     "qwen-max",
		BaseURL:   "https://example.invalid/v1",
		APIKey:    "key-1",
		IsEnabled: false,
	})
	seedLLMProvider(t, db, &model.AILLMProvider{
		ID:        2,
		Name:      "First Enabled",
		Provider:  "ark",
		Model:     "doubao-pro",
		BaseURL:   "https://example.invalid/v1",
		APIKey:    "key-2",
		IsEnabled: true,
		SortOrder: 10,
	})
	seedLLMProvider(t, db, &model.AILLMProvider{
		ID:        3,
		Name:      "Second Enabled",
		Provider:  "ollama",
		Model:     "llama3.2",
		BaseURL:   "http://127.0.0.1:11434",
		APIKey:    "key-3",
		IsEnabled: true,
		SortOrder: 20,
	})

	got, err := dao.GetFirstEnabled(ctx)
	if err != nil {
		t.Fatalf("get first enabled provider: %v", err)
	}
	if got == nil {
		t.Fatal("expected first enabled provider record")
	}
	if got.ID != 2 {
		t.Fatalf("expected lowest-id enabled provider, got id=%d", got.ID)
	}
}

func newLLMProviderDAOTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AILLMProvider{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

func seedLLMProvider(t *testing.T, db *gorm.DB, provider *model.AILLMProvider) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Millisecond)
	if err := db.Exec(
		`INSERT INTO ai_llm_providers
			(id, name, provider, model, base_url, api_key, api_key_version, temperature, thinking, is_default, is_enabled, sort_order, config_version, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		provider.ID,
		provider.Name,
		provider.Provider,
		provider.Model,
		provider.BaseURL,
		provider.APIKey,
		provider.APIKeyVersion,
		provider.Temperature,
		provider.Thinking,
		provider.IsDefault,
		provider.IsEnabled,
		provider.SortOrder,
		provider.ConfigVersion,
		now,
		now,
	).Error; err != nil {
		t.Fatalf("seed llm provider: %v", err)
	}
}
