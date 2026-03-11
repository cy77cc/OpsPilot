package ai

import "testing"

func TestRolloutDecisionRespectsThresholdsAndCompatibilityPath(t *testing.T) {
	cfg := RolloutConfig{
		UseMultiDomainArch: true,
		UseAssistantV2:     true,
	}
	if !cfg.AgenticEnabled() {
		t.Fatalf("AgenticEnabled() = false, want true")
	}
	if !cfg.CompatibilityEnabled() {
		t.Fatalf("CompatibilityEnabled() = false, want true")
	}

	thresholds := DefaultRolloutThresholds()
	decision := cfg.Decide(thresholds, 0.05, 0.01, 0.95)
	if !decision.Enabled {
		t.Fatalf("Decision.Enabled = false, want true (%s)", decision.Reason)
	}

	decision = cfg.Decide(thresholds, 0.11, 0.01, 0.95)
	if decision.Enabled || decision.Reason != "planner error rate above rollout threshold" {
		t.Fatalf("unexpected planner gate decision: %#v", decision)
	}

	decision = cfg.Decide(thresholds, 0.05, 0.03, 0.95)
	if decision.Enabled || decision.Reason != "resume failure rate above rollout threshold" {
		t.Fatalf("unexpected resume gate decision: %#v", decision)
	}

	decision = cfg.Decide(thresholds, 0.05, 0.01, 0.80)
	if decision.Enabled || decision.Reason != "rewrite success rate below rollout threshold" {
		t.Fatalf("unexpected rewrite gate decision: %#v", decision)
	}
}

func TestRolloutDecisionDisabledWhenAgenticFlagOff(t *testing.T) {
	cfg := RolloutConfig{}
	decision := cfg.Decide(DefaultRolloutThresholds(), 0, 0, 1)
	if decision.Enabled {
		t.Fatalf("Decision.Enabled = true, want false")
	}
	if decision.Reason != "agentic rollout disabled by config" {
		t.Fatalf("Reason = %q", decision.Reason)
	}
	if !cfg.CompatibilityEnabled() {
		t.Fatalf("CompatibilityEnabled() = false, want true when agentic arch disabled")
	}
}
