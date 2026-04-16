package host

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

func TestBuildHostSummary_NormalizesAgentAndDefaults(t *testing.T) {
	t.Parallel()

	got := BuildHostSummary(contracts.DelegationSummary{TaskID: "t-1", Status: contracts.StatusReturned}, "node-a")
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
