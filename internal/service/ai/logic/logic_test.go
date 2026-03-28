package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/adk"
	toolcomp "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildAugmentedMessage_IncludesStructuredSceneContextBlocks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:logic-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AIScenePrompt{}, &model.AISceneConfig{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}

	if err := db.Create(&model.AIScenePrompt{
		Scene:      "cluster",
		PromptText: "优先关注集群健康、节点状态和命名空间资源。",
		IsActive:   true,
	}).Error; err != nil {
		t.Fatalf("seed scene prompt: %v", err)
	}
	if err := db.Create(&model.AISceneConfig{
		Scene:            "cluster",
		Description:      "Kubernetes cluster operations",
		AllowedToolsJSON: `["cluster_inspect","k8s_topology"]`,
		BlockedToolsJSON: `["host_batch_exec_apply"]`,
		ConstraintsJSON:  `{"focus":"readonly diagnosis"}`,
	}).Error; err != nil {
		t.Fatalf("seed scene config: %v", err)
	}

	l := &Logic{svcCtx: &svc.ServiceContext{DB: db}}
	message := l.buildAugmentedMessage(context.Background(), "cluster", map[string]any{
		"route":       "/deployment/infrastructure/clusters/42",
		"resource_id": "42",
	}, "检查这个集群为什么不健康")

	for _, fragment := range []string{
		"[Hidden platform context",
		"[Scene]",
		"scene=cluster",
		"[Scene Context]",
		`scene_context={"resource_id":"42","route":"/deployment/infrastructure/clusters/42"}`,
		"[Scene Prompts & Constraints]",
		"scene_prompts=[",
		"[Tool Constraints]",
		"allowed_tools=[\"cluster_inspect\",\"k8s_topology\"]",
		"blocked_tools=[\"host_batch_exec_apply\"]",
		"These tool constraints are mandatory.",
		"User request:\n检查这个集群为什么不健康",
	} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("expected augmented message to contain %q, got: %s", fragment, message)
		}
	}
}

func TestRuntimeContext_AttachesServiceContext(t *testing.T) {
	baseCtx := context.WithValue(context.Background(), struct{}{}, "keep")
	svcCtx := &svc.ServiceContext{}
	l := &Logic{svcCtx: svcCtx}

	runtimeCtx := l.runtimeContext(baseCtx)

	if got := runtimectx.Services(runtimeCtx); got != svcCtx {
		t.Fatalf("expected runtime context to contain service context")
	}
	if got := runtimeCtx.Value(struct{}{}); got != "keep" {
		t.Fatalf("expected runtime context to preserve original values, got %#v", got)
	}
}

func TestChatInjectsAIMetadataIntoRuntimeContext(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-meta",
		UserID: 15,
		Scene:  "cluster",
		Title:  "Meta Session",
	})

	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("ok", nil), nil, schema.Assistant, ""),
		},
	}

	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-meta",
		Message:   "inspect this cluster",
		Scene:     "cluster",
		UserID:    15,
	}, func(string, any) {}); err != nil {
		t.Fatalf("chat returned error: %v", err)
	}

	if agent.capturedMeta.SessionID != "session-meta" {
		t.Fatalf("expected session meta to be injected, got %#v", agent.capturedMeta)
	}
	if agent.capturedMeta.UserID != 15 {
		t.Fatalf("expected user id 15 in meta, got %#v", agent.capturedMeta)
	}
	if agent.capturedMeta.Scene != "cluster" {
		t.Fatalf("expected scene cluster in meta, got %#v", agent.capturedMeta)
	}
	if agent.capturedMeta.RunID == "" {
		t.Fatalf("expected run id in meta, got %#v", agent.capturedMeta)
	}
}

func TestConsumeProjectedEvents_AccumulatesAssistantContentAndHandoff(t *testing.T) {
	t.Parallel()

	var (
		builder strings.Builder
		emitted []string
	)
	l := &Logic{}
	seq := 0

	update, err := l.consumeProjectedEvents(context.Background(), "run-1", "sess-1", &seq, []airuntime.PublicStreamEvent{
		{
			Event: "agent_handoff",
			Data: map[string]any{
				"to":     "K8sAgent",
				"intent": "k8s",
			},
		},
		{
			Event: "delta",
			Data: map[string]any{
				"content": "first ",
			},
		},
		{
			Event: "delta",
			Data: map[string]any{
				"content": "second",
			},
		},
	}, func(event string, data any) {
		emitted = append(emitted, event)
	}, &builder)
	if err != nil {
		t.Fatalf("consume projected events: %v", err)
	}

	if got := builder.String(); got != "first second" {
		t.Fatalf("unexpected assistant content: %q", got)
	}
	if update.AssistantType != "K8sAgent" || update.IntentType != "k8s" {
		t.Fatalf("unexpected handoff update: %#v", update)
	}
	if len(emitted) != 3 {
		t.Fatalf("expected all projected events to be emitted, got %#v", emitted)
	}
}

func TestConsumeProjectedEvents_IncludesAllAgentDeltaInAssistantContent(t *testing.T) {
	t.Parallel()

	var builder strings.Builder
	l := &Logic{}
	seq := 0

	_, err := l.consumeProjectedEvents(context.Background(), "run-1", "sess-1", &seq, []airuntime.PublicStreamEvent{
		{
			Event: "delta",
			Data: map[string]any{
				"agent":   "K8sAgent",
				"content": "K8s query result ",
			},
		},
		{
			Event: "delta",
			Data: map[string]any{
				"agent":   "OpsPilotAgent",
				"content": "final answer",
			},
		},
	}, func(string, any) {}, &builder)
	if err != nil {
		t.Fatalf("consume projected events: %v", err)
	}

	// DeepAgents: all agent content should be included
	if got := builder.String(); got != "K8s query result final answer" {
		t.Fatalf("expected all agent content, got %q", got)
	}
}

func TestMarshalProjectedEventIncludesToolApprovalAndRunState(t *testing.T) {
	t.Parallel()

	eventType, raw, err := marshalProjectedEvent("tool_approval", map[string]any{
		"approval_id":     "ap-1",
		"call_id":         "call-1",
		"tool_name":       "restart_workload",
		"preview":         map[string]any{"namespace": "prod"},
		"timeout_seconds": 300,
	})
	if err != nil {
		t.Fatalf("marshal tool_approval: %v", err)
	}
	if eventType != airuntime.EventTypeToolApproval {
		t.Fatalf("expected tool approval event type, got %q", eventType)
	}
	payload, err := airuntime.UnmarshalEventPayload(eventType, raw)
	if err != nil {
		t.Fatalf("decode tool approval payload: %v", err)
	}
	approval, ok := payload.(*airuntime.ToolApprovalPayload)
	if !ok {
		t.Fatalf("expected tool approval payload, got %#v", payload)
	}
	if approval.CallID != "call-1" || approval.ToolName != "restart_workload" {
		t.Fatalf("unexpected tool approval payload: %#v", approval)
	}

	eventType, raw, err = marshalProjectedEvent("run_state", map[string]any{
		"status": "waiting_approval",
		"agent":  "executor",
	})
	if err != nil {
		t.Fatalf("marshal run_state: %v", err)
	}
	if eventType != airuntime.EventTypeRunState {
		t.Fatalf("expected run state event type, got %q", eventType)
	}
	payload, err = airuntime.UnmarshalEventPayload(eventType, raw)
	if err != nil {
		t.Fatalf("decode run state payload: %v", err)
	}
	runState, ok := payload.(*airuntime.RunStatePayload)
	if !ok {
		t.Fatalf("expected run state payload, got %#v", payload)
	}
	if runState.Status != "waiting_approval" || runState.Agent != "executor" {
		t.Fatalf("unexpected run state payload: %#v", runState)
	}
}

