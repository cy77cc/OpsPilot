package runtime

import "testing"

func TestDecodeEventPayload_ToolCall(t *testing.T) {
	payload, err := UnmarshalEventPayload(EventTypeToolCall, `{"agent":"executor","call_id":"call-1","tool_name":"host_list_inventory","arguments":{"keyword":"volcano"}}`)
	if err != nil {
		t.Fatalf("unmarshal tool call: %v", err)
	}
	toolCall, ok := payload.(*ToolCallPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", payload)
	}
	if toolCall.CallID != "call-1" || toolCall.ToolName != "host_list_inventory" {
		t.Fatalf("unexpected payload: %#v", toolCall)
	}
}

func TestDecodeEventPayload_Delta(t *testing.T) {
	payload, err := UnmarshalEventPayload(EventTypeDelta, `{"agent":"executor","content":"hello"}`)
	if err != nil {
		t.Fatalf("unmarshal delta: %v", err)
	}
	delta, ok := payload.(*DeltaPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", payload)
	}
	if delta.Agent != "executor" || delta.Content != "hello" {
		t.Fatalf("unexpected delta: %#v", delta)
	}
}

func TestDecodeEventPayload_ToolApproval(t *testing.T) {
	payload, err := UnmarshalEventPayload(EventTypeToolApproval, `{"approval_id":"ap-1","call_id":"call-1","tool_name":"restart_workload","preview":{"namespace":"prod"},"timeout_seconds":300}`)
	if err != nil {
		t.Fatalf("unmarshal tool approval: %v", err)
	}
	approval, ok := payload.(*ToolApprovalPayload)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", payload)
	}
	if approval.ApprovalID != "ap-1" || approval.CallID != "call-1" || approval.ToolName != "restart_workload" {
		t.Fatalf("unexpected tool approval payload: %#v", approval)
	}
}

func TestDecodeEventPayload_RunState(t *testing.T) {
	payload, err := UnmarshalEventPayload(EventTypeRunState, `{"status":"waiting_approval","agent":"executor"}`)
	if err != nil {
		t.Fatalf("unmarshal run state: %v", err)
	}
	runState, ok := payload.(*RunStatePayload)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", payload)
	}
	if runState.Status != "waiting_approval" || runState.Agent != "executor" {
		t.Fatalf("unexpected run state payload: %#v", runState)
	}
}

func TestDecodeEventPayload_RejectsUnknownShape(t *testing.T) {
	if _, err := UnmarshalEventPayload(EventTypeToolCall, `{"agent":"executor","tool_name":"host_list_inventory"}`); err == nil {
		t.Fatal("expected invalid tool call payload error")
	}
}
