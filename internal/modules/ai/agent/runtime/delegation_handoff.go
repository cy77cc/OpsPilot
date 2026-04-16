package runtime

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

var explicitDelegationTargets = map[string]struct{}{
	"monitor":    {},
	"monitoring": {},
	"kubernetes": {},
	"host":       {},
	"cicd":       {},
}

var explicitDelegationIntents = map[string]struct{}{
	string(contracts.IntentMetricAnomalySummary): {},
	string(contracts.IntentResourceInventory):    {},
	string(contracts.IntentHostHealthSummary):    {},
	string(contracts.IntentPipelineFailure):      {},
	string(contracts.IntentReleaseReadiness):     {},
}

// IsDelegationHandoff reports whether a handoff is an explicit supervisor delegation.
func IsDelegationHandoff(_ string, to, intent string) bool {
	if _, ok := explicitDelegationTargets[normalizeAgentIdentity(to)]; ok {
		return true
	}
	_, ok := explicitDelegationIntents[strings.TrimSpace(intent)]
	return ok
}

func normalizeAgentIdentity(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if strings.HasSuffix(normalized, "_agent") {
		normalized = strings.TrimSuffix(normalized, "_agent")
	}
	if strings.HasSuffix(normalized, "agent") {
		normalized = strings.TrimSuffix(normalized, "agent")
		normalized = strings.TrimSuffix(normalized, "_")
		normalized = strings.TrimSuffix(normalized, "-")
	}
	return normalized
}
