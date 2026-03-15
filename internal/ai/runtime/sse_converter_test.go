package runtime

import "testing"

func TestSSEConverter_MetaEventCarriesCanonicalIDs(t *testing.T) {
	converter := NewSSEConverter()

	meta := converter.OnMeta("sess-1", "plan-1", "turn-1", "trace-1")
	if meta.Type != EventMeta {
		t.Fatalf("unexpected meta event type: %s", meta.Type)
	}
	if got := meta.Data["session_id"]; got != "sess-1" {
		t.Fatalf("expected session id, got %#v", got)
	}
	if got := meta.Data["plan_id"]; got != "plan-1" {
		t.Fatalf("expected plan id, got %#v", got)
	}
	if got := meta.Data["trace_id"]; got != "trace-1" {
		t.Fatalf("expected trace id, got %#v", got)
	}
}

func TestSSEConverter_ToolCallEvent(t *testing.T) {
	converter := NewSSEConverter()
	event := converter.OnToolCall("call-1", "get_pods", "获取 Pod 列表", `{"namespace":"default"}`)

	if event.Type != EventToolCall {
		t.Fatalf("unexpected event type: %s", event.Type)
	}
	if got := event.Data["call_id"]; got != "call-1" {
		t.Fatalf("expected call_id, got %#v", got)
	}
	if got := event.Data["tool_name"]; got != "get_pods" {
		t.Fatalf("expected tool_name, got %#v", got)
	}
}

func TestSSEConverter_ToolApprovalEvent(t *testing.T) {
	converter := NewSSEConverter()
	event := converter.OnToolApproval("call-1", "delete_pod", "删除 Pod", "high", "即将删除 pod/nginx", `{"name":"nginx"}`, "approval-1", "cp-1")

	if event.Type != EventToolApproval {
		t.Fatalf("unexpected event type: %s", event.Type)
	}
	if got := event.Data["call_id"]; got != "call-1" {
		t.Fatalf("expected call_id, got %#v", got)
	}
	if got := event.Data["risk"]; got != "high" {
		t.Fatalf("expected risk, got %#v", got)
	}
}

func TestSSEConverter_ToolResultEvent(t *testing.T) {
	converter := NewSSEConverter()
	event := converter.OnToolResult("call-1", "get_pods", `{"items":[]}`)

	if event.Type != EventToolResult {
		t.Fatalf("unexpected event type: %s", event.Type)
	}
	if got := event.Data["call_id"]; got != "call-1" {
		t.Fatalf("expected call_id, got %#v", got)
	}
}

func TestSSEConverter_DoneEvent(t *testing.T) {
	converter := NewSSEConverter()
	done := converter.OnDone("completed")

	if done.Type != EventDone {
		t.Fatalf("unexpected done event type: %s", done.Type)
	}
	if got := done.Data["status"]; got != "completed" {
		t.Fatalf("expected completed status, got %#v", got)
	}
}

func TestSSEConverter_ErrorEvent(t *testing.T) {
	converter := NewSSEConverter()
	err := converter.OnError("execution", nil)

	if err.Type != EventError {
		t.Fatalf("unexpected error event type: %s", err.Type)
	}
}
