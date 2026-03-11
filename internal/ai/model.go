// Package ai 提供 AI 模型的初始化和健康检查功能。
//
// 本文件负责根据配置创建不同类型的聊天模型，支持 Ollama 和 Qwen 两种 Provider。
// 不同阶段使用不同的模型配置以优化性能和成本。
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

// 模型超时配置常量。
const (
	defaultChatModelTimeout = 30 * time.Second  // 默认聊天模型超时
	rewriteChatModelTimeout = 60 * time.Second  // 改写模型超时 (较长，因为需要结构化输出)
	summaryChatModelTimeout = 45 * time.Second  // 总结模型超时
)

// StartupModelHealthResult 表示启动时模型健康检查的结果。
type StartupModelHealthResult struct {
	Name  string // 模型名称 (planner/rewrite/expert/summarizer)
	Model string // 模型标识
	Err   error  // 健康检查错误，nil 表示健康
}

// chatModelOptions 定义聊天模型的创建选项。
type chatModelOptions struct {
	timeout  time.Duration // 请求超时时间
	thinking bool          // 是否启用思考模式 (Qwen 专用)
	temp     float32       // 温度参数
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

// NewRewriteChatModel 创建用于改写阶段的聊天模型。
// 温度设为 0 以获得更稳定的结构化输出。
func NewRewriteChatModel(ctx context.Context) (einomodel.BaseChatModel, error) {
	return newChatModel(ctx, chatModelOptions{
		timeout:  rewriteChatModelTimeout,
		thinking: false,
		temp:     0,
	})
}

// NewSummarizerChatModel 创建用于总结阶段的聊天模型。
// 使用配置文件中的温度参数。
func NewSummarizerChatModel(ctx context.Context) (einomodel.BaseChatModel, error) {
	return newChatModel(ctx, chatModelOptions{
		timeout:  summaryChatModelTimeout,
		thinking: true,
		temp:     float32(config.CFG.LLM.Temperature),
	})
}

// NewAnswerChatModel 创建用于直接回答的聊天模型。
// 禁用思考模式以获得更直接的回答。
func NewAnswerChatModel(ctx context.Context) (einomodel.BaseChatModel, error) {
	return newChatModel(ctx, chatModelOptions{
		timeout:  summaryChatModelTimeout,
		thinking: false,
		temp:     float32(config.CFG.LLM.Temperature),
	})
}

// newChatModel 根据配置创建聊天模型实例。
// 支持 Ollama 和 Qwen 两种 Provider。
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

// CheckBaseModelHealth 检查基础聊天模型的健康状态。
func CheckBaseModelHealth(ctx context.Context, model einomodel.BaseChatModel) error {
	if model == nil {
		return fmt.Errorf("chat model not initialized")
	}
	_, err := model.Generate(ctx, []*schema.Message{schema.UserMessage("ping")})
	return err
}

// CheckStartupModelHealth 在启动时检查所有模型的健康状态。
// 检查 planner、rewrite、expert、summarizer 四个模型。
// 返回每个模型的健康检查结果。
func CheckStartupModelHealth(ctx context.Context) []StartupModelHealthResult {
	checks := []struct {
		name    string
		factory func(context.Context) (einomodel.BaseChatModel, error)
	}{
		{
			name: "planner",
			factory: func(ctx context.Context) (einomodel.BaseChatModel, error) {
				return NewToolCallingChatModel(ctx)
			},
		},
		{
			name:    "rewrite",
			factory: NewRewriteChatModel,
		},
		{
			name: "expert",
			factory: func(ctx context.Context) (einomodel.BaseChatModel, error) {
				return NewToolCallingChatModel(ctx)
			},
		},
		{
			name:    "summarizer",
			factory: NewSummarizerChatModel,
		},
	}

	results := make([]StartupModelHealthResult, 0, len(checks))
	for _, check := range checks {
		model, err := check.factory(ctx)
		if err != nil {
			results = append(results, StartupModelHealthResult{
				Name:  check.name,
				Model: strings.TrimSpace(config.CFG.LLM.Model),
				Err:   err,
			})
			continue
		}
		err = CheckBaseModelHealth(ctx, model)
		results = append(results, StartupModelHealthResult{
			Name:  check.name,
			Model: strings.TrimSpace(config.CFG.LLM.Model),
			Err:   err,
		})
	}
	return results
}
