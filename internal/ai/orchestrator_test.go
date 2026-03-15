package ai

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	aiobs "github.com/cy77cc/OpsPilot/internal/ai/observability"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

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
