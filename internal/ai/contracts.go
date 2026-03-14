// Package ai 是 AI 编排模块的顶层包。
//
// 本文件定义包级公开类型别名和灰度配置，供外部包（handler、service）引用，
// 避免外部直接依赖 runtime 子包。
package ai

import (
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/config"
)

// 以下是 runtime 包公开类型的别名，供顶层包使用方引用。
type StreamEvent = runtime.StreamEvent
type StreamEmitter = runtime.StreamEmitter
type SelectedResource = runtime.SelectedResource
type RuntimeContext = runtime.RuntimeContext
type RunRequest = runtime.RunRequest
type ResumeRequest = runtime.ResumeRequest
type ResumeResult = runtime.ResumeResult

// RolloutConfig 控制 AI 运行时的灰度开关。
// 各字段对应 config.FeatureFlags 中的标志位，用于在不同运行模式之间切换。
type RolloutConfig struct {
	UseMultiDomainArch          bool // 是否启用多域架构模式
	UseTurnBlockStreaming        bool // 是否启用轮次块级流式输出
	UseModelFirstRuntime        bool // 是否启用 model-first 运行时（新版）
	AllowLegacySemanticFallback bool // 是否允许降级到旧语义路由
	UseAssistantV2              bool // 是否启用 Assistant V2 模式
}

// CurrentRolloutConfig 从全局配置读取当前灰度开关。
func CurrentRolloutConfig() RolloutConfig {
	return RolloutConfig{
		UseMultiDomainArch:          config.CFG.AI.UseMultiDomainArch,
		UseTurnBlockStreaming:       config.CFG.AI.UseTurnBlockStreaming,
		UseModelFirstRuntime:        boolOrDefault(config.CFG.FeatureFlags.AIModelFirstRuntime, false),
		AllowLegacySemanticFallback: boolOrDefault(config.CFG.FeatureFlags.AILegacySemanticFallback, true),
		UseAssistantV2:              boolOrDefault(config.CFG.FeatureFlags.AIAssistantV2, false),
	}
}

// RuntimeMode 返回当前生效的运行时模式标识字符串，用于可观测性打标。
func (r RolloutConfig) RuntimeMode() string {
	switch {
	case r.UseAssistantV2:
		return "aiv2"
	case r.UseModelFirstRuntime:
		return "model_first"
	case r.AllowLegacySemanticFallback || r.UseMultiDomainArch:
		return "compatibility"
	default:
		return "legacy"
	}
}

// ModelFirstEnabled 返回是否启用 model-first 运行时。
func (r RolloutConfig) ModelFirstEnabled() bool {
	return r.UseModelFirstRuntime
}

// CompatibilityEnabled 返回是否启用兼容模式（多域架构或旧语义降级）。
func (r RolloutConfig) CompatibilityEnabled() bool {
	return r.UseMultiDomainArch || r.AllowLegacySemanticFallback
}

// TurnBlockStreamingEnabled 返回是否启用轮次块级流式输出。
func (r RolloutConfig) TurnBlockStreamingEnabled() bool {
	return r.UseTurnBlockStreaming
}

func boolOrDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}
