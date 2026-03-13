package ai

import (
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/config"
)

type StreamEvent = runtime.StreamEvent
type StreamEmitter = runtime.StreamEmitter
type SelectedResource = runtime.SelectedResource
type RuntimeContext = runtime.RuntimeContext
type RunRequest = runtime.RunRequest
type ResumeRequest = runtime.ResumeRequest
type ResumeResult = runtime.ResumeResult

type RolloutConfig struct {
	UseMultiDomainArch          bool
	UseTurnBlockStreaming       bool
	UseModelFirstRuntime        bool
	AllowLegacySemanticFallback bool
	UseAssistantV2              bool
}

func CurrentRolloutConfig() RolloutConfig {
	return RolloutConfig{
		UseMultiDomainArch:          config.CFG.AI.UseMultiDomainArch,
		UseTurnBlockStreaming:       config.CFG.AI.UseTurnBlockStreaming,
		UseModelFirstRuntime:        boolOrDefault(config.CFG.FeatureFlags.AIModelFirstRuntime, false),
		AllowLegacySemanticFallback: boolOrDefault(config.CFG.FeatureFlags.AILegacySemanticFallback, true),
		UseAssistantV2:              boolOrDefault(config.CFG.FeatureFlags.AIAssistantV2, false),
	}
}

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

func (r RolloutConfig) ModelFirstEnabled() bool {
	return r.UseModelFirstRuntime
}

func (r RolloutConfig) CompatibilityEnabled() bool {
	return r.UseMultiDomainArch || r.AllowLegacySemanticFallback
}

func (r RolloutConfig) TurnBlockStreamingEnabled() bool {
	return r.UseTurnBlockStreaming
}

func boolOrDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}
