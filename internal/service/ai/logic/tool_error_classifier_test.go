package logic

import (
	"encoding/json"
	"errors"
	"testing"

	toolcomp "github.com/cloudwego/eino/components/tool"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

func TestClassifyRecoverableToolInvocationError(t *testing.T) {
	t.Parallel()

	t.Run("stream tool call pattern", func(t *testing.T) {
		t.Parallel()

		info, ok := classifyRecoverableToolInvocationError(errors.New(
			"[NodeRunError] failed to stream tool call call-123: [LocalFunc] failed to invoke tool, toolName=host_exec, err=command denied",
		))
		if !ok {
			t.Fatal("expected stream tool call error to be classified")
		}
		if info.CallID != "call-123" || info.ToolName != "host_exec" {
			t.Fatalf("unexpected classification: %#v", info)
		}
	})

	t.Run("invoke tool pattern", func(t *testing.T) {
		t.Parallel()

		info, ok := classifyRecoverableToolInvocationError(errors.New(
			"failed to invoke tool[name:exec_command id:call-456]: command denied",
		))
		if !ok {
			t.Fatal("expected invoke tool error to be classified")
		}
		if info.CallID != "call-456" || info.ToolName != "exec_command" {
			t.Fatalf("unexpected classification: %#v", info)
		}
	})

	t.Run("rejects noise", func(t *testing.T) {
		t.Parallel()

		if info, ok := classifyRecoverableToolInvocationError(errors.New("tool execution failed")); ok || info != nil {
			t.Fatalf("expected noise to be rejected, got %#v", info)
		}
	})

	t.Run("generic invoke pattern", func(t *testing.T) {
		t.Parallel()

		info, ok := classifyRecoverableToolInvocationError(errors.New(
			"[Executor] failed to invoke tool: tool_name=host_exec_change call_id=call-789 timeout",
		))
		if !ok {
			t.Fatal("expected generic invoke tool error to be classified")
		}
		if info.CallID != "call-789" || info.ToolName != "host_exec_change" {
			t.Fatalf("unexpected generic classification: %#v", info)
		}
	})
}

func TestRecoverableToolErrorFromEventRejectsBusinessToolResult(t *testing.T) {
	t.Parallel()

	event := adk.EventFromMessage(
		schema.ToolMessage(`{"ok":false,"status":"error"}`, "call-1", schema.WithToolName("kubectl_get_pods")),
		nil,
		schema.Tool,
		"kubectl_get_pods",
	)
	event.Err = errors.New("tool execution failed")

	if result, ok := recoverableToolErrorFromEvent(event); ok || result != nil {
		t.Fatalf("expected tool_result business error to bypass invocation classification, got %#v", result)
	}
}

func TestBuildSyntheticToolResultEvent(t *testing.T) {
	t.Parallel()

	info := &recoverableToolInvocationError{
		CallID:   "call-9",
		ToolName: "host_exec",
		Message:  "failed to invoke tool[name:host_exec id:call-9]: command denied",
	}

	event, ok := buildSyntheticToolResultEvent("executor", info)
	if !ok {
		t.Fatal("expected synthetic tool result event")
	}
	if event.AgentName != "executor" {
		t.Fatalf("expected agent name executor, got %q", event.AgentName)
	}
	if event.Output == nil || event.Output.MessageOutput == nil {
		t.Fatal("expected synthetic event to include tool message output")
	}
	if event.Output.MessageOutput.Role != schema.Tool {
		t.Fatalf("expected tool role, got %v", event.Output.MessageOutput.Role)
	}
	msg, err := event.Output.MessageOutput.GetMessage()
	if err != nil {
		t.Fatalf("get synthetic message: %v", err)
	}
	if msg.ToolCallID != "call-9" || msg.ToolName != "host_exec" {
		t.Fatalf("unexpected synthetic message: %#v", msg)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(msg.Content), &payload); err != nil {
		t.Fatalf("decode synthetic payload: %v", err)
	}
	if payload["error_type"] != "tool_invocation" {
		t.Fatalf("expected tool_invocation error type, got %#v", payload)
	}
	if payload["call_id"] != "call-9" || payload["tool_name"] != "host_exec" {
		t.Fatalf("unexpected payload contents: %#v", payload)
	}
}

func TestRecoverableInterruptEventFromEvent_ConvertsInterruptSignalError(t *testing.T) {
	t.Parallel()

	interruptErr := toolcomp.StatefulInterrupt(
		adk.AppendAddressSegment(t.Context(), adk.AddressSegmentTool, "host_exec"),
		map[string]any{
			"approval_id":     "ap-1",
			"call_id":         "call-1",
			"tool_name":       "host_exec",
			"timeout_seconds": 300,
		},
		"state",
	)

	event := &adk.AgentEvent{AgentName: "executor", Err: interruptErr}
	converted, ok := recoverableInterruptEventFromEvent(event)
	if !ok || converted == nil {
		t.Fatal("expected interrupt error to be converted")
	}
	if converted.Action == nil || converted.Action.Interrupted == nil {
		t.Fatalf("expected interrupted action, got %#v", converted)
	}
	payload, ok := converted.Action.Interrupted.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected interrupt payload map, got %T", converted.Action.Interrupted.Data)
	}
	if payload["approval_id"] != "ap-1" || payload["tool_name"] != "host_exec" {
		t.Fatalf("unexpected interrupt payload: %#v", payload)
	}
}

