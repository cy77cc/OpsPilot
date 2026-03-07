package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	coreai "github.com/cy77cc/k8s-manage/internal/ai"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
	"github.com/gin-gonic/gin"
)

type fakeOrchestrator struct {
	chatReq       coreai.ChatStreamRequest
	chatCalled    bool
	resumeCalled  bool
	resumePayload map[string]any
}

func (f *fakeOrchestrator) ChatStream(_ context.Context, req coreai.ChatStreamRequest, emit func(event string, payload map[string]any) bool) error {
	f.chatCalled = true
	f.chatReq = req
	emit("meta", map[string]any{"sessionId": "sess-test"})
	emit("done", map[string]any{"stream_state": "ok"})
	return nil
}

func (f *fakeOrchestrator) ResumePayload(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	f.resumeCalled = true
	return f.resumePayload, nil
}

func newTestChatContext(t *testing.T, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewReader(raw))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("uid", uint64(7))
	return ctx, recorder
}

func TestChatDelegatesToAICoreOrchestrator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	orch := &fakeOrchestrator{}
	h := &AIHandler{
		orchestrator: orch,
		sessions:     logic.NewSessionStore(nil, nil),
		runtime:      logic.NewRuntimeStore(nil),
	}

	ctx, recorder := newTestChatContext(t, map[string]any{
		"message": "检查磁盘",
		"context": map[string]any{"scene": "global"},
	})
	h.chat(ctx)

	if !orch.chatCalled {
		t.Fatalf("expected orchestrator chat to be called")
	}
	if orch.chatReq.Message != "检查磁盘" {
		t.Fatalf("unexpected message: %q", orch.chatReq.Message)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "event: meta") || !strings.Contains(body, "event: done") {
		t.Fatalf("expected SSE meta and done events, got %q", body)
	}
}

func TestResumeApprovalDelegatesToAICoreOrchestrator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	orch := &fakeOrchestrator{resumePayload: map[string]any{"resumed": true, "content": "ok"}}
	h := &AIHandler{
		orchestrator: orch,
		sessions:     logic.NewSessionStore(nil, nil),
		runtime:      logic.NewRuntimeStore(nil),
	}

	ctx, recorder := newTestChatContext(t, map[string]any{
		"checkpoint_id": "sess-1",
		"target":        "call-1",
		"data":          true,
	})
	h.resumeADKApproval(ctx)

	if !orch.resumeCalled {
		t.Fatalf("expected orchestrator resume to be called")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

type fakeControlPlane struct{}

func (f *fakeControlPlane) ToolPolicy(context.Context, tools.ToolMeta, map[string]any) error { return nil }
func (f *fakeControlPlane) HasPermission(uint64, string) bool                                { return true }
func (f *fakeControlPlane) IsAdmin(uint64) bool                                              { return false }
func (f *fakeControlPlane) FindMeta(string) (tools.ToolMeta, bool)                           { return tools.ToolMeta{}, false }
