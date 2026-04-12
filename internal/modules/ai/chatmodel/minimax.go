package chatmodel

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/claude"
	einomodel "github.com/cloudwego/eino/components/model"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai"
)

func init() {
	Register("minimax", &minimaxFactory{})
}

type minimaxFactory struct{}

func (f *minimaxFactory) Create(ctx context.Context, p *ai.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if p == nil {
		return nil, fmt.Errorf("llm provider is nil")
	}

	return claude.NewChatModel(ctx, &claude.Config{
		BaseURL: &p.BaseURL,
		Model:   p.Model,
		APIKey:  p.APIKey,
		Thinking: &claude.Thinking{
			Enable: false,
		},
	})
}
