package client

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/model"
)

func init() {
	factory := &openaiFactory{}
	Register("openai", factory)
	Register("moonshot", factory)
	Register("zhipu", factory)
	Register("google", factory)
}

type openaiFactory struct{}

func (f *openaiFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if p == nil {
		return nil, fmt.Errorf("llm provider is nil")
	}

	temp := float32(p.Temperature)
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      p.APIKey,
		BaseURL:     p.BaseURL,
		Model:       p.Model,
		Temperature: &temp,
		Timeout:     opts.Timeout,
	})
}