func TestEmitExistingShellTerminal_WaitingApprovalEmitsRunState(t *testing.T) {
	l := &Logic{}
	shell := chatShell{
		Run: &model.AIRun{
			ID:     "run-waiting",
			Status: "waiting_approval",
		},
		AssistantMessage: &model.AIChatMessage{
			Content: "waiting for approval",
		},
	}

	var gotEvent string
	var gotData map[string]any
	l.emitExistingShellTerminal(context.Background(), shell, func(event string, data any) {
		gotEvent = event
		gotData, _ = data.(map[string]any)
	})

	if gotEvent != "run_state" {
		t.Fatalf("expected run_state event, got %q", gotEvent)
	}
	if gotData["status"] != "waiting_approval" {
		t.Fatalf("expected waiting_approval status payload, got %#v", gotData)
	}
	if gotData["run_id"] != "run-waiting" {
		t.Fatalf("expected run_id run-waiting, got %#v", gotData["run_id"])
	}
}

func TestEmitExistingShellTerminalReplaysAllPendingApprovals(t *testing.T) {
	db := newLogicTestDB(t)
	runID := "run-replay-approvals"
	sessionID := "session-replay-approvals"

	events := []model.AIRunEvent{
		{
			ID:          "evt-replay-1",
			RunID:       runID,
			SessionID:   sessionID,
			Seq:         1,
			EventType:   "tool_approval",
			ToolCallID:  "call-1",
			PayloadJSON: `{"approval_id":"ap-1","call_id":"call-1","tool_name":"host_exec","preview":{"cmd":"date"}}`,
		},
		{
			ID:          "evt-replay-2",
			RunID:       runID,
			SessionID:   sessionID,
			Seq:         2,
			EventType:   "tool_result",
			ToolCallID:  "call-1",
			PayloadJSON: `{"call_id":"call-1","tool_name":"host_exec","status":"success","content":"done"}`,
		},
		{
			ID:          "evt-replay-3",
			RunID:       runID,
			SessionID:   sessionID,
			Seq:         3,
			EventType:   "tool_approval",
			ToolCallID:  "call-2",
			PayloadJSON: `{"approval_id":"ap-2","call_id":"call-2","tool_name":"host_exec","preview":{"cmd":"whoami"}}`,
		},
		{
			ID:          "evt-replay-4",
			RunID:       runID,
			SessionID:   sessionID,
			Seq:         4,
			EventType:   "delta",
			PayloadJSON: `{"content":"intermediate"}`,
		},
	}
	for _, event := range events {
		if err := db.Create(&event).Error; err != nil {
			t.Fatalf("seed run event %s: %v", event.ID, err)
		}
	}

	l := &Logic{
		RunEventDAO: aidao.NewAIRunEventDAO(db),
	}
	shell := chatShell{
		Run: &model.AIRun{
			ID:     runID,
			Status: "waiting_approval",
		},
		AssistantMessage: &model.AIChatMessage{
			Content: "waiting for approvals",
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l.emitExistingShellTerminal(context.Background(), shell, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	})

	approvalCalls := make([]string, 0, 4)
	for _, event := range emitted {
		if event.Event != "tool_approval" {
			continue
		}
		payload, ok := event.Data.(map[string]any)
		if !ok {
			t.Fatalf("expected tool_approval payload map, got %T", event.Data)
		}
		approvalCalls = append(approvalCalls, strings.TrimSpace(stringValue(payload, "call_id")))
	}
	if len(approvalCalls) != 1 {
		t.Fatalf("expected one unresolved replayed tool approval, got %d (%v)", len(approvalCalls), approvalCalls)
	}
	if approvalCalls[0] != "call-2" {
		t.Fatalf("expected replay order [call-2], got %v", approvalCalls)
	}
	if emitted[len(emitted)-1].Event != "run_state" {
		t.Fatalf("expected final replay event to be run_state, got %q", emitted[len(emitted)-1].Event)
	}
}

func TestEmitExistingShellTerminalReplaysLatestApprovalSnapshotPerCall(t *testing.T) {
	db := newLogicTestDB(t)
	runID := "run-replay-latest-approval"
	sessionID := "session-replay-latest-approval"

	events := []model.AIRunEvent{
		{
			ID:          "evt-latest-1",
			RunID:       runID,
			SessionID:   sessionID,
			Seq:         1,
			EventType:   "tool_approval",
			ToolCallID:  "call-1",
			PayloadJSON: `{"approval_id":"ap-1","call_id":"call-1","tool_name":"host_exec","preview":{"cmd":"date"},"timeout_seconds":300}`,
		},
		{
			ID:          "evt-latest-2",
			RunID:       runID,
			SessionID:   sessionID,
			Seq:         2,
			EventType:   "tool_approval",
			ToolCallID:  "call-1",
			PayloadJSON: `{"approval_id":"ap-1-v2","call_id":"call-1","tool_name":"host_exec","preview":{"cmd":"whoami"},"timeout_seconds":600}`,
		},
	}
	for _, event := range events {
		if err := db.Create(&event).Error; err != nil {
			t.Fatalf("seed run event %s: %v", event.ID, err)
		}
	}

	l := &Logic{
		RunEventDAO: aidao.NewAIRunEventDAO(db),
	}
	shell := chatShell{
		Run: &model.AIRun{
			ID:     runID,
			Status: "waiting_approval",
		},
		AssistantMessage: &model.AIChatMessage{
			Content: "waiting for approvals",
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l.emitExistingShellTerminal(context.Background(), shell, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	})

	var approvals []map[string]any
	for _, event := range emitted {
		if event.Event != "tool_approval" {
			continue
		}
		payload, ok := event.Data.(map[string]any)
		if !ok {
			t.Fatalf("expected tool_approval payload map, got %T", event.Data)
		}
		approvals = append(approvals, payload)
	}
	if len(approvals) != 1 {
		t.Fatalf("expected single latest replayed approval for call-1, got %d", len(approvals))
	}
	if got := strings.TrimSpace(stringValue(approvals[0], "approval_id")); got != "ap-1-v2" {
		t.Fatalf("expected latest approval snapshot id ap-1-v2, got %q", got)
	}
	preview := mapValue(approvals[0], "preview")
	if got := strings.TrimSpace(stringValue(preview, "cmd")); got != "whoami" {
		t.Fatalf("expected latest preview cmd whoami, got %q", got)
	}
}

func TestEmitExistingShellTerminalCancelledOrExpiredEmitsRunStateSnapshot(t *testing.T) {
	testCases := []struct {
		name      string
		runStatus string
	}{
		{name: "cancelled", runStatus: "cancelled"},
		{name: "expired", runStatus: "expired"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := &Logic{}
			shell := chatShell{
				Run: &model.AIRun{
					ID:     "run-terminal",
					Status: tc.runStatus,
				},
				AssistantMessage: &model.AIChatMessage{
					Content: "approval stopped",
				},
			}

			var gotEvent string
			var gotData map[string]any
			l.emitExistingShellTerminal(context.Background(), shell, func(event string, data any) {
				gotEvent = event
				gotData, _ = data.(map[string]any)
			})

			if gotEvent != "run_state" {
				t.Fatalf("expected run_state event, got %q", gotEvent)
			}
			if gotData["status"] != tc.runStatus {
				t.Fatalf("expected %s status payload, got %#v", tc.runStatus, gotData)
			}
			if gotData["run_id"] != "run-terminal" {
				t.Fatalf("expected run_id run-terminal, got %#v", gotData["run_id"])
			}
			if gotData["summary"] != "approval stopped" {
				t.Fatalf("expected summary approval stopped, got %#v", gotData["summary"])
			}
		})
	}
}

func TestLegacySuspendedRun_IsIgnoredWhenCutoverEnabled(t *testing.T) {
	db := newLogicTestDB(t)
	runID := "run-legacy-suspended-cutover"
	sessionID := "session-legacy-suspended-cutover"

	events := []model.AIRunEvent{
		{
			ID:          "evt-legacy-1",
			RunID:       runID,
			SessionID:   sessionID,
			Seq:         1,
			EventType:   "tool_result",
			ToolCallID:  "call-legacy-1",
			PayloadJSON: `{"call_id":"call-legacy-1","tool_name":"host_exec_readonly","status":"suspended","approval_id":"ap-legacy-1","preview":{"command":"systemctl restart nginx"},"timeout_seconds":300}`,
		},
	}
	for _, event := range events {
		if err := db.Create(&event).Error; err != nil {
			t.Fatalf("seed run event %s: %v", event.ID, err)
		}
	}

	l := &Logic{
		RunEventDAO: aidao.NewAIRunEventDAO(db),
	}
	shell := chatShell{
		Run: &model.AIRun{
			ID:     runID,
			Status: "waiting_approval",
		},
		AssistantMessage: &model.AIChatMessage{
			Content: "waiting",
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 4)
	l.emitExistingShellTerminal(context.Background(), shell, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	})

	for _, event := range emitted {
		if event.Event == "tool_approval" {
			t.Fatalf("expected legacy suspended payload to be ignored after cutover, got events: %#v", emitted)
		}
	}
}

func TestBuildSessionTitle_TruncatesByRune(t *testing.T) {
	input := strings.Repeat("火", 60)
	got := buildSessionTitle(input)
	if !utf8.ValidString(got) {
		t.Fatalf("expected valid utf8 title, got %q", got)
	}
	if len([]rune(got)) != 48 {
		t.Fatalf("expected 48 runes, got %d", len([]rune(got)))
	}
}

func TestChatKeepsRunAliveOnRecoverableToolFailure(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-recoverable",
		UserID: 11,
		Scene:  "ai",
		Title:  "Recoverable Tool Failure",
	})

	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("checking cluster", nil), nil, schema.Assistant, ""),
			func() *adk.AgentEvent {
				event := adk.EventFromMessage(
					schema.ToolMessage(`{"ok":false}`, "call-1", schema.WithToolName("kubectl_get_pods")),
					nil,
					schema.Tool,
					"kubectl_get_pods",
				)
				event.Err = errors.New("tool execution failed")
				return event
			}(),
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-recoverable",
		Message:   "inspect pods",
		Scene:     "ai",
		UserID:    11,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected recoverable tool failure to return nil, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run record to exist")
	}
	if run.Status != "completed_with_tool_errors" {
		t.Fatalf("expected run status completed_with_tool_errors, got %q", run.Status)
	}

	var assistant model.AIChatMessage
	if err := db.Where("session_id = ? AND role = ?", "session-recoverable", "assistant").First(&assistant).Error; err != nil {
		t.Fatalf("load assistant message: %v", err)
	}
	if assistant.Content != "" {
		t.Fatalf("expected assistant message content to stay empty, got %q", assistant.Content)
	}
	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), runID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if projection == nil || projection.Status != "completed_with_tool_errors" {
		t.Fatalf("expected completed_with_tool_errors projection, got %#v", projection)
	}
	if summary := projectionSummaryContent(t, projection); summary == "" {
		t.Fatalf("expected projection summary content, got %#v", projection)
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-recoverable",
		Message:   "continue after tool failure",
		Scene:     "ai",
		UserID:    11,
	}, func(string, any) {}); err != nil {
		t.Fatalf("expected session to remain usable, got %v", err)
	}
}

