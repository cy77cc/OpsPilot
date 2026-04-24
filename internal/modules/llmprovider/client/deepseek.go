package client

import (
	"context"
	"fmt"

	deepseekmodel "github.com/cloudwego/eino-ext/components/model/deepseek"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/model"
)

func init() {
	Register("deepseek", &deepseekFactory{})
}

type deepseekFactory struct{}

func (f *deepseekFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if p == nil {
		return nil, fmt.Errorf("llm provider is nil")
	}

	temp := float32(p.Temperature)
	return deepseekmodel.NewChatModel(ctx, &deepseekmodel.ChatModelConfig{
		APIKey:      p.APIKey,
		BaseURL:     p.BaseURL,
		Model:       p.Model,
		Temperature: temp,
		Timeout:     opts.Timeout,
	})
}
