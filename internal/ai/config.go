// Package ai 提供 AI 编排层的配置管理。
//
// 本文件定义功能发布 (Rollout) 相关的配置结构和决策逻辑。
// 用于控制 AI 功能的灰度发布和降级策略。
package ai

import "github.com/cy77cc/OpsPilot/internal/config"

// RolloutConfig 定义 AI 功能的发布配置。
//
// 字段说明:
//   - UseMultiDomainArch: 是否启用多领域架构 (Agentic 模式)
//   - UseAssistantV2: 是否使用 Assistant V2 版本
//   - UseModelFirstRuntime: 是否使用模型优先运行时
//   - AllowLegacySemanticFallback: 是否允许降级到旧版语义处理
type RolloutConfig struct {
	UseMultiDomainArch          bool `json:"use_multi_domain_arch"`           // 多领域架构开关
	UseAssistantV2              bool `json:"use_assistant_v2"`                // Assistant V2 开关
	UseModelFirstRuntime        bool `json:"use_model_first_runtime"`         // 模型优先运行时开关
	AllowLegacySemanticFallback bool `json:"allow_legacy_semantic_fallback"`  // 旧版语义降级开关
}

// RolloutThresholds 定义发布决策的阈值。
//
// 当错误率超过阈值时，会自动降级或阻止发布。
type RolloutThresholds struct {
	MaxPlannerErrorRate    float64 `json:"max_planner_error_rate"`    // 最大规划器错误率
	MaxResumeFailureRate   float64 `json:"max_resume_failure_rate"`   // 最大恢复失败率
	MinRewriteSuccessRate  float64 `json:"min_rewrite_success_rate"`  // 最小改写成功率
	AllowCompatibilityPath bool    `json:"allow_compatibility_path"`  // 是否允许兼容路径
}

// RolloutDecision 表示发布决策结果。
type RolloutDecision struct {
	Enabled bool   `json:"enabled"` // 是否启用
	Reason  string `json:"reason"`  // 决策原因
}

// CurrentRolloutConfig 获取当前的发布配置。
// 从全局配置和功能开关中读取配置值。
func CurrentRolloutConfig() RolloutConfig {
	cfg := RolloutConfig{
		UseMultiDomainArch:   config.CFG.AI.UseMultiDomainArch,
		UseModelFirstRuntime: true, // 默认启用模型优先运行时
	}
	// 从功能开关中读取配置 (如果设置)
	if config.CFG.FeatureFlags.AIAssistantV2 != nil {
		cfg.UseAssistantV2 = *config.CFG.FeatureFlags.AIAssistantV2
	}
	if config.CFG.FeatureFlags.AIModelFirstRuntime != nil {
		cfg.UseModelFirstRuntime = *config.CFG.FeatureFlags.AIModelFirstRuntime
	}
	if config.CFG.FeatureFlags.AILegacySemanticFallback != nil {
		cfg.AllowLegacySemanticFallback = *config.CFG.FeatureFlags.AILegacySemanticFallback
	}
	return cfg
}

// AgenticEnabled 检查是否启用了 Agentic (多领域) 架构。
func (c RolloutConfig) AgenticEnabled() bool {
	return c.UseMultiDomainArch
}

// ModelFirstEnabled 检查是否启用了模型优先运行时。
func (c RolloutConfig) ModelFirstEnabled() bool {
	return c.UseModelFirstRuntime
}

// CompatibilityEnabled 检查是否启用了兼容模式。
// 当未启用多领域架构、或启用了 V2/旧版降级时，兼容模式可用。
func (c RolloutConfig) CompatibilityEnabled() bool {
	return !c.UseMultiDomainArch || c.UseAssistantV2 || c.AllowLegacySemanticFallback
}

// RuntimeMode 返回当前运行时模式。
// 返回值: "model_first" | "compatibility" | "disabled"
func (c RolloutConfig) RuntimeMode() string {
	switch {
	case c.ModelFirstEnabled():
		return "model_first"
	case c.AllowLegacySemanticFallback:
		return "compatibility"
	default:
		return "disabled"
	}
}

// DefaultRolloutThresholds 返回默认的发布阈值。
func DefaultRolloutThresholds() RolloutThresholds {
	return RolloutThresholds{
		MaxPlannerErrorRate:    0.10, // 10% 最大规划器错误率
		MaxResumeFailureRate:   0.02, // 2% 最大恢复失败率
		MinRewriteSuccessRate:  0.90, // 90% 最小改写成功率
		AllowCompatibilityPath: true, // 默认允许兼容路径
	}
}

// Decide 根据阈值和当前指标做出发布决策。
//
// 参数:
//   - th: 发布阈值
//   - plannerErrorRate: 当前规划器错误率
//   - resumeFailureRate: 当前恢复失败率
//   - rewriteSuccessRate: 当前改写成功率
//
// 返回: 发布决策结果
func (c RolloutConfig) Decide(th RolloutThresholds, plannerErrorRate, resumeFailureRate, rewriteSuccessRate float64) RolloutDecision {
	// 检查是否启用 Agentic 架构
	if !c.AgenticEnabled() {
		return RolloutDecision{Enabled: false, Reason: "agentic rollout disabled by config"}
	}
	// 检查改写成功率是否达标
	if rewriteSuccessRate < th.MinRewriteSuccessRate {
		return RolloutDecision{Enabled: false, Reason: "rewrite success rate below rollout threshold"}
	}
	// 检查规划器错误率是否超限
	if plannerErrorRate > th.MaxPlannerErrorRate {
		return RolloutDecision{Enabled: false, Reason: "planner error rate above rollout threshold"}
	}
	// 检查恢复失败率是否超限
	if resumeFailureRate > th.MaxResumeFailureRate {
		return RolloutDecision{Enabled: false, Reason: "resume failure rate above rollout threshold"}
	}
	return RolloutDecision{Enabled: true, Reason: "rollout thresholds satisfied"}
}
