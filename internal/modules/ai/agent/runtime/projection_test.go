package runtime

import (
	"testing"

	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

func TestBuildProjection_IncludesDelegationNode(t *testing.T) {
	events := []ai.AIRunEvent{
		{
			ID:          "evt-1",
			RunID:       "run-1",
			SessionID:   "sess-1",
			EventType:   string(EventTypeDelegationNode),
			PayloadJSON: `{"delegation_id":"d-1","agent_name":"monitor","status":"returned","title":"Monitor summary","summary":"p95 increased","risk_level":"medium"}`,
		},
	}

	projection, _, err := BuildProjection(events)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	if len(projection.Blocks) != 1 {
		t.Fatalf("expected one block, got %d", len(projection.Blocks))
	}
	block := projection.Blocks[0]
	if block.Type != "delegation.node" {
		t.Fatalf("expected delegation.node block, got %q", block.Type)
	}
	if block.Agent != "monitor" {
		t.Fatalf("expected block agent monitor, got %q", block.Agent)
	}
	if summary, _ := block.Data["summary"].(string); summary != "p95 increased" {
		t.Fatalf("expected summary to be projected, got %#v", block.Data)
	}
}