func TestChatCircuitBreaksRepeatedSameToolFailure(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-circuit-breaker",
		UserID: 17,
		Scene:  "ai",
		Title:  "Tool Failure Circuit Breaker",
	})

	toolCall := func(callID string) *adk.AgentEvent {
		return adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{
			{
				ID: callID,
				Function: schema.FunctionCall{
					Name:      "host_exec",
					Arguments: `{"cmd":"date","scope":"cluster"}`,
				},
			},
		}), nil, schema.Assistant, "")
	}
	toolError := func(callID string) *adk.AgentEvent {
		return &adk.AgentEvent{
			AgentName: "executor",
			Err:       errors.New("failed to invoke tool[name:host_exec id:" + callID + "]: command denied"),
		}
	}

	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			toolCall("call-1"),
			toolError("call-1"),
			toolCall("call-2"),
			toolError("call-2"),
			adk.EventFromMessage(schema.AssistantMessage("should not be reached", nil), nil, schema.Assistant, ""),
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 16)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-circuit-breaker",
		Message:   "run repeated tool calls",
		Scene:     "ai",
		UserID:    17,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected circuit-broken recoverable failure to return nil, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run record to exist")
	}
	if run.Status != "completed_with_tool_errors" {
		t.Fatalf("expected run status completed_with_tool_errors, got %q", run.Status)
	}

	for _, event := range emitted {
		if event.Event != "delta" {
			continue
		}
		data, ok := event.Data.(map[string]any)
		if !ok {
			continue
		}
		if content, _ := data["content"].(string); strings.Contains(content, "should not be reached") {
			t.Fatalf("expected circuit breaker to stop later events, saw %#v", event)
		}
	}

	var toolResultCount int64
	if err := db.Model(&model.AIRunEvent{}).Where("run_id = ? AND event_type = ?", runID, string(airuntime.EventTypeToolResult)).Count(&toolResultCount).Error; err != nil {
		t.Fatalf("count tool_result events: %v", err)
	}
	if toolResultCount < 2 {
		t.Fatalf("expected repeated tool failures to be persisted, got %d", toolResultCount)
	}
}

func TestChatMarksFatalRuntimeFailure(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-fatal",
		UserID: 12,
		Scene:  "ai",
		Title:  "Fatal Runtime Failure",
	})

	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("starting run", nil), nil, schema.Assistant, ""),
			func() *adk.AgentEvent {
				event := adk.EventFromMessage(schema.AssistantMessage("runtime crashed", nil), nil, schema.Assistant, "")
				event.Err = errors.New("runtime failure")
				return event
			}(),
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-fatal",
		Message:   "run diagnosis",
		Scene:     "ai",
		UserID:    12,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected fatal runtime failure to return nil, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run record to exist")
	}
	if run.Status != "failed_runtime" {
		t.Fatalf("expected run status failed_runtime, got %q", run.Status)
	}

	var assistant model.AIChatMessage
	if err := db.Where("session_id = ? AND role = ?", "session-fatal", "assistant").First(&assistant).Error; err != nil {
		t.Fatalf("load assistant message: %v", err)
	}
	if assistant.Content != "" {
		t.Fatalf("expected assistant content to stay empty, got %q", assistant.Content)
	}
	if assistant.Status != "error" {
		t.Fatalf("expected assistant message status error, got %q", assistant.Status)
	}
	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), runID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if projection == nil || projection.Status != "failed_runtime" {
		t.Fatalf("expected failed_runtime projection, got %#v", projection)
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-fatal",
		Message:   "follow up after fatal failure",
		Scene:     "ai",
		UserID:    12,
	}, func(string, any) {}); err != nil {
		t.Fatalf("expected session to remain usable, got %v", err)
	}
}

