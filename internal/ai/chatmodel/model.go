// Package ai 提供 AI 模型的初始化和健康检查功能。
//
// 本文件负责根据配置创建不同类型的聊天模型，支持 Ollama 和 Qwen 两种 Provider。
// 不同阶段使用不同的模型配置以优化性能和成本。
package chatmodel

import (
	"context"
	"fmt"
	"strings"
	"time"

	arkmodel "github.com/cloudwego/eino-ext/components/model/ark"
	ollamamodel "github.com/cloudwego/eino-ext/components/model/ollama"
	qwenmodel "github.com/cloudwego/eino-ext/components/model/qwen"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type ChatModelConfig struct {
	// Timeout 模型调用的超时时间。
	Timeout time.Duration
	// Thinking 是否启用模型的思考模式。
	Thinking bool
	// Temp 模型生成文本的温度参数。
	Temp float32
}

// NewChatModel 根据配置创建聊天模型实例。
// 支持 Ollama 和 Qwen 两种 Provider。
func NewChatModel(ctx context.Context, opts ChatModelConfig) (einomodel.ToolCallingChatModel, error) {
	if !config.CFG.LLM.Enable {
		return nil, fmt.Errorf("llm disabled")
	}
	switch strings.TrimSpace(strings.ToLower(config.CFG.LLM.Provider)) {
	case "ollama":
		return ollamamodel.NewChatModel(ctx, &ollamamodel.ChatModelConfig{
			BaseURL: config.CFG.LLM.BaseURL,
			Model:   config.CFG.LLM.Model,
			Timeout: opts.Timeout,
		})
	case "qwen":
		thinking := opts.Thinking
		temp := opts.Temp
		return qwenmodel.NewChatModel(ctx, &qwenmodel.ChatModelConfig{
			APIKey:         config.CFG.LLM.APIKey,
			BaseURL:        config.CFG.LLM.BaseURL,
			Model:          config.CFG.LLM.Model,
			Temperature:    &temp,
			Timeout:        opts.Timeout,
			EnableThinking: &thinking,
		})
	case "ark":
		temp := opts.Temp
		return arkmodel.NewChatModel(ctx, &arkmodel.ChatModelConfig{
			APIKey:         config.CFG.LLM.APIKey,
			BaseURL:        config.CFG.LLM.BaseURL,
			Model:          config.CFG.LLM.Model,
			Temperature:    &temp,
			Timeout:        &opts.Timeout,
			Thinking: &model.Thinking{
				Type: model.ThinkingTypeDisabled,
			},
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
func CheckModelHealth(ctx context.Context) error {
	model, err := NewChatModel(ctx, ChatModelConfig{
		Timeout:  10 * time.Second,
		Thinking: false,
		Temp:     0,
	})
	if err != nil {
		return err
	}
	_, err = model.Generate(ctx, []*schema.Message{schema.UserMessage("ping")})
	return err
}
