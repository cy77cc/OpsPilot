package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	aiobs "github.com/cy77cc/OpsPilot/internal/ai/observability"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

func TestBuildRuntimeContextEnvelope_OmitsEmptyBlock(t *testing.T) {
	if got := buildRuntimeContextEnvelope(airuntime.RuntimeContext{}); got != "" {
		t.Fatalf("expected empty envelope, got %q", got)
	}
}

func TestBuildRuntimeContextEnvelope_UsesCanonicalFieldOrder(t *testing.T) {
	got := buildRuntimeContextEnvelope(airuntime.RuntimeContext{
		Scene:       "deployment:hosts",
		ProjectID:   "1",
		CurrentPage: "/deployment/infrastructure/hosts",
		SelectedResources: []airuntime.SelectedResource{
			{Name: "host-a", Type: "host"},
			{ID: "svc-1", Type: "service"},
		},
	})

	want := strings.Join([]string{
		"[Runtime Context]",
		"scene: deployment:hosts",
		"project: 1",
		"page: /deployment/infrastructure/hosts",
		"selected_resources: host-a(host), svc-1(service)",
	}, "\n")
	if got != want {
		t.Fatalf("unexpected envelope:\n%s", got)
	}
}

func TestBuildRuntimeContextEnvelope_SummarizesSelectedResourcesAndNormalizesWhitespace(t *testing.T) {
	got := buildRuntimeContextEnvelope(airuntime.RuntimeContext{
		Scene: "host:detail",
		SelectedResources: []airuntime.SelectedResource{
			{Name: "host-a\nprod", Type: " host "},
			{Name: " svc-a\tblue ", Type: "service"},
		},
	})

	if !strings.Contains(got, "selected_resources: host-a prod(host), svc-a blue(service)") {
		t.Fatalf("expected normalized selected resources, got %q", got)
	}
	if strings.Contains(got, "\nprod") || strings.Contains(got, "\t") {
		t.Fatalf("expected whitespace normalization, got %q", got)
	}
}

func TestBuildRuntimeContextEnvelope_DoesNotDumpMetadataOrUserContext(t *testing.T) {
	got := buildRuntimeContextEnvelope(airuntime.RuntimeContext{
		Scene:       "deployment:hosts",
		ProjectID:   "1",
		UserContext: map[string]any{"uid": 7},
		Metadata:    map[string]any{"scene": "shadow"},
	})

	if strings.Contains(got, "uid") || strings.Contains(got, "shadow") {
		t.Fatalf("expected metadata and user context to be omitted, got %q", got)
	}
}

func TestComposeUserInput_PreservesRawUserRequest(t *testing.T) {
	envelope := "[Runtime Context]\nscene: deployment:hosts"
	raw := "  请检查主机状态\n并告诉我异常点。  "

	got := composeUserInput(envelope, raw)
	want := envelope + "\n\n[User Request]\n" + raw
	if got != want {
		t.Fatalf("unexpected composed input:\n%s", got)
	}
}

