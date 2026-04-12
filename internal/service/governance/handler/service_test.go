package handler

import (
	"encoding/json"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/service/governance/logic"
)

func TestBuildEnvelope_UsesCanonicalFieldsOnly(t *testing.T) {
	approval := &logic.ApprovalInfo{Ticket: "gov-appr-1"}
	env := BuildEnvelope(logic.Decision{
		Allowed:  true,
		State:    logic.StateCompleted,
		Code:     CodeSuccess,
		Message:  "done",
		Approval: approval,
	}, 42, map[string]any{"changed": true})

	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	for _, key := range []string{"state", "approval", "audit_id", "code", "message", "data"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("expected envelope to contain key %q, got %v", key, got)
		}
	}
	if len(got) != 6 {
		t.Fatalf("expected only canonical keys, got %v", got)
	}
}
