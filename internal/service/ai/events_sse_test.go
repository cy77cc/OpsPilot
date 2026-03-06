package ai

import (
	"strings"
	"net/http/httptest"
	"testing"

	adkcore "github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/gin-gonic/gin"
)

func TestSSEWriterEmitAndFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	w := newSSEWriter(c, rec, "turn-test-1")
	if ok := w.Emit("meta", gin.H{"sessionId": "s1"}); !ok {
		t.Fatalf("expected meta emit ok")
	}
	if ok := w.Emit("delta", gin.H{"contentChunk": "hello"}); !ok {
		t.Fatalf("expected delta emit ok")
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close sse writer: %v", err)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: meta") {
		t.Fatalf("missing meta event: %s", body)
	}
	if !strings.Contains(body, "event: delta") {
		t.Fatalf("missing delta event: %s", body)
	}
	if !strings.Contains(body, "\"turn_id\":\"turn-test-1\"") {
		t.Fatalf("missing turn_id in payload: %s", body)
	}
	if !strings.Contains(body, "\"contentChunk\":\"hello\"") {
		t.Fatalf("missing delta payload content: %s", body)
	}
}

func TestApplyMessageChunkDoesNotAppendToolContentToAssistant(t *testing.T) {
	h := &handler{}
	var assistantContent strings.Builder
	var reasoningContent strings.Builder
	tracker := newToolEventTracker()
	emitted := map[string]int{}

	h.applyMessageChunk(func(event string, payload gin.H) bool {
		emitted[event]++
		return true
	}, tracker, &schema.Message{
		Role:    schema.Tool,
		Content: "{\"ok\":true,\"data\":\"raw tool output\"}",
	}, &assistantContent, &reasoningContent)

	if got := assistantContent.String(); got != "" {
		t.Fatalf("expected tool content to stay out of assistant transcript, got %q", got)
	}
	if emitted["tool_result"] != 0 {
		t.Fatalf("expected schema.Tool message not to emit extra tool_result event, got %d", emitted["tool_result"])
	}
}

func TestApplyMessageChunkDoesNotEmitAssistantToolCallDeclarations(t *testing.T) {
	h := &handler{}
	var assistantContent strings.Builder
	var reasoningContent strings.Builder
	tracker := newToolEventTracker()
	emitted := map[string]int{}

	h.applyMessageChunk(func(event string, payload gin.H) bool {
		emitted[event]++
		return true
	}, tracker, &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{
			{Type: "function"},
			{Type: "function", ID: "call-1", Function: schema.FunctionCall{Name: "host_list_inventory"}},
		},
	}, &assistantContent, &reasoningContent)

	if emitted["tool_call"] != 0 {
		t.Fatalf("expected assistant tool call declarations to be suppressed, got %d", emitted["tool_call"])
	}
	summary := tracker.summary()
	if summary.Calls != 0 {
		t.Fatalf("expected tracker to ignore assistant tool call declarations, got %d", summary.Calls)
	}
}

func TestHandleInterruptIncludesRootTargets(t *testing.T) {
	h := &handler{}
	var payload gin.H
	err := h.handleInterrupt(func(event string, p gin.H) bool {
		if event == "approval_required" {
			payload = p
		}
		return true
	}, &adkcore.InterruptInfo{
		Data: &aitools.ApprovalInfo{ToolName: "service_deploy_apply"},
		InterruptContexts: []*adkcore.InterruptCtx{
			{ID: "tool:approval-1", IsRootCause: true},
			{ID: "agent:parent", IsRootCause: false},
		},
	})
	if err != nil {
		t.Fatalf("handleInterrupt returned error: %v", err)
	}
	targets, _ := payload["interrupt_targets"].([]string)
	if len(targets) != 1 || targets[0] != "tool:approval-1" {
		t.Fatalf("unexpected interrupt targets: %#v", payload["interrupt_targets"])
	}
}

func TestInterruptPayloadFromSignalIncludesApprovalMetadata(t *testing.T) {
	sig := &adkcore.InterruptSignal{ID: "tool:host_batch_exec_apply:call-1"}
	sig.Info = &aitools.ApprovalInfo{
		ToolName:        "host_batch_exec_apply",
		ArgumentsInJSON: `{"host_ids":[2]}`,
		Risk:            aitools.ToolRiskHigh,
		Preview:         map[string]any{"count": 1},
	}
	sig.IsRootCause = true
	payload := interruptPayloadFromSignal(sig)

	targets, _ := payload["interrupt_targets"].([]string)
	if len(targets) != 1 || targets[0] != "tool:host_batch_exec_apply:call-1" {
		t.Fatalf("unexpected interrupt targets: %#v", payload["interrupt_targets"])
	}
	if payload["tool"] != "host_batch_exec_apply" {
		t.Fatalf("unexpected tool: %#v", payload["tool"])
	}
	if payload["approval_required"] != true {
		t.Fatalf("expected approval_required true, got %#v", payload["approval_required"])
	}
}