func TestChatMarksFatalRuntimeFailure_PropagatesPersistArtifactsError(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-fatal-persist-error",
		UserID: 18,
		Scene:  "ai",
		Title:  "Fatal Runtime Persist Error",
	})
	if err := db.Migrator().DropTable(&model.AIRunProjection{}); err != nil {
		t.Fatalf("drop projection table: %v", err)
	}

	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("starting run", nil), nil, schema.Assistant, ""),
			{Err: errors.New("fatal agent failure")},
		},
	}

	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}

	err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-fatal-persist-error",
		Message:   "run diagnosis",
		Scene:     "ai",
		UserID:    18,
	}, func(string, any) {})
	if err == nil {
		t.Fatal("expected fatal runtime persist error to be returned")
	}
	if !strings.Contains(err.Error(), "persist run artifacts") {
		t.Fatalf("expected persist run artifacts error, got %v", err)
	}
}

func TestChatStopsConsumingAfterStreamingMessageError(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-stream-error",
		UserID: 13,
		Scene:  "ai",
		Title:  "Streaming Error",
	})

	streamReader, streamWriter := schema.Pipe[*schema.Message](2)
	go func() {
		streamWriter.Send(schema.AssistantMessage("partial output", nil), nil)
		streamWriter.Send(nil, errors.New("stream recv failed"))
		streamWriter.Close()
	}()

	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			{
				AgentName: "scripted-agent",
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{
						IsStreaming:   true,
						MessageStream: streamReader,
					},
				},
			},
			adk.EventFromMessage(schema.AssistantMessage("should not be emitted", nil), nil, schema.Assistant, ""),
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-stream-error",
		Message:   "trigger stream error",
		Scene:     "ai",
		UserID:    13,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected streaming message error to return nil, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run record to exist")
	}
	if run.Status != "failed_runtime" {
		t.Fatalf("expected run status failed_runtime, got %q", run.Status)
	}

	var assistant model.AIChatMessage
	if err := db.Where("session_id = ? AND role = ?", "session-stream-error", "assistant").First(&assistant).Error; err != nil {
		t.Fatalf("load assistant message: %v", err)
	}
	if assistant.Content != "" {
		t.Fatalf("expected assistant content to stay empty, got %q", assistant.Content)
	}
	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), runID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if projection == nil || projection.Status != "failed_runtime" {
		t.Fatalf("expected failed_runtime projection, got %#v", projection)
	}

	for _, event := range emitted {
		data, ok := event.Data.(map[string]any)
		if !ok {
			continue
		}
		content, _ := data["content"].(string)
		if strings.Contains(content, "should not be emitted") {
			t.Fatalf("expected no emitted event after stream failure, found %#v", event)
		}
	}

	events, err := aidao.NewAIRunEventDAO(db).ListByRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("list run events: %v", err)
	}
	var sawError bool
	for _, event := range events {
		if event.EventType == string(airuntime.EventTypeError) {
			sawError = true
			break
		}
	}
	if !sawError {
		t.Fatalf("expected persisted error event, got %#v", events)
	}
}

func TestChatKeepsRunAliveOnToolInvocationNodeError(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-tool-node-error",
		UserID: 14,
		Scene:  "ai",
		Title:  "Tool Invocation Node Error",
	})

	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{
				{ID: "call-499641dbc60642ffb66c1e05", Function: schema.FunctionCall{Name: "host_exec", Arguments: `{"host_id":1,"command":"systemctl status nginx"}`}},
			}), nil, schema.Assistant, ""),
			{
				AgentName: "executor",
				Err: errors.New("[NodeRunError] failed to stream tool call call-499641dbc60642ffb66c1e05: " +
					"[LocalFunc] failed to invoke tool, toolName=host_exec, err=command not allowed: only readonly commands are permitted\n" +
					"------------------------\nnode path: [node_1, ToolNode]"),
			},
			adk.EventFromMessage(schema.AssistantMessage("tool failed, continue with fallback", nil), nil, schema.Assistant, ""),
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-tool-node-error",
		Message:   "run host command",
		Scene:     "ai",
		UserID:    14,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected tool invocation node error to return nil, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run record to exist")
	}
	if run.Status != "completed_with_tool_errors" {
		t.Fatalf("expected run status completed_with_tool_errors, got %q", run.Status)
	}

	var (
		sawToolResult bool
		sawDone       bool
	)
	for _, event := range emitted {
		if event.Event == "error" {
			t.Fatalf("expected tool invocation node error to avoid terminal error event, got %#v", event)
		}
		if event.Event == "tool_result" {
			sawToolResult = true
		}
		if event.Event == "done" {
			sawDone = true
		}
	}
	if !sawToolResult {
		t.Fatal("expected synthesized tool_result event for tool invocation node error")
	}
	if !sawDone {
		t.Fatal("expected done event after tool invocation node error")
	}

	var assistant model.AIChatMessage
	if err := db.Where("session_id = ? AND role = ?", "session-tool-node-error", "assistant").First(&assistant).Error; err != nil {
		t.Fatalf("load assistant message: %v", err)
	}
	if assistant.Content != "" {
		t.Fatalf("expected assistant content to stay empty, got %q", assistant.Content)
	}
	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), runID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if projection == nil || projection.Status != "completed_with_tool_errors" {
		t.Fatalf("expected completed_with_tool_errors projection, got %#v", projection)
	}
	if summary := projectionSummaryContent(t, projection); !strings.Contains(summary, "fallback") {
		t.Fatalf("expected projection summary to preserve fallback content, got %q", summary)
	}
}

func TestChatKeepsRunAliveOnStreamingToolInvocationRecvError(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-stream-tool-error",
		UserID: 21,
		Scene:  "ai",
		Title:  "Streaming Tool Error",
	})

	streamReader, streamWriter := schema.Pipe[*schema.Message](2)
	go func() {
		streamWriter.Send(schema.AssistantMessage("", []schema.ToolCall{
			{ID: "call-stream-1", Function: schema.FunctionCall{Name: "host_exec", Arguments: `{"host_id":1,"command":"systemctl status nginx"}`}},
		}), nil)
		streamWriter.Send(nil, errors.New("[NodeRunError] failed to stream tool call call-stream-1: [LocalFunc] failed to invoke tool, toolName=host_exec, err=command not allowed"))
		streamWriter.Close()
	}()

	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			{
				AgentName: "executor",
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{
						IsStreaming:   true,
						MessageStream: streamReader,
					},
				},
			},
			adk.EventFromMessage(schema.AssistantMessage("tool failed, continue with fallback", nil), nil, schema.Assistant, ""),
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 10)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-stream-tool-error",
		Message:   "run host command",
		Scene:     "ai",
		UserID:    21,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected streaming tool recv error to return nil, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run record to exist")
	}
	if run.Status != "completed_with_tool_errors" {
		t.Fatalf("expected run status completed_with_tool_errors, got %q", run.Status)
	}

	var (
		sawToolResult bool
		sawDone       bool
	)
	for _, event := range emitted {
		if event.Event == "error" {
			t.Fatalf("expected streaming tool recv error to avoid terminal error event, got %#v", event)
		}
		if event.Event == "tool_result" {
			sawToolResult = true
		}
		if event.Event == "done" {
			sawDone = true
		}
	}
	if !sawToolResult {
		t.Fatal("expected synthesized tool_result event for streaming tool recv error")
	}
	if !sawDone {
		t.Fatal("expected done event after streaming tool recv error")
	}
}

