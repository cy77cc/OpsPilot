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

func TestNormalizeAgentEvent_AssistantWithToolCalls(t *testing.T) {
	tests := []struct {
		name       string
		event      *adk.AgentEvent
		wantKinds  []NormalizedKind
		wantLen    int
		wantToolID string
	}{
		{
			name: "assistant with content and tool_calls",
			event: &adk.AgentEvent{
				AgentName: "executor",
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{Message: &schema.Message{
						Role:    schema.Assistant,
						Content: "some content",
						ToolCalls: []schema.ToolCall{
							{ID: "call-1", Function: schema.FunctionCall{Name: "tool_a", Arguments: `{"arg": "value"}`}},
						},
					}},
				},
			},
			wantKinds:  []NormalizedKind{NormalizedKindMessage, NormalizedKindToolCall},
			wantLen:    2,
			wantToolID: "call-1",
		},
		{
			name: "assistant with only tool_calls",
			event: &adk.AgentEvent{
				AgentName: "executor",
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{Message: &schema.Message{
						Role:      schema.Assistant,
						ToolCalls: []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "tool_a"}}},
					}},
				},
			},
			wantKinds:  []NormalizedKind{NormalizedKindToolCall},
			wantLen:    1,
			wantToolID: "call-1",
		},
		{
			name: "assistant with nil tool_calls",
			event: &adk.AgentEvent{
				AgentName: "executor",
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{Message: &schema.Message{
						Role:      schema.Assistant,
						Content:   "content",
						ToolCalls: nil,
					}},
				},
			},
			wantKinds: []NormalizedKind{NormalizedKindMessage},
			wantLen:   1,
		},
		{
			name: "assistant with empty tool_calls",
			event: &adk.AgentEvent{
				AgentName: "executor",
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{Message: &schema.Message{
						Role:      schema.Assistant,
						Content:   "content",
						ToolCalls: []schema.ToolCall{},
					}},
				},
			},
			wantKinds: []NormalizedKind{NormalizedKindMessage},
			wantLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeAgentEvent(tt.event)
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d events, got %d", tt.wantLen, len(got))
			}
			for i, kind := range tt.wantKinds {
				if got[i].Kind != kind {
					t.Errorf("event %d: expected kind %s, got %s", i, kind, got[i].Kind)
				}
				// 验证 tool call ID
				if kind == NormalizedKindToolCall && tt.wantToolID != "" {
					if got[i].Tool == nil || got[i].Tool.CallID != tt.wantToolID {
						t.Errorf("expected tool call_id=%s, got %v", tt.wantToolID, got[i].Tool)
					}
				}
			}
		})
	}
}
