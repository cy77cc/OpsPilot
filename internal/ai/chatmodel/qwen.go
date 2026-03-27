package chatmodel

import (
	"context"
	"fmt"

	qwenmodel "github.com/cloudwego/eino-ext/components/model/qwen"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cy77cc/OpsPilot/internal/model"
)

func init() {
	Register("qwen", &qwenFactory{})
}

type qwenFactory struct{}

func (f *qwenFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if p == nil {
		return nil, fmt.Errorf("llm provider is nil")
	}

	temp := float32(p.Temperature)
	thinking := p.Thinking
	return qwenmodel.NewChatModel(ctx, &qwenmodel.ChatModelConfig{
		APIKey:         p.APIKey,
		BaseURL:        p.BaseURL,
		Model:          p.Model,
		Temperature:    &temp,
		Timeout:        opts.Timeout,
		EnableThinking: &thinking,
	})
}