func TestChatPausesWaitingApprovalOnStreamingInterruptRecvError(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-stream-interrupt",
		UserID: 27,
		Scene:  "ai",
		Title:  "Streaming Interrupt",
	})

	streamReader, streamWriter := schema.Pipe[*schema.Message](2)
	go func() {
		streamWriter.Send(nil, toolcomp.StatefulInterrupt(
			adk.AppendAddressSegment(context.Background(), adk.AddressSegmentTool, "host_exec"),
			map[string]any{
				"approval_id":     "approval-stream-1",
				"call_id":         "call-stream-approval-1",
				"tool_name":       "host_exec",
				"preview":         map[string]any{"command": "systemctl restart nginx"},
				"timeout_seconds": 300,
			},
			"state",
		))
		streamWriter.Close()
	}()

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter: &scriptedAgent{
			runEvents: []*adk.AgentEvent{
				{
					AgentName: "executor",
					Output: &adk.AgentOutput{
						MessageOutput: &adk.MessageVariant{
							IsStreaming:   true,
							MessageStream: streamReader,
						},
					},
				},
			},
		},
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-stream-interrupt",
		Message:   "restart service",
		Scene:     "ai",
		UserID:    27,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected streaming interrupt to return nil, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run record to exist")
	}
	if run.Status != "waiting_approval" {
		t.Fatalf("expected run status waiting_approval, got %q", run.Status)
	}

	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), runID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if projection == nil || projection.Status != "waiting_approval" {
		t.Fatalf("expected waiting_approval projection, got %#v", projection)
	}

	var (
		sawApproval bool
		sawDone     bool
		sawError    bool
	)
	for _, event := range emitted {
		if event.Event == "tool_approval" {
			sawApproval = true
		}
		if event.Event == "done" {
			sawDone = true
		}
		if event.Event == "error" {
			sawError = true
		}
	}
	if !sawApproval {
		t.Fatalf("expected streaming interrupt to emit tool_approval event, got %#v", emitted)
	}
	if sawDone {
		t.Fatalf("expected no done event when waiting approval, got %#v", emitted)
	}
	if sawError {
		t.Fatalf("expected no terminal error event when waiting approval, got %#v", emitted)
	}
}

func TestChatPausesWaitingApprovalOnIteratorInterruptError(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-iterator-interrupt",
		UserID: 33,
		Scene:  "ai",
		Title:  "Iterator Interrupt",
	})

	interruptErr := toolcomp.StatefulInterrupt(
		adk.AppendAddressSegment(context.Background(), adk.AddressSegmentTool, "host_exec"),
		map[string]any{
			"approval_id":     "approval-iterator-1",
			"call_id":         "call-iterator-approval-1",
			"tool_name":       "host_exec",
			"preview":         map[string]any{"command": "systemctl restart nginx"},
			"timeout_seconds": 300,
		},
		"state",
	)

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter: &scriptedAgent{
			runEvents: []*adk.AgentEvent{
				{
					AgentName: "executor",
					Err:       interruptErr,
				},
			},
		},
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-iterator-interrupt",
		Message:   "restart service",
		Scene:     "ai",
		UserID:    33,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected iterator interrupt to return nil, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run record to exist")
	}
	if run.Status != "waiting_approval" {
		t.Fatalf("expected run status waiting_approval, got %q", run.Status)
	}

	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), runID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if projection == nil || projection.Status != "waiting_approval" {
		t.Fatalf("expected waiting_approval projection, got %#v", projection)
	}

	var (
		sawApproval bool
		sawRunState bool
	)
	for _, event := range emitted {
		switch event.Event {
		case "tool_approval":
			sawApproval = true
		case "run_state":
			payload, _ := event.Data.(map[string]any)
			if payload != nil && stringValue(payload, "status") == "waiting_approval" {
				sawRunState = true
			}
		case "done", "error":
			t.Fatalf("expected interrupt path to avoid terminal events, got %#v", event)
		}
	}
	if !sawApproval {
		t.Fatal("expected tool_approval event from iterator interrupt")
	}
	if !sawRunState {
		t.Fatal("expected run_state waiting_approval event from iterator interrupt")
	}
}

func TestChatPausesWaitingApprovalOnIteratorInterruptedAction(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-iterator-interrupted-action",
		UserID: 34,
		Scene:  "ai",
		Title:  "Iterator Interrupted Action",
	})

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter: &scriptedAgent{
			runEvents: []*adk.AgentEvent{
				{
					AgentName: "executor",
					Action: &adk.AgentAction{
						Interrupted: &adk.InterruptInfo{
							Data: map[string]any{
								"approval_id":     "approval-iterator-action-1",
								"call_id":         "call-iterator-action-1",
								"tool_name":       "host_exec",
								"preview":         map[string]any{"command": "systemctl restart nginx"},
								"timeout_seconds": 300,
							},
						},
					},
				},
			},
		},
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-iterator-interrupted-action",
		Message:   "restart service",
		Scene:     "ai",
		UserID:    34,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected iterator interrupted action to return nil, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run record to exist")
	}
	if run.Status != "waiting_approval" {
		t.Fatalf("expected run status waiting_approval, got %q", run.Status)
	}

	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), runID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if projection == nil || projection.Status != "waiting_approval" {
		t.Fatalf("expected waiting_approval projection, got %#v", projection)
	}

	var (
		sawApproval bool
		sawRunState bool
	)
	for _, event := range emitted {
		switch event.Event {
		case "tool_approval":
			sawApproval = true
		case "run_state":
			payload, _ := event.Data.(map[string]any)
			if payload != nil && stringValue(payload, "status") == "waiting_approval" {
				sawRunState = true
			}
		case "done", "error":
			t.Fatalf("expected interrupt path to avoid terminal events, got %#v", event)
		}
	}
	if !sawApproval {
		t.Fatal("expected tool_approval event from iterator interrupted action")
	}
	if !sawRunState {
		t.Fatal("expected run_state waiting_approval event from iterator interrupted action")
	}
}

func TestChatPersistsDoneSummaryInProjection(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-summary",
		UserID: 16,
		Scene:  "ai",
		Title:  "Summary Persistence",
	})

	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("final answer", nil), nil, schema.Assistant, ""),
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-summary",
		Message:   "summarize",
		Scene:     "ai",
		UserID:    16,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("chat returned error: %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), runID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if projection == nil {
		t.Fatal("expected projection to exist")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(projection.ProjectionJSON), &payload); err != nil {
		t.Fatalf("decode projection: %v", err)
	}
	summary, _ := payload["summary"].(map[string]any)
	if summary == nil {
		t.Fatalf("expected projection summary, got %#v", payload)
	}
	if summary["content"] == "" {
		t.Fatalf("expected projection summary content, got %#v", summary)
	}
}

func TestChat_PersistsProjectionSummaryWithoutAssistantMessageContent(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-summary-skeleton",
		UserID: 17,
		Scene:  "ai",
		Title:  "Summary Skeleton",
	})

	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("final answer", nil), nil, schema.Assistant, ""),
		},
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l := &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-summary-skeleton",
		Message:   "summarize",
		Scene:     "ai",
		UserID:    17,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("chat returned error: %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	var assistant model.AIChatMessage
	if err := db.Where("session_id = ? AND role = ?", "session-summary-skeleton", "assistant").First(&assistant).Error; err != nil {
		t.Fatalf("load assistant message: %v", err)
	}
	if assistant.Content != "" {
		t.Fatalf("expected assistant session message content to stay empty, got %q", assistant.Content)
	}

	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), runID)
	if err != nil {
		t.Fatalf("load projection: %v", err)
	}
	if projection == nil {
		t.Fatal("expected projection to exist")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(projection.ProjectionJSON), &payload); err != nil {
		t.Fatalf("decode projection: %v", err)
	}
	summary, _ := payload["summary"].(map[string]any)
	if summary == nil {
		t.Fatalf("expected projection summary, got %#v", payload)
	}
	if summary["content"] == "" {
		t.Fatalf("expected projection summary content, got %#v", summary)
	}
}

