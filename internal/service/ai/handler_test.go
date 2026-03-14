package ai

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	coreai "github.com/cy77cc/OpsPilot/internal/ai"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aistate "github.com/cy77cc/OpsPilot/internal/ai/state"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestBuildResumeRequestMapsLegacyCheckpointToStepIdentity(t *testing.T) {
	req := buildResumeRequest(approvalResponseRequest{
		SessionID:    "session-1",
		PlanID:       "plan-1",
		CheckpointID: "legacy-step",
		Approved:     true,
	})

	if req.SessionID != "session-1" || req.PlanID != "plan-1" {
		t.Fatalf("unexpected request: %#v", req)
	}
	if req.StepID != "legacy-step" {
		t.Fatalf("StepID = %q, want %q", req.StepID, "legacy-step")
	}
	if req.Target != "legacy-step" {
		t.Fatalf("Target = %q, want %q", req.Target, "legacy-step")
	}
}

func TestBuildResumeResponseMarksLegacyADKCompatibility(t *testing.T) {
	payload := buildResumeResponse(&coreai.ResumeResult{
		Resumed:   true,
		SessionID: "session-1",
		PlanID:    "plan-1",
		StepID:    "step-1",
		TurnID:    "turn-1",
		Status:    "approved",
		Message:   "审批已通过，待审批步骤会继续执行。",
	}, true)

	if payload["compat_mode"] != "legacy_adk_resume" {
		t.Fatalf("compat_mode = %#v", payload["compat_mode"])
	}
	if payload["deprecated"] != true {
		t.Fatalf("deprecated = %#v", payload["deprecated"])
	}
	msg, _ := payload["message"].(string)
	if msg == "" || msg == "审批已通过，待审批步骤会继续执行。" {
		t.Fatalf("legacy message = %q", msg)
	}
	if payload["turn_id"] != "turn-1" {
		t.Fatalf("turn_id = %#v", payload["turn_id"])
	}
}

func TestAttachRolloutMetadataIncludesRuntimeFlags(t *testing.T) {
	payload := attachRolloutMetadata(map[string]any{
		"session_id": "session-1",
	}, coreai.RolloutConfig{
		UseMultiDomainArch:          true,
		UseTurnBlockStreaming:       true,
		UseModelFirstRuntime:        false,
		AllowLegacySemanticFallback: true,
	})

	if payload["runtime_mode"] != "compatibility" {
		t.Fatalf("runtime_mode = %#v, want compatibility", payload["runtime_mode"])
	}
	if payload["model_first_enabled"] != false {
		t.Fatalf("model_first_enabled = %#v, want false", payload["model_first_enabled"])
	}
	if payload["compatibility_enabled"] != true {
		t.Fatalf("compatibility_enabled = %#v, want true", payload["compatibility_enabled"])
	}
	if payload["turn_block_streaming_enabled"] != true {
		t.Fatalf("turn_block_streaming_enabled = %#v, want true", payload["turn_block_streaming_enabled"])
	}
}

