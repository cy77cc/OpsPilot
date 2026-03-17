package runtime

import (
	"testing"

	"github.com/cloudwego/eino/adk"
)

func TestNormalizeAgentEvent_TransferAction(t *testing.T) {
	t.Parallel()

	event := &adk.AgentEvent{
		AgentName: "OpsPilotAgent",
		Action:    adk.NewTransferToAgentAction("DiagnosisAgent"),
	}

	got := NormalizeAgentEvent(event)

	if len(got) != 1 {
		t.Fatalf("expected one normalized event, got %d", len(got))
	}
	if got[0].Kind != NormalizedKindHandoff {
		t.Fatalf("expected handoff event, got %q", got[0].Kind)
	}
	if got[0].Handoff == nil {
		t.Fatalf("expected handoff payload, got nil")
	}
	if got[0].Handoff.From != "OpsPilotAgent" || got[0].Handoff.To != "DiagnosisAgent" {
		t.Fatalf("unexpected handoff payload: %#v", got[0].Handoff)
	}
}