func TestChat_PersistsShellBeforeRunnerStarts(t *testing.T) {
	db := newLogicTestDB(t)

	var observed shellSnapshot
	agent := &scriptedAgent{
		beforeRun: func(ctx context.Context) {
			meta := runtimectx.AIMetadataFrom(ctx)
			observed.SessionID = meta.SessionID
			observed.RunID = meta.RunID

			var session model.AIChatSession
			if err := db.First(&session, "id = ?", meta.SessionID).Error; err == nil {
				observed.HasSession = true
			}

			var run model.AIRun
			if err := db.First(&run, "id = ?", meta.RunID).Error; err == nil {
				observed.RunStatus = run.Status
				observed.UserMessageID = run.UserMessageID
				observed.AssistantMessageID = run.AssistantMessageID
			}

			var messages []model.AIChatMessage
			if err := db.Where("session_id = ?", meta.SessionID).Order("session_id_num ASC").Find(&messages).Error; err == nil {
				observed.MessageCount = len(messages)
				for _, message := range messages {
					switch message.Role {
					case "user":
						observed.UserStatus = message.Status
					case "assistant":
						observed.AssistantStatus = message.Status
					}
				}
			}
		},
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("ready", nil), nil, schema.Assistant, ""),
		},
	}

	l := newLogicTestLogic(db, agent)
	if err := l.Chat(context.Background(), ChatInput{
		Message: "inspect cluster",
		Scene:   "ai",
		UserID:  21,
	}, func(string, any) {}); err != nil {
		t.Fatalf("chat returned error: %v", err)
	}

	if !observed.HasSession {
		t.Fatal("expected session shell to exist before runner starts")
	}
	if observed.RunID == "" {
		t.Fatal("expected run shell to exist before runner starts")
	}
	if observed.MessageCount != 2 {
		t.Fatalf("expected user and assistant message shells before runner starts, got %d", observed.MessageCount)
	}
	if observed.RunStatus != "running" {
		t.Fatalf("expected run shell status running before runner starts, got %q", observed.RunStatus)
	}
	if observed.UserStatus != "done" {
		t.Fatalf("expected user shell status done before runner starts, got %q", observed.UserStatus)
	}
	if observed.AssistantStatus != "streaming" {
		t.Fatalf("expected assistant shell status streaming before runner starts, got %q", observed.AssistantStatus)
	}
}

func TestChat_PersistsFailedStartupTurn(t *testing.T) {
	db := newLogicTestDB(t)
	internalErr := errors.New("dial tcp 10.23.4.5:5432: connect: connection refused")
	emitted := make([]airuntime.PublicStreamEvent, 0, 4)

	l := newLogicTestLogic(db, &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			{Err: internalErr},
		},
	})

	if err := l.Chat(context.Background(), ChatInput{
		Message:         "start diagnostics",
		Scene:           "ai",
		UserID:          22,
		ClientRequestID: "req-startup-fail",
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected startup failure to be finalized in-band, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	if runID == "" {
		t.Fatal("expected meta event to include run_id")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected persisted run")
	}
	if run.Status != "failed_runtime" {
		t.Fatalf("expected failed_runtime startup status, got %q", run.Status)
	}
	if !strings.Contains(run.ErrorMessage, "10.23.4.5") {
		t.Fatalf("expected internal run error message to retain diagnostic details, got %q", run.ErrorMessage)
	}

	var assistant model.AIChatMessage
	if err := db.Where("id = ?", run.AssistantMessageID).First(&assistant).Error; err != nil {
		t.Fatalf("load assistant message: %v", err)
	}
	if assistant.Status != "error" {
		t.Fatalf("expected assistant status error, got %q", assistant.Status)
	}
	if strings.TrimSpace(assistant.Content) == "" {
		t.Fatal("expected assistant failure snapshot to be persisted")
	}
	if strings.Contains(assistant.Content, "10.23.4.5") {
		t.Fatalf("expected assistant failure snapshot to be sanitized, got %q", assistant.Content)
	}

	errorMessage := findEventField(t, emitted, "error", "message")
	if strings.TrimSpace(errorMessage) == "" {
		t.Fatal("expected error event payload")
	}
	if strings.Contains(errorMessage, "10.23.4.5") {
		t.Fatalf("expected emitted error payload to be sanitized, got %q", errorMessage)
	}
}

func TestChat_PersistsPartialBodyOnFatalStreamError(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-partial-fatal",
		UserID: 23,
		Scene:  "ai",
		Title:  "Partial Fatal",
	})

	streamReader, streamWriter := schema.Pipe[*schema.Message](2)
	go func() {
		streamWriter.Send(schema.AssistantMessage("partial answer", nil), nil)
		streamWriter.Send(nil, errors.New("read tcp 10.0.0.8:443->10.0.0.9:1234: i/o timeout"))
		streamWriter.Close()
	}()

	emitted := make([]airuntime.PublicStreamEvent, 0, 8)
	l := newLogicTestLogic(db, &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			{
				AgentName: "scripted-agent",
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{
						IsStreaming:   true,
						MessageStream: streamReader,
					},
				},
			},
			adk.EventFromMessage(schema.AssistantMessage("should not be emitted", nil), nil, schema.Assistant, ""),
		},
	})

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-partial-fatal",
		Message:   "trigger stream error",
		Scene:     "ai",
		UserID:    23,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected fatal stream error to be finalized in-band, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected persisted run")
	}
	if run.Status != "failed_runtime" {
		t.Fatalf("expected failed_runtime, got %q", run.Status)
	}

	var assistant model.AIChatMessage
	if err := db.Where("id = ?", run.AssistantMessageID).First(&assistant).Error; err != nil {
		t.Fatalf("load assistant message: %v", err)
	}
	if assistant.Status != "error" {
		t.Fatalf("expected assistant status error, got %q", assistant.Status)
	}
	if !strings.Contains(assistant.Content, "partial answer") {
		t.Fatalf("expected assistant snapshot to retain partial body, got %q", assistant.Content)
	}
	if strings.Contains(assistant.Content, "10.0.0.8") {
		t.Fatalf("expected assistant failure snapshot to be sanitized, got %q", assistant.Content)
	}

	for _, event := range emitted {
		data, ok := event.Data.(map[string]any)
		if !ok {
			continue
		}
		content, _ := data["content"].(string)
		if strings.Contains(content, "should not be emitted") {
			t.Fatalf("expected no emitted content after stream failure, found %#v", event)
		}
	}
}

