package kubernetes

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

func TestBuildKubernetesSummary_NormalizesAgentAndDefaults(t *testing.T) {
	t.Parallel()

	got := BuildKubernetesSummary(contracts.DelegationSummary{TaskID: "t-1", Status: contracts.StatusReturned}, "prod-cluster", "payments")
	if got.AgentName != Name() {
		t.Fatalf("agent name = %q, want %q", got.AgentName, Name())
	}
	if got.Summary == "" {
		t.Fatal("expected non-empty summary")
	}
	if got.RecommendedNextAction == "" {
		t.Fatal("expected non-empty next action")
	}
}