func TestSessionHandlersRespectSceneAndExposeThoughtChain(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	handler := NewHTTPHandler(suite.SvcCtx)
	store := aistate.NewChatStore(suite.DB)
	ctx := t.Context()

	if err := store.AppendUserMessage(ctx, "session-global", 1, "global", "全局对话", "你好"); err != nil {
		t.Fatalf("AppendUserMessage(global) error = %v", err)
	}
	assistantID, err := store.CreateAssistantMessage(ctx, "session-global", 1, "global", "全局对话", "turn-global")
	if err != nil {
		t.Fatalf("CreateAssistantMessage(global) error = %v", err)
	}
	if err := store.UpdateAssistantMessage(ctx, "session-global", assistantID, "turn-global", aistate.ChatMessageRecord{
		Content: "回答",
		Status:  "completed",
		TraceID: "trace-1",
		ThoughtChain: []map[string]any{
			{"key": "rewrite", "title": "理解你的问题", "status": "success"},
		},
		RawEvidence: []string{"命令执行成功"},
	}); err != nil {
		t.Fatalf("UpdateAssistantMessage(global) error = %v", err)
	}
	if err := store.AppendUserMessage(ctx, "session-k8s", 1, "scene:k8s", "K8s 对话", "查日志"); err != nil {
		t.Fatalf("AppendUserMessage(k8s) error = %v", err)
	}

	t.Run("CurrentSession filters by scene", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/ai/sessions/current?scene=scene:k8s", nil)
		c.Request = req
		c.Set("uid", uint64(1))

		handler.CurrentSession(c)

		var resp struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.Data["id"] != "session-k8s" {
			t.Fatalf("CurrentSession id = %#v, want session-k8s", resp.Data["id"])
		}
	})

	t.Run("GetSession returns persisted thoughtChain", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/ai/sessions/session-global?scene=global", nil)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "session-global"}}
		c.Set("uid", uint64(1))

		handler.GetSession(c)

		var resp struct {
			Data struct {
				ID       string           `json:"id"`
				Messages []map[string]any `json:"messages"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.Data.ID != "session-global" {
			t.Fatalf("GetSession id = %q, want session-global", resp.Data.ID)
		}
		if len(resp.Data.Messages) != 2 {
			t.Fatalf("GetSession messages = %#v, want 2", resp.Data.Messages)
		}
		if turns, ok := resp.Data.Messages[1]["turns"]; ok && turns != nil {
			t.Fatalf("assistant legacy message unexpectedly embedded turns: %#v", turns)
		}
		if _, ok := resp.Data.Messages[1]["thoughtChain"]; !ok {
			t.Fatalf("assistant message missing thoughtChain: %#v", resp.Data.Messages[1])
		}
		if _, ok := resp.Data.Messages[1]["rawEvidence"]; !ok {
			t.Fatalf("assistant message missing rawEvidence: %#v", resp.Data.Messages[1])
		}
	})

	t.Run("GetSession returns structured turns alongside legacy messages", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/ai/sessions/session-global?scene=global", nil)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "session-global"}}
		c.Set("uid", uint64(1))

		handler.GetSession(c)

		var resp struct {
			Data struct {
				Turns []map[string]any `json:"turns"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if len(resp.Data.Turns) != 1 {
			t.Fatalf("turn count = %#v, want 1 assistant replay turn", resp.Data.Turns)
		}
		if resp.Data.Turns[0]["id"] != "turn-global" {
			t.Fatalf("assistant turn id = %#v, want turn-global", resp.Data.Turns[0]["id"])
		}
	})
}

func TestResumeStepStreamContinuesExistingTurnWithNativeLifecycle(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)
	prevV2 := config.CFG.FeatureFlags.AIAssistantV2
	disabled := false
	config.CFG.FeatureFlags.AIAssistantV2 = &disabled
	t.Cleanup(func() {
		config.CFG.FeatureFlags.AIAssistantV2 = prevV2
	})
	prevTurnBlock := config.CFG.AI.UseTurnBlockStreaming
	config.CFG.AI.UseTurnBlockStreaming = true
	t.Cleanup(func() {
		config.CFG.AI.UseTurnBlockStreaming = prevTurnBlock
	})
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	suite.SvcCtx.Rdb = client
	handler := NewHTTPHandler(suite.SvcCtx)
	store := runtime.NewExecutionStore(client, "ai:execution:")
	handler.orchestrator = coreai.NewOrchestrator(handler.sessions, store, common.PlatformDeps{DB: suite.DB})
	ctx := t.Context()
	if err := store.Save(ctx, runtime.ExecutionState{
		TraceID:   "trace-resume",
		SessionID: "session-resume",
		PlanID:    "plan-resume",
		TurnID:    "turn-resume",
		Message:   "restart nginx",
		Status:    runtime.ExecutionStatusWaitingApproval,
		Phase:     "waiting_approval:strict",
		Steps: map[string]runtime.StepState{
			"step-1": {
				StepID:             "step-1",
				Title:              "重启服务",
				Expert:             "hostops",
				Status:             runtime.StepWaitingApproval,
				Mode:               "mutating",
				Risk:               "high",
				UserVisibleSummary: "审批已拒绝，当前步骤不会执行。",
			},
		},
		PendingApproval: &runtime.PendingApproval{
			PlanID:      "plan-resume",
			StepID:      "step-1",
			Status:      "pending",
			Title:       "重启服务",
			Mode:        "mutating",
			Risk:        "high",
			Summary:     "重启 nginx 需要审批",
			ApprovalKey: "plan-resume:step-1",
		},
	}); err != nil {
		t.Fatalf("save execution state: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/ai/resume/step/stream", bytes.NewBufferString(`{"session_id":"session-resume","plan_id":"plan-resume","step_id":"step-1","approved":false}`))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Set("uid", uint64(1))

	handler.ResumeStepStream(c)

	body := w.Body.String()
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, body)
	}
	if got := w.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q", got)
	}
	for _, fragment := range []string{
		"event: meta",
		"event: turn_started",
		"event: phase_started",
		"event: turn_state",
		"event: step_complete",
		"event: done",
		"\"turn_id\":\"turn-resume\"",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("stream body missing %q: %s", fragment, body)
		}
	}
}
