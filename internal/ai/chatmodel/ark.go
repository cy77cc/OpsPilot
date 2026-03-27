package chatmodel

import (
	"context"
	"fmt"

	arkmodel "github.com/cloudwego/eino-ext/components/model/ark"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cy77cc/OpsPilot/internal/model"
	arkruntime "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

func init() {
	Register("ark", &arkFactory{})
}

type arkFactory struct{}

func (f *arkFactory) Create(ctx context.Context, p *model.AILLMProvider, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if p == nil {
		return nil, fmt.Errorf("llm provider is nil")
	}

	temp := float32(p.Temperature)
	return arkmodel.NewChatModel(ctx, &arkmodel.ChatModelConfig{
		APIKey:      p.APIKey,
		BaseURL:     p.BaseURL,
		Model:       p.Model,
		Temperature: &temp,
		Timeout:     &opts.Timeout,
		Thinking: &arkruntime.Thinking{
			Type: arkruntime.ThinkingTypeDisabled,
		},
	})
}
