package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

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
}

func (s *scriptedAgent) Name(context.Context) string        { return "scripted-agent" }
func (s *scriptedAgent) Description(context.Context) string { return "scripted agent for tests" }

func (s *scriptedAgent) Run(ctx context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	s.capturedMeta = runtimectx.AIMetadataFrom(ctx)
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