func TestRecoverableInterruptEventFromErr_ConvertsInterruptSignalError(t *testing.T) {
	t.Parallel()

	interruptErr := toolcomp.StatefulInterrupt(
		adk.AppendAddressSegment(t.Context(), adk.AddressSegmentTool, "host_exec"),
		map[string]any{
			"approval_id":     "ap-3",
			"call_id":         "call-3",
			"tool_name":       "host_exec",
			"timeout_seconds": 300,
		},
		"state",
	)

	converted, ok := recoverableInterruptEventFromErr(interruptErr, "executor")
	if !ok || converted == nil {
		t.Fatal("expected interrupt error to be converted")
	}
	if converted.AgentName != "executor" {
		t.Fatalf("expected agent name executor, got %q", converted.AgentName)
	}
	if converted.Action == nil || converted.Action.Interrupted == nil {
		t.Fatalf("expected interrupted action, got %#v", converted)
	}
	payload, ok := converted.Action.Interrupted.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected interrupt payload map, got %T", converted.Action.Interrupted.Data)
	}
	if payload["approval_id"] != "ap-3" || payload["tool_name"] != "host_exec" {
		t.Fatalf("unexpected interrupt payload: %#v", payload)
	}
}

func TestRecoverableInterruptEventFromEvent_PassthroughInterruptedAction(t *testing.T) {
	t.Parallel()

	event := &adk.AgentEvent{
		AgentName: "executor",
		Err:       errors.New("interrupt"),
		Action: &adk.AgentAction{
			Interrupted: &adk.InterruptInfo{Data: map[string]any{"approval_id": "ap-2"}},
		},
	}
	converted, ok := recoverableInterruptEventFromEvent(event)
	if !ok || converted != event {
		t.Fatalf("expected passthrough interrupted event, got %#v", converted)
	}
}

func TestRecoverableInterruptEventFromEvent_ActionInterruptBackfillsDataFromContexts(t *testing.T) {
	t.Parallel()

	event := &adk.AgentEvent{
		AgentName: "executor",
		Action: &adk.AgentAction{
			Interrupted: &adk.InterruptInfo{
				InterruptContexts: []*adk.InterruptCtx{
					{
						ID:          "it-1",
						IsRootCause: true,
						Info: map[string]any{
							"approval_id": "ap-ctx-1",
							"call_id":     "call-ctx-1",
							"tool_name":   "host_exec",
						},
					},
				},
			},
		},
	}

	converted, ok := recoverableInterruptEventFromEvent(event)
	if !ok || converted == nil {
		t.Fatal("expected interrupted action to pass through")
	}
	if converted.Action == nil || converted.Action.Interrupted == nil {
		t.Fatalf("expected interrupted action, got %#v", converted)
	}
	payload, ok := converted.Action.Interrupted.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected backfilled payload map, got %T", converted.Action.Interrupted.Data)
	}
	if payload["approval_id"] != "ap-ctx-1" || payload["tool_name"] != "host_exec" {
		t.Fatalf("unexpected backfilled payload: %#v", payload)
	}
}

func TestRecoverableInterruptEventFromEvent_InterruptSignalKeepsContextsForFallback(t *testing.T) {
	t.Parallel()

	interruptErr := toolcomp.StatefulInterrupt(
		adk.AppendAddressSegment(t.Context(), adk.AddressSegmentTool, "host_exec"),
		map[string]any{
			"tool_name": "host_exec",
			// simulate missing call_id from upstream tool-call metadata
		},
		"state",
	)

	event := &adk.AgentEvent{AgentName: "executor", Err: interruptErr}
	converted, ok := recoverableInterruptEventFromEvent(event)
	if !ok || converted == nil {
		t.Fatal("expected interrupt error to be converted")
	}
	if converted.Action == nil || converted.Action.Interrupted == nil {
		t.Fatalf("expected interrupted action, got %#v", converted)
	}
	if len(converted.Action.Interrupted.InterruptContexts) == 0 {
		t.Fatalf("expected interrupt contexts to be preserved for fallback id extraction, got %#v", converted.Action.Interrupted)
	}

	normalized := airuntime.NormalizeAgentEvent(converted)
	if len(normalized) != 1 || normalized[0].Kind != airuntime.NormalizedKindInterrupt {
		t.Fatalf("expected normalized interrupt event after fallback, got %#v", normalized)
	}
	if normalized[0].Interrupt == nil || normalized[0].Interrupt.CallID == "" || normalized[0].Interrupt.ToolName != "host_exec" {
		t.Fatalf("unexpected normalized interrupt payload: %#v", normalized[0].Interrupt)
	}
}
