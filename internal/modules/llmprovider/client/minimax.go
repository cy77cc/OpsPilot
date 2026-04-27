package client

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/claude"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/model"
)

func init() {
	Register("minimax", &minimaxFactory{})
}

type minimaxFactory struct{}

func (f *minimaxFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if p == nil {
		return nil, fmt.Errorf("llm provider is nil")
	}

	// MiniMax 兼容 Claude/OpenAI 协议，通过 Claude adapter 路由。
	temp := float32(p.Temperature)
	return claude.NewChatModel(ctx, &claude.Config{
		BaseURL: &p.BaseURL,
		Model:   p.Model,
		APIKey:  p.APIKey,
		Temperature: &temp,
		Thinking: &claude.Thinking{
			Enable: p.Thinking,
		},
	})
}
