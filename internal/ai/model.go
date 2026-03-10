package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	ollamamodel "github.com/cloudwego/eino-ext/components/model/ollama"
	qwenmodel "github.com/cloudwego/eino-ext/components/model/qwen"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/config"
)

const (
	defaultChatModelTimeout = 30 * time.Second
	rewriteChatModelTimeout = 60 * time.Second
	summaryChatModelTimeout = 45 * time.Second
)

type chatModelOptions struct {
	timeout  time.Duration
	thinking bool
	temp     float32
}

// NewToolCallingChatModel 创建支持工具调用的聊天模型。
// 根据配置文件中的 Provider 选择 Ollama 或 Qwen 模型。
//
// 参数:
//   - ctx: 上下文。
//
// 返回:
//   - einomodel.ToolCallingChatModel: 聊天模型实例。
//   - error: 创建错误。
func NewToolCallingChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	return newChatModel(ctx, chatModelOptions{
		timeout:  defaultChatModelTimeout,
		thinking: true,
		temp:     float32(config.CFG.LLM.Temperature),
	})
}

func NewRewriteChatModel(ctx context.Context) (einomodel.BaseChatModel, error) {
	return newChatModel(ctx, chatModelOptions{
		timeout:  rewriteChatModelTimeout,
		thinking: true,
		temp:     0,
	})
}

func NewSummarizerChatModel(ctx context.Context) (einomodel.BaseChatModel, error) {
	return newChatModel(ctx, chatModelOptions{
		timeout:  summaryChatModelTimeout,
		thinking: true,
		temp:     float32(config.CFG.LLM.Temperature),
	})
}

func newChatModel(ctx context.Context, opts chatModelOptions) (einomodel.ToolCallingChatModel, error) {
	if !config.CFG.LLM.Enable {
		return nil, fmt.Errorf("llm disabled")
	}
	switch strings.TrimSpace(strings.ToLower(config.CFG.LLM.Provider)) {
	case "ollama":
		return ollamamodel.NewChatModel(ctx, &ollamamodel.ChatModelConfig{
			BaseURL: config.CFG.LLM.BaseURL,
			Model:   config.CFG.LLM.Model,
			Timeout: opts.timeout,
		})
	case "qwen":
		thinking := opts.thinking
		temp := opts.temp
		return qwenmodel.NewChatModel(ctx, &qwenmodel.ChatModelConfig{
			APIKey:         config.CFG.LLM.APIKey,
			BaseURL:        config.CFG.LLM.BaseURL,
			Model:          config.CFG.LLM.Model,
			Temperature:    &temp,
			Timeout:        opts.timeout,
			EnableThinking: &thinking,
		})
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", config.CFG.LLM.Provider)
	}
}

// CheckModelHealth 检查模型健康状态。
// 发送简单的 ping 消息验证模型是否正常响应。
//
// 参数:
//   - ctx: 上下文。
//   - model: 聊天模型实例。
//
// 返回:
//   - error: 健康检查错误。
func CheckModelHealth(ctx context.Context, model einomodel.ToolCallingChatModel) error {
	if model == nil {
		return fmt.Errorf("chat model not initialized")
	}
	_, err := model.Generate(ctx, []*schema.Message{schema.UserMessage("ping")})
	return err
}
