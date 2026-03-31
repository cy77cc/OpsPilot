package chatmodel

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cy77cc/OpsPilot/internal/model"
)

func init() {
	Register("openai", &openaiFactory{})
}

type openaiFactory struct{}

func (f *openaiFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if p == nil {
		return nil, fmt.Errorf("llm provider is nil")
	}

	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: p.BaseURL,
		Model:   p.Model,
		Timeout: opts.Timeout,
	})
}
