package ai

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/cy77cc/OpsPilot/internal/ai/events"
	"github.com/cy77cc/OpsPilot/internal/ai/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/ai/summarizer"
	appconfig "github.com/cy77cc/OpsPilot/internal/config"
)

type stubOrchestratorStepRunner struct {
	result executor.StepResult
}

func (s stubOrchestratorStepRunner) RunStep(_ context.Context, _ executor.Request, step planner.PlanStep) (executor.StepResult, error) {
	out := s.result
	if out.StepID == "" {
		out.StepID = step.StepID
	}
	if out.Summary == "" {
		out.Summary = "step completed"
	}
	return out, nil
}

func newOrchestratorExecutionStore(t *testing.T) *runtime.ExecutionStore {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	return runtime.NewExecutionStore(client, "ai:test:orchestrator:")
}

func enableTurnBlockStreamingForTest(t *testing.T) {
	t.Helper()
	previous := appconfig.CFG
	t.Cleanup(func() {
		appconfig.CFG = previous
	})
	appconfig.CFG.AI.UseTurnBlockStreaming = true
}

func TestOrchestratorRunEmitsNativeAndCompatibilityStreamEvents(t *testing.T) {
	enableTurnBlockStreamingForTest(t)
	store := newOrchestratorExecutionStore(t)
	eventsSeen := make([]StreamEvent, 0, 16)
	emit := func(evt StreamEvent) bool {
		eventsSeen = append(eventsSeen, evt)
		return true
	}

	orch := &Orchestrator{
		executions: store,
		rewriter: rewrite.NewWithFunc(func(_ context.Context, _ rewrite.Input, onDelta func(string)) (rewrite.Output, error) {
			onDelta("rewrite")
			return rewrite.Output{Narrative: "rewrite ok"}, nil
		}),
		planner: planner.NewWithFunc(func(_ context.Context, _ planner.Input, onDelta func(string)) (planner.Decision, error) {
			onDelta("plan")
			return planner.Decision{Type: planner.DecisionPlan, Narrative: "plan ok", Plan: &planner.ExecutionPlan{PlanID: "plan-1", Narrative: "plan ok", Steps: []planner.PlanStep{{StepID: "step-1", Title: "检查服务", Expert: "service", Mode: "readonly", Risk: "low"}}}}, nil
		}),
		executor: executor.New(store, executor.WithStepRunner(stubOrchestratorStepRunner{result: executor.StepResult{Summary: "service expert completed"}})),
		summarizer: summarizer.NewWithFunc(func(_ context.Context, _ summarizer.Input, onThinkingDelta func(string), onAnswerDelta func(string)) (string, error) {
			onThinkingDelta("thinking")
			onAnswerDelta("answer")
			return "answer", nil
		}),
		heartbeatInterval: 0,
	}

	if err := orch.Run(t.Context(), RunRequest{Message: "inspect service"}, emit); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertContainsEvent(t, eventsSeen, events.Meta)
	assertContainsEvent(t, eventsSeen, events.TurnStarted)
	assertContainsEvent(t, eventsSeen, events.BlockOpen)
	assertContainsEvent(t, eventsSeen, events.BlockDelta)
	assertContainsEvent(t, eventsSeen, events.PlanCreated)
	assertContainsEvent(t, eventsSeen, events.ToolCall)
	assertContainsEvent(t, eventsSeen, events.ToolResult)
	assertContainsEvent(t, eventsSeen, events.ThinkingDelta)
	assertContainsEvent(t, eventsSeen, events.Delta)
	assertContainsEvent(t, eventsSeen, events.TurnDone)
	assertContainsEvent(t, eventsSeen, events.Done)

	metaEvt := findEvent(eventsSeen, events.Meta)
	if metaEvt == nil || stringValue(metaEvt.Data["turn_id"]) == "" {
		t.Fatalf("meta event missing turn_id: %#v", metaEvt)
	}
}

