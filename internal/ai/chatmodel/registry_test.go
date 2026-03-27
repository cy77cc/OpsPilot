package chatmodel_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/config"
	domainmodel "github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRegistry_RegisterAndGetFactory(t *testing.T) {
	chatmodel.ResetRegistryForTest()

	chatmodel.Register("test_provider", &mockFactory{})

	factory, ok := chatmodel.GetFactory("test_provider")
	if !ok {
		t.Fatal("expected registered factory")
	}
	if factory == nil {
		t.Fatal("expected non-nil factory")
	}
}

func TestRegistry_GetFactory_NotFound(t *testing.T) {
	chatmodel.ResetRegistryForTest()

	factory, ok := chatmodel.GetFactory("missing")
	if ok {
		t.Fatal("did not expect factory to exist")
	}
	if factory != nil {
		t.Fatal("expected nil factory when provider not found")
	}
}

func TestGetDefaultChatModel_UsesDatabaseDefaultProvider(t *testing.T) {
	chatmodel.ResetRegistryForTest()

	cfg := config.CFG.LLM
	config.CFG.LLM.Enable = true
	t.Cleanup(func() { config.CFG.LLM = cfg })
	secKey := config.CFG.Security.EncryptionKey
	config.CFG.Security.EncryptionKey = "chatmodel-test-key"
	t.Cleanup(func() { config.CFG.Security.EncryptionKey = secKey })

	var (
		calls int
		seen  []*domainmodel.AILLMProvider
	)
	chatmodel.Register("test-default", &capturingFactory{
		fn: func(_ context.Context, provider *domainmodel.AILLMProvider, _ chatmodel.ChatModelConfig) (model.ToolCallingChatModel, error) {
			calls++
			seen = append(seen, provider)
			return nil, nil
		},
	})

	db := newChatModelTestDB(t)
	seedChatModelProvider(t, db, &domainmodel.AILLMProvider{
		ID:        11,
		Name:      "Default",
		Provider:  "test-default",
		Model:     "test-model",
		BaseURL:   "https://example.invalid/v1",
		APIKey:    mustEncryptAPIKey(t, "secret"),
		IsDefault: true,
		IsEnabled: true,
		SortOrder: 100,
	})
	seedChatModelProvider(t, db, &domainmodel.AILLMProvider{
		ID:        12,
		Name:      "Enabled",
		Provider:  "test-enabled",
		Model:     "other-model",
		BaseURL:   "https://example.invalid/v1",
		APIKey:    mustEncryptAPIKey(t, "secret-2"),
		IsEnabled: true,
		SortOrder: 10,
	})

	got, err := chatmodel.GetDefaultChatModel(context.Background(), db, chatmodel.ChatModelConfig{Timeout: 3 * time.Second})
	if err != nil {
		t.Fatalf("get default chat model: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil chat model from capturing factory")
	}
	if calls != 1 {
		t.Fatalf("expected factory to be called once, got %d", calls)
	}
	if len(seen) != 1 || seen[0] == nil {
		t.Fatalf("expected provider to be passed to factory, got %#v", seen)
	}
	if seen[0].ID != 11 {
		t.Fatalf("expected default provider id 11, got %d", seen[0].ID)
	}
	if seen[0].APIKey != "secret" {
		t.Fatalf("expected decrypted api key, got %q", seen[0].APIKey)
	}
}

func TestGetDefaultChatModel_UsesRuntimeContextDBWhenNil(t *testing.T) {
	chatmodel.ResetRegistryForTest()

	cfg := config.CFG.LLM
	config.CFG.LLM.Enable = true
	t.Cleanup(func() { config.CFG.LLM = cfg })
	secKey := config.CFG.Security.EncryptionKey
	config.CFG.Security.EncryptionKey = "chatmodel-test-key"
	t.Cleanup(func() { config.CFG.Security.EncryptionKey = secKey })

	var calls int
	chatmodel.Register("runtime-default", &capturingFactory{
		fn: func(_ context.Context, provider *domainmodel.AILLMProvider, _ chatmodel.ChatModelConfig) (model.ToolCallingChatModel, error) {
			calls++
			if provider == nil {
				t.Fatal("expected provider")
			}
			if provider.ID != 21 {
				t.Fatalf("expected runtime DB default provider id 21, got %d", provider.ID)
			}
			return nil, nil
		},
	})

	db := newChatModelTestDB(t)
	seedChatModelProvider(t, db, &domainmodel.AILLMProvider{
		ID:        21,
		Name:      "Runtime Default",
		Provider:  "runtime-default",
		Model:     "runtime-model",
		BaseURL:   "https://example.invalid/v1",
		APIKey:    mustEncryptAPIKey(t, "secret"),
		IsDefault: true,
		IsEnabled: true,
	})

	svcCtx := &svc.ServiceContext{DB: db}
	ctx := runtimectx.WithServices(context.Background(), svcCtx)

	got, err := chatmodel.GetDefaultChatModel(ctx, nil, chatmodel.ChatModelConfig{Timeout: time.Second})
	if err != nil {
		t.Fatalf("get default chat model via runtime context: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil chat model from capturing factory")
	}
	if calls != 1 {
		t.Fatalf("expected factory to be called once, got %d", calls)
	}
}

func TestGetDefaultChatModel_FallsBackToConfigModel(t *testing.T) {
	cfg := config.CFG.LLM
	config.CFG.LLM = config.LLM{
		Enable:   true,
		Provider: "ollama",
		BaseURL:  "http://127.0.0.1:11434",
		Model:    "llama3.2",
	}
	t.Cleanup(func() { config.CFG.LLM = cfg })

	got, err := chatmodel.GetDefaultChatModel(context.Background(), nil, chatmodel.ChatModelConfig{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("expected config fallback to succeed, got error: %v", err)
	}
	if got == nil {
		t.Fatal("expected config fallback to return a chat model")
	}
}

type mockFactory struct{}

func (f *mockFactory) Create(context.Context, *domainmodel.AILLMProvider, chatmodel.ChatModelConfig) (model.ToolCallingChatModel, error) {
	return nil, nil
}

type capturingFactory struct {
	fn func(context.Context, *domainmodel.AILLMProvider, chatmodel.ChatModelConfig) (model.ToolCallingChatModel, error)
}

func (f *capturingFactory) Create(ctx context.Context, provider *domainmodel.AILLMProvider, opts chatmodel.ChatModelConfig) (model.ToolCallingChatModel, error) {
	if f.fn == nil {
		return nil, nil
	}
	return f.fn(ctx, provider, opts)
}

func newChatModelTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&domainmodel.AILLMProvider{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

func seedChatModelProvider(t *testing.T, db *gorm.DB, provider *domainmodel.AILLMProvider) {
	t.Helper()
	if err := db.Create(provider).Error; err != nil {
		t.Fatalf("seed llm provider: %v", err)
	}
}

func mustEncryptAPIKey(t *testing.T, plain string) string {
	t.Helper()
	cipher, err := utils.EncryptText(plain, config.CFG.Security.EncryptionKey)
	if err != nil {
		t.Fatalf("encrypt api key: %v", err)
	}
	return cipher
}