func TestChat_RetryWithSameClientRequestIDReusesExistingShell(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-idempotent-shell",
		UserID: 24,
		Scene:  "ai",
		Title:  "Idempotent Shell",
	})

	firstEvents := make([]airuntime.PublicStreamEvent, 0, 4)
	l := newLogicTestLogic(db, &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			{Err: errors.New("startup failed once")},
		},
	})

	input := ChatInput{
		SessionID:       "session-idempotent-shell",
		ClientRequestID: "req-reuse-shell",
		Message:         "same turn",
		Scene:           "ai",
		UserID:          24,
	}

	if err := l.Chat(context.Background(), input, func(event string, data any) {
		firstEvents = append(firstEvents, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("first attempt returned error: %v", err)
	}

	secondEvents := make([]airuntime.PublicStreamEvent, 0, 4)
	if err := l.Chat(context.Background(), input, func(event string, data any) {
		secondEvents = append(secondEvents, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("second attempt returned error: %v", err)
	}

	firstRunID := findEventField(t, firstEvents, "meta", "run_id")
	secondRunID := findEventField(t, secondEvents, "meta", "run_id")
	if firstRunID == "" || secondRunID == "" {
		t.Fatalf("expected both attempts to emit meta run ids, got %q and %q", firstRunID, secondRunID)
	}
	if firstRunID != secondRunID {
		t.Fatalf("expected retry to reuse existing run shell, got %q then %q", firstRunID, secondRunID)
	}

	var runCount int64
	if err := db.Model(&model.AIRun{}).Where("session_id = ?", input.SessionID).Count(&runCount).Error; err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if runCount != 1 {
		t.Fatalf("expected one run shell for duplicate client request id, got %d", runCount)
	}

	var messageCount int64
	if err := db.Model(&model.AIChatMessage{}).Where("session_id = ?", input.SessionID).Count(&messageCount).Error; err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if messageCount != 2 {
		t.Fatalf("expected one user and one assistant shell, got %d messages", messageCount)
	}
}

func TestChat_ProjectionWriteFailureStillFinalizesMessageAndRun(t *testing.T) {
	db := newLogicTestDB(t)
	seedLogicTestSession(t, db, model.AIChatSession{
		ID:     "session-projection-failure",
		UserID: 25,
		Scene:  "ai",
		Title:  "Projection Failure",
	})
	if err := db.Migrator().DropTable(&model.AIRunProjection{}); err != nil {
		t.Fatalf("drop projection table: %v", err)
	}

	emitted := make([]airuntime.PublicStreamEvent, 0, 4)
	l := newLogicTestLogic(db, &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			{Err: errors.New("fatal runtime failure")},
		},
	})

	if err := l.Chat(context.Background(), ChatInput{
		SessionID: "session-projection-failure",
		Message:   "fail after shell",
		Scene:     "ai",
		UserID:    25,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("expected projection write failure to be best-effort, got %v", err)
	}

	runID := findEventField(t, emitted, "meta", "run_id")
	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if run == nil {
		t.Fatal("expected persisted run")
	}
	if run.Status != "failed_runtime" {
		t.Fatalf("expected run to finalize as failed_runtime, got %q", run.Status)
	}

	var assistant model.AIChatMessage
	if err := db.Where("id = ?", run.AssistantMessageID).First(&assistant).Error; err != nil {
		t.Fatalf("load assistant message: %v", err)
	}
	if assistant.Status != "error" {
		t.Fatalf("expected assistant message to finalize as error, got %q", assistant.Status)
	}
}

func TestChat_SuccessStatusesRemainCompletedAndCompletedWithToolErrors(t *testing.T) {
	testCases := []struct {
		name           string
		sessionID      string
		events         []*adk.AgentEvent
		expectedStatus string
	}{
		{
			name:      "completed",
			sessionID: "session-success-finalize",
			events: []*adk.AgentEvent{
				adk.EventFromMessage(schema.AssistantMessage("all good", nil), nil, schema.Assistant, ""),
			},
			expectedStatus: "completed",
		},
		{
			name:      "completed_with_tool_errors",
			sessionID: "session-success-with-tool-errors",
			events: []*adk.AgentEvent{
				adk.EventFromMessage(schema.AssistantMessage("checking", nil), nil, schema.Assistant, ""),
				func() *adk.AgentEvent {
					event := adk.EventFromMessage(
						schema.ToolMessage(`{"ok":false}`, "call-1", schema.WithToolName("kubectl_get_pods")),
						nil,
						schema.Tool,
						"kubectl_get_pods",
					)
					event.Err = errors.New("tool execution failed")
					return event
				}(),
			},
			expectedStatus: "completed_with_tool_errors",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := newLogicTestDB(t)
			seedLogicTestSession(t, db, model.AIChatSession{
				ID:     tc.sessionID,
				UserID: 26,
				Scene:  "ai",
				Title:  tc.name,
			})
			if err := db.Migrator().DropTable(&model.AIRunProjection{}); err != nil {
				t.Fatalf("drop projection table: %v", err)
			}

			emitted := make([]airuntime.PublicStreamEvent, 0, 8)
			l := newLogicTestLogic(db, &scriptedAgent{runEvents: tc.events})

			if err := l.Chat(context.Background(), ChatInput{
				SessionID: tc.sessionID,
				Message:   "finalize despite projection failure",
				Scene:     "ai",
				UserID:    26,
			}, func(event string, data any) {
				emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
			}); err != nil {
				t.Fatalf("expected projection persistence to be best-effort, got %v", err)
			}

			runID := findEventField(t, emitted, "meta", "run_id")
			run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), runID)
			if err != nil {
				t.Fatalf("load run: %v", err)
			}
			if run == nil {
				t.Fatal("expected persisted run")
			}
			if run.Status != tc.expectedStatus {
				t.Fatalf("expected run status %q, got %q", tc.expectedStatus, run.Status)
			}

			var assistant model.AIChatMessage
			if err := db.Where("id = ?", run.AssistantMessageID).First(&assistant).Error; err != nil {
				t.Fatalf("load assistant message: %v", err)
			}
			if assistant.Status != "done" {
				t.Fatalf("expected assistant status done, got %q", assistant.Status)
			}
		})
	}
}

func TestLogic_ChatRecoverableToolFailure(t *testing.T) {
	TestChatKeepsRunAliveOnRecoverableToolFailure(t)
}

func TestLogic_ChatApprovalInterruptMaintainsWaitingApprovalState(t *testing.T) {
	TestChatPausesWaitingApprovalOnIteratorInterruptError(t)
}

func TestLogic_ChatRunStateWaitingApprovalInterruptMaintainsOrdering(t *testing.T) {
	TestChatPausesWaitingApprovalOnIteratorInterruptedAction(t)
}

func TestLogic_ChatDoneStatusesRemainCompletedAndCompletedWithToolErrors(t *testing.T) {
	TestChat_SuccessStatusesRemainCompletedAndCompletedWithToolErrors(t)
}