func TestOrchestratorRunEmitsApprovalRequiredStreamState(t *testing.T) {
	enableTurnBlockStreamingForTest(t)
	store := newOrchestratorExecutionStore(t)
	eventsSeen := make([]StreamEvent, 0, 16)
	emit := func(evt StreamEvent) bool {
		eventsSeen = append(eventsSeen, evt)
		return true
	}

	orch := &Orchestrator{
		executions: store,
		rewriter: rewrite.NewWithFunc(func(_ context.Context, _ rewrite.Input, onDelta func(string)) (rewrite.Output, error) {
			onDelta("rewrite")
			return rewrite.Output{Narrative: "rewrite ok"}, nil
		}),
		planner: planner.NewWithFunc(func(_ context.Context, _ planner.Input, onDelta func(string)) (planner.Decision, error) {
			onDelta("plan")
			return planner.Decision{Type: planner.DecisionPlan, Narrative: "plan ok", Plan: &planner.ExecutionPlan{PlanID: "plan-approval", Narrative: "plan ok", Steps: []planner.PlanStep{{StepID: "step-1", Title: "重启服务", Expert: "hostops", Mode: "mutating", Risk: "high"}}}}, nil
		}),
		executor: executor.New(store, executor.WithStepRunner(stubOrchestratorStepRunner{})),
		summarizer: summarizer.NewWithFunc(func(_ context.Context, _ summarizer.Input, onThinkingDelta func(string), onAnswerDelta func(string)) (string, error) {
			onThinkingDelta("thinking")
			onAnswerDelta("answer")
			return "answer", nil
		}),
		heartbeatInterval: 0,
	}

	if err := orch.Run(t.Context(), RunRequest{Message: "restart service"}, emit); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertContainsEvent(t, eventsSeen, events.ApprovalRequired)
	if findEvent(eventsSeen, events.ToolCall) != nil {
		t.Fatalf("approval-gated flow should not emit tool_call before approval: %#v", eventTypes(eventsSeen))
	}
	if !containsTurnState(eventsSeen, "waiting_user") {
		t.Fatalf("expected waiting_user turn_state in stream: %#v", eventTypes(eventsSeen))
	}
	if findEvent(eventsSeen, events.ThinkingDelta) != nil {
		t.Fatalf("approval flow should not emit thinking deltas before approval: %#v", eventTypes(eventsSeen))
	}
	if findEvent(eventsSeen, events.Summary) != nil {
		t.Fatalf("approval flow should not emit summary before approval: %#v", eventTypes(eventsSeen))
	}
	if !containsEventData(eventsSeen, events.Delta, "content_chunk", "此步骤需要你确认后继续执行。") {
		t.Fatalf("approval flow should emit waiting summary delta: %#v", eventsSeen)
	}
}

func assertContainsEvent(t *testing.T, seen []StreamEvent, want events.Name) {
	t.Helper()
	if findEvent(seen, want) == nil {
		t.Fatalf("missing event %s in %v", want, eventTypes(seen))
	}
}

func findEvent(seen []StreamEvent, want events.Name) *StreamEvent {
	for i := range seen {
		if seen[i].Type == want {
			return &seen[i]
		}
	}
	return nil
}

func containsTurnState(seen []StreamEvent, status string) bool {
	for _, evt := range seen {
		if evt.Type != events.TurnState {
			continue
		}
		if stringValue(evt.Data["status"]) == status {
			return true
		}
	}
	return false
}

func eventTypes(seen []StreamEvent) []events.Name {
	out := make([]events.Name, 0, len(seen))
	for _, evt := range seen {
		out = append(out, evt.Type)
	}
	return out
}

func containsEventData(seen []StreamEvent, want events.Name, key, value string) bool {
	for _, evt := range seen {
		if evt.Type != want {
			continue
		}
		if stringValue(evt.Data[key]) == value {
			return true
		}
	}
	return false
}
