package chatmodel

import (
	"context"
	"fmt"

	ollamamodel "github.com/cloudwego/eino-ext/components/model/ollama"
	einomodel "github.com/cloudwego/eino/components/model"
	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

func init() {
	Register("ollama", &ollamaFactory{})
}

type ollamaFactory struct{}

func (f *ollamaFactory) Create(ctx context.Context, p *ai.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if p == nil {
		return nil, fmt.Errorf("llm provider is nil")
	}

	return ollamamodel.NewChatModel(ctx, &ollamamodel.ChatModelConfig{
		BaseURL: p.BaseURL,
		Model:   p.Model,
		Timeout: opts.Timeout,
	})
}
