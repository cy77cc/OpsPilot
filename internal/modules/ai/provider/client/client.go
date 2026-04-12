package client

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	chatmodel "github.com/cy77cc/OpsPilot/internal/modules/ai/chatmodel"
	rootai "github.com/cy77cc/OpsPilot/internal/modules/ai"
	"gorm.io/gorm"
)

type ChatModelConfig = chatmodel.ChatModelConfig
type Factory = chatmodel.ModelFactory
type LLMProviderRecord = rootai.AILLMProvider

func NewChatModel(ctx context.Context, opts ChatModelConfig) (model.ToolCallingChatModel, error) {
	return chatmodel.NewChatModel(ctx, opts)
}

func GetDefaultChatModel(ctx context.Context, db *gorm.DB, opts ChatModelConfig) (model.ToolCallingChatModel, error) {
	return chatmodel.GetDefaultChatModel(ctx, db, opts)
}

func CheckModelHealth(ctx context.Context) error {
	return chatmodel.CheckModelHealth(ctx)
}

func Register(provider string, factory Factory) {
	chatmodel.Register(provider, factory)
}

func GetFactory(provider string) (Factory, bool) {
	return chatmodel.GetFactory(provider)
}

func ResetRegistryForTest() {
	chatmodel.ResetRegistryForTest()
}
