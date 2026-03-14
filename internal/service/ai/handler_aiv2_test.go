package ai

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	coreai "github.com/cy77cc/OpsPilot/internal/ai"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/testutil"
	"github.com/gin-gonic/gin"
)

type fakeRuntime struct {
	runFn          func(context.Context, coreai.RunRequest, coreai.StreamEmitter) error
	resumeFn       func(context.Context, coreai.ResumeRequest) (*coreai.ResumeResult, error)
	resumeStreamFn func(context.Context, coreai.ResumeRequest, coreai.StreamEmitter) (*coreai.ResumeResult, error)
}

func (f *fakeRuntime) Run(ctx context.Context, req coreai.RunRequest, emit coreai.StreamEmitter) error {
	return f.runFn(ctx, req, emit)
}

func (f *fakeRuntime) Resume(ctx context.Context, req coreai.ResumeRequest) (*coreai.ResumeResult, error) {
	return f.resumeFn(ctx, req)
}

func (f *fakeRuntime) ResumeStream(ctx context.Context, req coreai.ResumeRequest, emit coreai.StreamEmitter) (*coreai.ResumeResult, error) {
	return f.resumeStreamFn(ctx, req, emit)
}

func TestChatUsesConfiguredRuntimeModeHeader(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)
	prevModelFirst := config.CFG.FeatureFlags.AIModelFirstRuntime
	enabled := true
	config.CFG.FeatureFlags.AIModelFirstRuntime = &enabled
	t.Cleanup(func() {
		config.CFG.FeatureFlags.AIModelFirstRuntime = prevModelFirst
	})

	handler := &HTTPHandler{
		svcCtx: &svc.ServiceContext{DB: suite.DB},
		orchestrator: &fakeRuntime{
			runFn: func(_ context.Context, _ coreai.RunRequest, emit coreai.StreamEmitter) error {
				emit(coreai.StreamEvent{Type: "meta", Data: map[string]any{"session_id": "session-1"}})
				emit(coreai.StreamEvent{Type: "done", Data: map[string]any{"status": "completed"}})
				return nil
			},
			resumeFn: func(context.Context, coreai.ResumeRequest) (*coreai.ResumeResult, error) { return nil, nil },
			resumeStreamFn: func(context.Context, coreai.ResumeRequest, coreai.StreamEmitter) (*coreai.ResumeResult, error) {
				return nil, nil
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/ai/chat", bytes.NewBufferString(`{"message":"hello","context":{"scene":"global"}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("uid", uint64(1))

	handler.Chat(c)

	if got := w.Header().Get("X-AI-Runtime-Mode"); got != "model_first" {
		t.Fatalf("X-AI-Runtime-Mode = %q, want model_first", got)
	}
	if !strings.Contains(w.Body.String(), "event: meta") || !strings.Contains(w.Body.String(), "event: done") {
		t.Fatalf("unexpected SSE body = %s", w.Body.String())
	}
}

func TestChatFallsBackToCompatibilityHeader(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)
	prevModelFirst := config.CFG.FeatureFlags.AIModelFirstRuntime
	prevCompat := config.CFG.FeatureFlags.AILegacySemanticFallback
	disabled := false
	compat := true
	config.CFG.FeatureFlags.AIModelFirstRuntime = &disabled
	config.CFG.FeatureFlags.AILegacySemanticFallback = &compat
	t.Cleanup(func() {
		config.CFG.FeatureFlags.AIModelFirstRuntime = prevModelFirst
		config.CFG.FeatureFlags.AILegacySemanticFallback = prevCompat
	})

	handler := &HTTPHandler{
		svcCtx: &svc.ServiceContext{DB: suite.DB},
		orchestrator: &fakeRuntime{
			runFn: func(_ context.Context, _ coreai.RunRequest, emit coreai.StreamEmitter) error {
				emit(coreai.StreamEvent{Type: "meta", Data: map[string]any{"session_id": "session-legacy"}})
				emit(coreai.StreamEvent{Type: "done", Data: map[string]any{"status": "completed"}})
				return nil
			},
			resumeFn: func(context.Context, coreai.ResumeRequest) (*coreai.ResumeResult, error) { return nil, nil },
			resumeStreamFn: func(context.Context, coreai.ResumeRequest, coreai.StreamEmitter) (*coreai.ResumeResult, error) {
				return nil, nil
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/ai/chat", bytes.NewBufferString(`{"message":"hello","context":{"scene":"global"}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("uid", uint64(1))

	handler.Chat(c)

	if got := w.Header().Get("X-AI-Runtime-Mode"); got != "compatibility" {
		t.Fatalf("X-AI-Runtime-Mode = %q, want compatibility", got)
	}
	if !strings.Contains(w.Body.String(), "session-legacy") {
		t.Fatalf("unexpected SSE body = %s", w.Body.String())
	}
}

func TestChatStreamsThoughtChainSSEFlow(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	handler := &HTTPHandler{
		svcCtx: &svc.ServiceContext{DB: suite.DB},
		orchestrator: &fakeRuntime{
			runFn: func(_ context.Context, _ coreai.RunRequest, emit coreai.StreamEmitter) error {
				emit(coreai.StreamEvent{Type: "meta", Data: map[string]any{"session_id": "session-stream", "plan_id": "plan-1"}})
				emit(coreai.StreamEvent{Type: "stage_delta", Data: map[string]any{"stage": "plan", "status": "loading", "summary": "正在整理执行步骤"}})
				emit(coreai.StreamEvent{Type: "step_update", Data: map[string]any{"plan_id": "plan-1", "step_id": "step-1", "tool": "scale_deployment", "status": "loading", "user_visible_summary": "准备调用扩容工具"}})
				emit(coreai.StreamEvent{Type: "approval_required", Data: map[string]any{"id": "approval-1", "session_id": "session-stream", "plan_id": "plan-1", "step_id": "step-1", "checkpoint_id": "cp-1", "tool": "scale_deployment", "status": "pending"}})
				emit(coreai.StreamEvent{Type: "delta", Data: map[string]any{"content_chunk": "请确认后继续"}})
				emit(coreai.StreamEvent{Type: "done", Data: map[string]any{"status": "waiting_approval"}})
				return nil
			},
			resumeFn: func(context.Context, coreai.ResumeRequest) (*coreai.ResumeResult, error) { return nil, nil },
			resumeStreamFn: func(context.Context, coreai.ResumeRequest, coreai.StreamEmitter) (*coreai.ResumeResult, error) {
				return nil, nil
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/ai/chat", bytes.NewBufferString(`{"message":"把 nginx 扩容到 3 个副本","context":{"scene":"global"}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("uid", uint64(1))

	handler.Chat(c)

	body := w.Body.String()
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, body)
	}
	for _, fragment := range []string{
		"event: meta",
		"event: stage_delta",
		"\"stage\":\"plan\"",
		"event: step_update",
		"\"step_id\":\"step-1\"",
		"event: approval_required",
		"\"checkpoint_id\":\"cp-1\"",
		"event: delta",
		"请确认后继续",
		"event: done",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("stream body missing %q: %s", fragment, body)
		}
	}
}

func TestChatStreamsNativePlanExecuteReplanEvents(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	handler := &HTTPHandler{
		svcCtx: &svc.ServiceContext{DB: suite.DB},
		orchestrator: &fakeRuntime{
			runFn: func(_ context.Context, _ coreai.RunRequest, emit coreai.StreamEmitter) error {
				emit(coreai.StreamEvent{Type: "meta", Data: map[string]any{"session_id": "session-native", "plan_id": "plan-42", "turn_id": "turn-1"}})
				emit(coreai.StreamEvent{Type: "phase_started", Data: map[string]any{"phase": "planning", "title": "整理执行步骤", "status": "loading"}})
				emit(coreai.StreamEvent{Type: "plan_generated", Data: map[string]any{
					"plan_id": "plan-42",
					"steps": []map[string]any{
						{"id": "step-1", "content": "检查集群状态", "tool_hint": "get_cluster_info"},
					},
					"total": 1,
				}})
				emit(coreai.StreamEvent{Type: "phase_complete", Data: map[string]any{"phase": "planning", "status": "success"}})
				emit(coreai.StreamEvent{Type: "phase_started", Data: map[string]any{"phase": "executing", "title": "执行步骤", "status": "loading"}})
				emit(coreai.StreamEvent{Type: "step_started", Data: map[string]any{"step_id": "step-1", "title": "检查集群状态", "tool_name": "get_cluster_info", "status": "running"}})
				emit(coreai.StreamEvent{Type: "tool_call", Data: map[string]any{"step_id": "step-1", "tool_name": "get_cluster_info", "params": map[string]any{"namespace": "kube-system"}}})
				emit(coreai.StreamEvent{Type: "tool_result", Data: map[string]any{"step_id": "step-1", "tool_name": "get_cluster_info", "status": "success", "result": map[string]any{"ok": true, "data": "3 pods found"}}})
				emit(coreai.StreamEvent{Type: "step_complete", Data: map[string]any{"step_id": "step-1", "status": "success", "summary": "集群状态正常"}})
				emit(coreai.StreamEvent{Type: "replan_triggered", Data: map[string]any{"reason": "发现后续需要重新规划"}})
				emit(coreai.StreamEvent{Type: "done", Data: map[string]any{"status": "completed"}})
				return nil
			},
			resumeFn: func(context.Context, coreai.ResumeRequest) (*coreai.ResumeResult, error) { return nil, nil },
			resumeStreamFn: func(context.Context, coreai.ResumeRequest, coreai.StreamEmitter) (*coreai.ResumeResult, error) {
				return nil, nil
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/v1/ai/chat", bytes.NewBufferString(`{"message":"检查集群状态","context":{"scene":"global"}}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("uid", uint64(1))

	handler.Chat(c)

	body := w.Body.String()
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, body)
	}
	for _, fragment := range []string{
		"event: meta",
		"event: phase_started",
		"\"phase\":\"planning\"",
		"event: plan_generated",
		"\"tool_hint\":\"get_cluster_info\"",
		"event: phase_complete",
		"event: step_started",
		"event: tool_call",
		"event: tool_result",
		"event: step_complete",
		"event: replan_triggered",
		"event: done",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("stream body missing %q: %s", fragment, body)
		}
	}
}

func TestResumeStepStreamPassesCheckpointIdentityToRuntime(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	var captured coreai.ResumeRequest
	handler := &HTTPHandler{
		svcCtx: &svc.ServiceContext{DB: suite.DB},
		orchestrator: &fakeRuntime{
			runFn:    func(context.Context, coreai.RunRequest, coreai.StreamEmitter) error { return nil },
			resumeFn: func(context.Context, coreai.ResumeRequest) (*coreai.ResumeResult, error) { return nil, nil },
			resumeStreamFn: func(_ context.Context, req coreai.ResumeRequest, emit coreai.StreamEmitter) (*coreai.ResumeResult, error) {
				captured = req
				emit(coreai.StreamEvent{Type: "meta", Data: map[string]any{"session_id": req.SessionID, "plan_id": req.PlanID}})
				emit(coreai.StreamEvent{Type: "stage_delta", Data: map[string]any{"stage": "execute", "status": "loading", "summary": "继续执行审批后的步骤"}})
				emit(coreai.StreamEvent{Type: "step_update", Data: map[string]any{"step_id": req.StepID, "status": "success", "checkpoint_id": req.CheckpointID}})
				emit(coreai.StreamEvent{Type: "done", Data: map[string]any{"status": "completed"}})
				return &coreai.ResumeResult{
					Resumed:   true,
					SessionID: req.SessionID,
					PlanID:    req.PlanID,
					StepID:    req.StepID,
					Status:    "completed",
				}, nil
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/api/v1/ai/resume/step/stream",
		bytes.NewBufferString(`{"session_id":"sess-1","plan_id":"plan-1","step_id":"step-1","checkpoint_id":"cp-1","approved":true}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("uid", uint64(1))

	handler.ResumeStepStream(c)

	if captured.SessionID != "sess-1" || captured.PlanID != "plan-1" || captured.StepID != "step-1" {
		t.Fatalf("captured request = %#v", captured)
	}
	if captured.CheckpointID != "cp-1" || !captured.Approved {
		t.Fatalf("captured request = %#v", captured)
	}

	body := w.Body.String()
	for _, fragment := range []string{
		"event: meta",
		"event: stage_delta",
		"event: step_update",
		"\"checkpoint_id\":\"cp-1\"",
		"event: done",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("stream body missing %q: %s", fragment, body)
		}
	}
}

func TestResumeStepStreamEmitsNativeExecutionEvents(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	handler := &HTTPHandler{
		svcCtx: &svc.ServiceContext{DB: suite.DB},
		orchestrator: &fakeRuntime{
			runFn:    func(context.Context, coreai.RunRequest, coreai.StreamEmitter) error { return nil },
			resumeFn: func(context.Context, coreai.ResumeRequest) (*coreai.ResumeResult, error) { return nil, nil },
			resumeStreamFn: func(_ context.Context, req coreai.ResumeRequest, emit coreai.StreamEmitter) (*coreai.ResumeResult, error) {
				emit(coreai.StreamEvent{Type: "meta", Data: map[string]any{"session_id": req.SessionID, "plan_id": req.PlanID, "turn_id": "turn-resume"}})
				emit(coreai.StreamEvent{Type: "phase_started", Data: map[string]any{"phase": "executing", "title": "执行步骤", "status": "running"}})
				emit(coreai.StreamEvent{Type: "step_complete", Data: map[string]any{"step_id": req.StepID, "status": "success", "summary": "审批后的步骤已完成"}})
				emit(coreai.StreamEvent{Type: "done", Data: map[string]any{"status": "completed"}})
				return &coreai.ResumeResult{
					Resumed:   true,
					SessionID: req.SessionID,
					PlanID:    req.PlanID,
					StepID:    req.StepID,
					Status:    "completed",
				}, nil
			},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		"POST",
		"/api/v1/ai/resume/step/stream",
		bytes.NewBufferString(`{"session_id":"sess-1","plan_id":"plan-1","step_id":"step-1","checkpoint_id":"cp-1","approved":true}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("uid", uint64(1))

	handler.ResumeStepStream(c)

	body := w.Body.String()
	for _, fragment := range []string{
		"event: meta",
		"event: phase_started",
		"\"phase\":\"executing\"",
		"event: step_complete",
		"审批后的步骤已完成",
		"event: done",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("stream body missing %q: %s", fragment, body)
		}
	}
}
