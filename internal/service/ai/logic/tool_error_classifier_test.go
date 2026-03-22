package logic

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
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
