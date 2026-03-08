package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/config"
)

// NewToolCallingChatModel creates a tool-calling model based on current config.
func NewToolCallingChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	if !config.CFG.LLM.Enable {
		return nil, nil
	}

	provider := strings.ToLower(strings.TrimSpace(config.CFG.LLM.Provider))
	switch provider {
	case "qwen":
		thinking := true
		return qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
			BaseURL:        config.CFG.LLM.BaseURL,
			Model:          config.CFG.LLM.Model,
			APIKey:         config.CFG.LLM.APIKey,
			EnableThinking: &thinking,
		})
	case "ollama":
		return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			BaseURL: config.CFG.LLM.BaseURL,
			Model:   config.CFG.LLM.Model,
			Options: &ollama.Options{
				Temperature: float32(config.CFG.LLM.Temperature),
				NumPredict:  1024,
			},
		})
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", config.CFG.LLM.Provider)
	}
}

// CheckModelHealth verifies the model can respond to a minimal probe prompt.
func CheckModelHealth(ctx context.Context, chatModel model.ToolCallingChatModel) error {
	if chatModel == nil {
		return nil
	}

	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := chatModel.Generate(healthCtx, []*schema.Message{
		schema.SystemMessage("health check"),
		schema.UserMessage("ping"),
	})
	if err != nil {
		return fmt.Errorf("model health check failed: %w", err)
	}
	return nil
}
