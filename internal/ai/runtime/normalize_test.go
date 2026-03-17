package runtime

import (
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
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

func TestNormalizeAgentEvent_AssistantMessage(t *testing.T) {
	t.Parallel()

	event := adk.EventFromMessage(schema.AssistantMessage("investigating rollout", nil), nil, schema.Assistant, "")
	event.AgentName = "executor"

	got := NormalizeAgentEvent(event)

	if len(got) != 1 {
		t.Fatalf("expected one normalized event, got %d", len(got))
	}
	if got[0].Kind != NormalizedKindMessage {
		t.Fatalf("expected message event, got %q", got[0].Kind)
	}
	if got[0].Message == nil || got[0].Message.Content != "investigating rollout" {
		t.Fatalf("unexpected message payload: %#v", got[0].Message)
	}
}

func TestNormalizeAgentEvent_ToolResult(t *testing.T) {
	t.Parallel()

	message := schema.ToolMessage(`{"ok":true}`, "call-1", schema.WithToolName("kubectl_get_pods"))
	event := adk.EventFromMessage(message, nil, schema.Tool, message.ToolName)
	event.AgentName = "executor"

	got := NormalizeAgentEvent(event)

	if len(got) != 1 {
		t.Fatalf("expected one normalized event, got %d", len(got))
	}
	if got[0].Kind != NormalizedKindToolResult {
		t.Fatalf("expected tool result event, got %q", got[0].Kind)
	}
	if got[0].Tool == nil || got[0].Tool.ToolName != "kubectl_get_pods" || got[0].Tool.CallID != "call-1" {
		t.Fatalf("unexpected tool payload: %#v", got[0].Tool)
	}
}

func TestNormalizeAgentEvent_InterruptAction(t *testing.T) {
	t.Parallel()

	event := &adk.AgentEvent{
		AgentName: "executor",
		Action: &adk.AgentAction{
			Interrupted: &adk.InterruptInfo{
				Data: map[string]any{
					"approval_id":     "ap-1",
					"call_id":         "call-1",
					"tool_name":       "restart_workload",
					"preview":         map[string]any{"namespace": "prod"},
					"timeout_seconds": 300,
				},
			},
		},
	}

	got := NormalizeAgentEvent(event)

	if len(got) != 1 {
		t.Fatalf("expected one normalized event, got %d", len(got))
	}
	if got[0].Kind != NormalizedKindInterrupt {
		t.Fatalf("expected interrupt event, got %q", got[0].Kind)
	}
	if got[0].Interrupt == nil || got[0].Interrupt.ApprovalID != "ap-1" || got[0].Interrupt.ToolName != "restart_workload" {
		t.Fatalf("unexpected interrupt payload: %#v", got[0].Interrupt)
	}
}