func TestRun_UsesEnvelopeAndPreservesRawUserRequest(t *testing.T) {
	ctx := context.Background()
	var capturedQuery string

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(assistantEvent("agent", "已收到请求。"))
	gen.Close()

	orchestrator := &Orchestrator{
		executions: airuntime.NewExecutionStore(nil, ""),
		converter:  airuntime.NewSSEConverter(),
		runQuery: func(_ context.Context, query string, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
			capturedQuery = query
			return iter
		},
	}

	// scene bias is encoded in the runtime envelope, but the raw user request stays untouched.
	err := orchestrator.Run(ctx, airuntime.RunRequest{
		Message: "  请检查主机状态\n并告诉我异常点。  ",
		RuntimeContext: airuntime.RuntimeContext{
			Scene:       "deployment:hosts",
			ProjectID:   "1",
			CurrentPage: "/deployment/infrastructure/hosts",
		},
	}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	want := strings.Join([]string{
		"[Runtime Context]",
		"scene: deployment:hosts",
		"project: 1",
		"page: /deployment/infrastructure/hosts",
		"",
		"[User Request]",
		"  请检查主机状态",
		"并告诉我异常点。  ",
	}, "\n")
	if capturedQuery != want {
		t.Fatalf("unexpected query:\n%s", capturedQuery)
	}
}

type lifecycleObserver struct {
	records []aiobs.ThoughtChainLifecycleRecord
}

func (o *lifecycleObserver) OnThoughtChainLifecycle(record aiobs.ThoughtChainLifecycleRecord) {
	o.records = append(o.records, record)
}

func TestOrchestratorStreamExecution_EmitsPrimaryEvents(t *testing.T) {
	ctx := context.Background()
	observer := &lifecycleObserver{}
	unregister := aiobs.RegisterObserver(observer)
	defer unregister()

	orchestrator := &Orchestrator{
		executions: airuntime.NewExecutionStore(nil, ""),
		converter:  airuntime.NewSSEConverter(),
	}
	state := airuntime.ExecutionState{
		TraceID:      "trace-1",
		SessionID:    "sess-1",
		PlanID:       "plan-1",
		TurnID:       "turn-1",
		Scene:        "deployment:hosts",
		Status:       airuntime.ExecutionStatusRunning,
		Phase:        "running",
		CheckpointID: "checkpoint-1",
		Steps:        map[string]airuntime.StepState{},
	}

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(assistantEvent("agent", "先查看主机状态。"))
	gen.Send(toolEvent("host_list_inventory", `{"total":1,"list":[{"id":1,"name":"test"}]}`))
	gen.Send(assistantEvent("agent", "主机状态正常。"))
	gen.Close()

	var emitted []airuntime.StreamEvent
	result, err := orchestrator.streamExecution(ctx, iter, &state, func(event airuntime.StreamEvent) bool {
		emitted = append(emitted, event)
		return true
	})
	if err != nil {
		t.Fatalf("streamExecution returned error: %v", err)
	}
	if result == nil || result.Status != string(airuntime.ExecutionStatusCompleted) {
		t.Fatalf("expected completed result, got %#v", result)
	}

	assertEventTypePresent(t, emitted, airuntime.EventChainStarted)
	assertEventTypePresent(t, emitted, airuntime.EventDelta)
	assertEventTypePresent(t, emitted, airuntime.EventToolResult)
	assertEventTypePresent(t, emitted, airuntime.EventChainCompleted)
	assertEventTypePresent(t, emitted, airuntime.EventDone)
	assertEventTypeAbsent(t, emitted, airuntime.EventChainNodeOpen)
	assertEventTypeAbsent(t, emitted, airuntime.EventFinalAnswerStart)

	if state.Status != airuntime.ExecutionStatusCompleted {
		t.Fatalf("expected completed state, got %q", state.Status)
	}
	assertLifecycleEvent(t, observer.records, "chain_started")
	assertLifecycleEvent(t, observer.records, "chain_completed")
}

func TestOrchestratorApprovalPauseAndResumeEmitLifecycleCallbacks(t *testing.T) {
	ctx := context.Background()
	observer := &lifecycleObserver{}
	unregister := aiobs.RegisterObserver(observer)
	defer unregister()

	executions := airuntime.NewExecutionStore(nil, "")
	checkpoints := airuntime.NewCheckpointStore(nil, "")
	orchestrator := &Orchestrator{
		executions:  executions,
		checkpoints: checkpoints,
		converter:   airuntime.NewSSEConverter(),
	}
	state := airuntime.ExecutionState{
		TraceID:      "trace-2",
		SessionID:    "sess-2",
		PlanID:       "plan-2",
		TurnID:       "turn-2",
		Scene:        "deployment:hosts",
		Status:       airuntime.ExecutionStatusRunning,
		Phase:        "running",
		CheckpointID: "checkpoint-2",
		Steps:        map[string]airuntime.StepState{},
	}

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(&adk.AgentEvent{
		Action: &adk.AgentAction{
			Interrupted: &adk.InterruptInfo{
				InterruptContexts: []*adk.InterruptCtx{{
					ID: "step-approval",
					Info: map[string]any{
						"step_id":           "step-approval",
						"tool_name":         "host_delete",
						"tool_display_name": "删除主机",
						"mode":              "mutating",
						"risk_level":        "high",
						"summary":           "删除主机前需要审批",
						"params":            map[string]any{"id": 7},
						"arguments_json":    `{"id":7}`,
					},
				}},
			},
		},
	})
	gen.Close()

	var pauseEvents []airuntime.StreamEvent
	resumeState, err := orchestrator.streamExecution(ctx, iter, &state, func(event airuntime.StreamEvent) bool {
		pauseEvents = append(pauseEvents, event)
		return true
	})
	if err != nil {
		t.Fatalf("streamExecution returned error: %v", err)
	}
	if resumeState == nil || !resumeState.Interrupted {
		t.Fatalf("expected interrupted resume result, got %#v", resumeState)
	}
	assertEventTypePresent(t, pauseEvents, airuntime.EventChainPaused)
	assertLifecycleEvent(t, observer.records, "chain_paused")
	assertLifecycleEvent(t, observer.records, "approval_pending")

	saved, ok, err := executions.Load(ctx, "sess-2", "plan-2")
	if err != nil || !ok {
		t.Fatalf("expected persisted execution state, got ok=%v err=%v", ok, err)
	}
	if saved.PendingApproval == nil || saved.PendingApproval.ArgumentsInJSON != `{"id":7}` {
		t.Fatalf("expected pending approval arguments, got %#v", saved.PendingApproval)
	}

	var resumeEvents []airuntime.StreamEvent
	result, err := orchestrator.resume(ctx, airuntime.ResumeRequest{
		SessionID: "sess-2",
		PlanID:    "plan-2",
		StepID:    "step-approval",
		Approved:  true,
		Reason:    "looks safe",
	}, func(event airuntime.StreamEvent) bool {
		resumeEvents = append(resumeEvents, event)
		return true
	})
	if err != nil {
		t.Fatalf("resume returned error: %v", err)
	}
	if result == nil || result.Status != string(airuntime.ExecutionStatusCompleted) {
		t.Fatalf("expected completed resume result, got %#v", result)
	}

	assertEventTypePresent(t, resumeEvents, airuntime.EventChainMeta)
	assertEventTypePresent(t, resumeEvents, airuntime.EventChainResumed)
	assertEventTypePresent(t, resumeEvents, airuntime.EventDone)
	assertLifecycleEvent(t, observer.records, "approval_resolved")
	assertLifecycleEvent(t, observer.records, "chain_resumed")
}

func TestOrchestratorRun_ReturnsInitializationErrorWhenRunnerUnavailable(t *testing.T) {
	orchestrator := &Orchestrator{
		initErr: errors.New("llm disabled"),
	}

	err := orchestrator.Run(context.Background(), airuntime.RunRequest{
		Message: "check cluster status",
	}, nil)
	if err == nil {
		t.Fatal("expected initialization error")
	}
	if got := err.Error(); got != "orchestrator unavailable: llm disabled" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func assistantEvent(agentName, content string) *adk.AgentEvent {
	return &adk.AgentEvent{
		AgentName: agentName,
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				Message: schema.AssistantMessage(content, nil),
				Role:    schema.Assistant,
			},
		},
	}
}

func toolEvent(toolName, content string) *adk.AgentEvent {
	return &adk.AgentEvent{
		AgentName: "agent",
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				Message:  schema.ToolMessage(content, "call-1", schema.WithToolName(toolName)),
				Role:     schema.Tool,
				ToolName: toolName,
			},
		},
	}
}

func assertEventTypePresent(t *testing.T, events []airuntime.StreamEvent, want airuntime.EventType) {
	t.Helper()
	for _, event := range events {
		if event.Type == want {
			return
		}
	}
	t.Fatalf("expected stream event %q in %#v", want, events)
}

func assertEventTypeAbsent(t *testing.T, events []airuntime.StreamEvent, forbidden airuntime.EventType) {
	t.Helper()
	for _, event := range events {
		if event.Type == forbidden {
			t.Fatalf("did not expect stream event %q in %#v", forbidden, events)
		}
	}
}

func assertLifecycleEvent(t *testing.T, records []aiobs.ThoughtChainLifecycleRecord, want string) {
	t.Helper()
	for _, record := range records {
		if record.Event == want {
			return
		}
	}
	t.Fatalf("expected lifecycle event %q in %#v", want, records)
}