func TestLogic_ResumeApprovalProjectsRecoverableToolErrorAndDone(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	store := &inMemoryCheckpointStore{data: make(map[string][]byte)}
	checkpointID := "checkpoint-resumeapproval-parity"
	approvalID := "approval-resumeapproval-parity"
	sessionID := "session-resumeapproval-parity"

	var resumeTargetID string
	agent := &scriptedAgent{
		runFn: func(ctx context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				defer gen.Close()
				event := adk.StatefulInterrupt(ctx, map[string]any{
					"approval_id":     approvalID,
					"call_id":         "call-resumeapproval-parity",
					"tool_name":       "host_exec",
					"preview":         map[string]any{"command": "uptime"},
					"timeout_seconds": 300,
				}, "resumeapproval-state")
				gen.Send(event)
			}()
			return iter
		},
		resumeFn: func(_ context.Context, info *adk.ResumeInfo, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
			if info == nil || !info.IsResumeTarget {
				t.Fatalf("expected resume target info, got %#v", info)
			}
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				defer gen.Close()
				gen.Send(adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{
					{
						ID: "call-resumeapproval-parity",
						Function: schema.FunctionCall{
							Name:      "host_exec",
							Arguments: `{"command":"uptime"}`,
						},
					},
				}), nil, schema.Assistant, ""))
				gen.Send(&adk.AgentEvent{
					AgentName: "executor",
					Err:       errors.New("[NodeRunError] failed to stream tool call call-resumeapproval-parity: [LocalFunc] failed to invoke tool, toolName=host_exec, err=command denied"),
				})
			}()
			return iter
		},
	}

	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: store,
	})
	iter := runner.Query(context.Background(), "seed resume checkpoint", adk.WithCheckPointID(checkpointID))
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Action == nil || event.Action.Interrupted == nil || len(event.Action.Interrupted.InterruptContexts) == 0 {
			continue
		}
		resumeTargetID = event.Action.Interrupted.InterruptContexts[0].ID
	}
	if strings.TrimSpace(resumeTargetID) == "" {
		t.Fatal("expected interrupt checkpoint target id")
	}

	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          sessionID,
		userID:             71,
		runID:              "run-resumeapproval-parity",
		userMessageID:      "msg-resumeapproval-parity-user",
		assistantMessageID: "msg-resumeapproval-parity-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     approvalID,
		CheckpointID:   checkpointID,
		SessionID:      sessionID,
		RunID:          "run-resumeapproval-parity",
		UserID:         71,
		ToolName:       "host_exec",
		ToolCallID:     resumeTargetID,
		ArgumentsJSON:  `{"command":"uptime"}`,
		PreviewJSON:    `{}`,
		Status:         "pending",
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(10 * time.Minute)),
		DecisionSource: ptrString("user"),
		PolicyVersion:  ptrString("v1"),
	})

	l := &Logic{
		svcCtx:          &svc.ServiceContext{DB: db},
		ChatDAO:         aidao.NewAIChatDAO(db),
		ApprovalDAO:     aidao.NewAIApprovalTaskDAO(db),
		AIRouter:        agent,
		CheckpointStore: store,
	}

	var emitted []airuntime.PublicStreamEvent
	if err := l.ResumeApproval(context.Background(), ResumeApprovalInput{
		ApprovalID: approvalID,
		Approved:   true,
		Comment:    "resume it",
		UserID:     71,
	}, func(event string, data any) {
		emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
	}); err != nil {
		t.Fatalf("resume approval: %v", err)
	}

	if len(emitted) == 0 || emitted[0].Event != "meta" {
		t.Fatalf("expected meta event first, got %#v", emitted)
	}
	assertPublicEventPresent(t, emitted, "tool_call")
	assertPublicEventPresent(t, emitted, "tool_result")
	assertPublicEventPresent(t, emitted, "done")
	assertPublicEventAbsent(t, emitted, "error")

	donePayload := findPublicEventPayload(t, emitted, "done")
	if status := stringValue(donePayload, "status"); status != "completed_with_tool_errors" {
		t.Fatalf("expected done status completed_with_tool_errors, got %#v", donePayload)
	}
	if strings.TrimSpace(stringValue(donePayload, "summary")) == "" {
		t.Fatalf("expected done summary fallback for tool error completion, got %#v", donePayload)
	}

	task, err := aidao.NewAIApprovalTaskDAO(db).GetByApprovalID(context.Background(), approvalID)
	if err != nil {
		t.Fatalf("reload approval task: %v", err)
	}
	if task == nil || task.Status != "approved" {
		t.Fatalf("expected approval task to be approved after resume, got %#v", task)
	}
}

func newLogicTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AIChatSession{},
		&model.AIChatMessage{},
		&model.AIRun{},
		&model.AIRunEvent{},
		&model.AIRunProjection{},
		&model.AIRunContent{},
		&model.AIScenePrompt{},
		&model.AISceneConfig{},
	); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

func newLogicTestLogic(db *gorm.DB, agent adk.ResumableAgent) *Logic {
	return &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
		AIRouter:         agent,
	}
}

func TestEnsureDoneSummaryAddsFallbackOnToolErrors(t *testing.T) {
	payload := map[string]any{}
	ensureDoneSummary(payload, "", true)
	got, _ := payload["summary"].(string)
	if strings.TrimSpace(got) == "" {
		t.Fatal("expected non-empty fallback summary for tool error completion")
	}
}

func seedLogicTestSession(t *testing.T, db *gorm.DB, session model.AIChatSession) {
	t.Helper()
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func findEventField(t *testing.T, events []airuntime.PublicStreamEvent, wantEvent, field string) string {
	t.Helper()
	for _, event := range events {
		if event.Event != wantEvent {
			continue
		}
		data, ok := event.Data.(map[string]any)
		if !ok {
			t.Fatalf("expected %s event data to be a map, got %T", wantEvent, event.Data)
		}
		value, _ := data[field].(string)
		return value
	}
	return ""
}

func findPublicEventPayload(t *testing.T, events []airuntime.PublicStreamEvent, wantEvent string) map[string]any {
	t.Helper()
	for _, event := range events {
		if event.Event != wantEvent {
			continue
		}
		payload, ok := event.Data.(map[string]any)
		if !ok {
			t.Fatalf("expected %s payload to be a map, got %T", wantEvent, event.Data)
		}
		return payload
	}
	t.Fatalf("expected %s event, got %#v", wantEvent, events)
	return nil
}

func assertPublicEventPresent(t *testing.T, events []airuntime.PublicStreamEvent, wantEvent string) {
	t.Helper()
	_ = findPublicEventPayload(t, events, wantEvent)
}

func assertPublicEventAbsent(t *testing.T, events []airuntime.PublicStreamEvent, wantEvent string) {
	t.Helper()
	for _, event := range events {
		if event.Event == wantEvent {
			t.Fatalf("expected %s to be absent, got %#v", wantEvent, events)
		}
	}
}

func projectionSummaryContent(t *testing.T, projection *model.AIRunProjection) string {
	t.Helper()
	if projection == nil {
		t.Fatal("expected projection to exist")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(projection.ProjectionJSON), &payload); err != nil {
		t.Fatalf("decode projection: %v", err)
	}
	summary, _ := payload["summary"].(map[string]any)
	if summary == nil {
		t.Fatalf("expected projection summary, got %#v", payload)
	}
	content, _ := summary["content"].(string)
	return content
}

type scriptedAgent struct {
	runEvents    []*adk.AgentEvent
	resumeEvents []*adk.AgentEvent
	capturedMeta runtimectx.AIMetadata
	beforeRun    func(context.Context)
	beforeResume func(context.Context, *adk.ResumeInfo)
	runFn        func(context.Context, *adk.AgentInput, ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent]
	resumeFn     func(context.Context, *adk.ResumeInfo, ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent]
}

func (s *scriptedAgent) Name(context.Context) string        { return "scripted-agent" }
func (s *scriptedAgent) Description(context.Context) string { return "scripted agent for tests" }

func (s *scriptedAgent) Run(ctx context.Context, input *adk.AgentInput, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	if s.runFn != nil {
		return s.runFn(ctx, input, opts...)
	}
	s.capturedMeta = runtimectx.AIMetadataFrom(ctx)
	if s.beforeRun != nil {
		s.beforeRun(ctx)
	}
	return iteratorFromAgentEvents(s.runEvents)
}

func (s *scriptedAgent) Resume(ctx context.Context, info *adk.ResumeInfo, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	if s.beforeResume != nil {
		s.beforeResume(ctx, info)
	}
	if s.resumeFn != nil {
		return s.resumeFn(ctx, info, opts...)
	}
	if s.resumeEvents != nil {
		return iteratorFromAgentEvents(s.resumeEvents)
	}
	return s.Run(ctx, nil)
}

type inMemoryCheckpointStore struct {
	data map[string][]byte
}

func (m *inMemoryCheckpointStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	value, ok := m.data[key]
	return value, ok, nil
}

func (m *inMemoryCheckpointStore) Set(_ context.Context, key string, value []byte) error {
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = value
	return nil
}

func iteratorFromAgentEvents(events []*adk.AgentEvent) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		for _, event := range events {
			gen.Send(event)
		}
	}()
	return iter
}

type shellSnapshot struct {
	SessionID          string
	RunID              string
	HasSession         bool
	RunStatus          string
	UserMessageID      string
	AssistantMessageID string
	MessageCount       int
	UserStatus         string
	AssistantStatus    string
}
