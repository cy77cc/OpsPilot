package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cloudwego/eino/adk"
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
				"to":     "DiagnosisAgent",
				"intent": "diagnosis",
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
	if update.AssistantType != "DiagnosisAgent" || update.IntentType != "diagnosis" {
		t.Fatalf("unexpected handoff update: %#v", update)
	}
	if len(emitted) != 3 {
		t.Fatalf("expected all projected events to be emitted, got %#v", emitted)
	}
}

func TestConsumeProjectedEvents_ExcludesExecutorDeltaFromAssistantContent(t *testing.T) {
	t.Parallel()

	var builder strings.Builder
	l := &Logic{}
	seq := 0

	_, err := l.consumeProjectedEvents(context.Background(), "run-1", "sess-1", &seq, []airuntime.PublicStreamEvent{
		{
			Event: "delta",
			Data: map[string]any{
				"agent":   "executor",
				"content": "executor trace ",
			},
		},
		{
			Event: "delta",
			Data: map[string]any{
				"agent":   "replanner",
				"content": "final answer",
			},
		},
	}, func(string, any) {}, &builder)
	if err != nil {
		t.Fatalf("consume projected events: %v", err)
	}

	if got := builder.String(); got != "final answer" {
		t.Fatalf("expected assistant content to exclude executor delta, got %q", got)
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
	capturedMeta runtimectx.AIMetadata
	beforeRun    func(context.Context)
}

func (s *scriptedAgent) Name(context.Context) string        { return "scripted-agent" }
func (s *scriptedAgent) Description(context.Context) string { return "scripted agent for tests" }

func (s *scriptedAgent) Run(ctx context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	s.capturedMeta = runtimectx.AIMetadataFrom(ctx)
	if s.beforeRun != nil {
		s.beforeRun(ctx)
	}
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		for _, event := range s.runEvents {
			gen.Send(event)
		}
		gen.Close()
	}()
	return iter
}

func (s *scriptedAgent) Resume(ctx context.Context, _ *adk.ResumeInfo, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	return s.Run(ctx, nil)
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
