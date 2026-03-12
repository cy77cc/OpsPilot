package ai

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/cy77cc/OpsPilot/internal/ai/events"
	"github.com/cy77cc/OpsPilot/internal/ai/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/ai/summarizer"
	"github.com/cy77cc/OpsPilot/internal/config"
)

type resumeStreamStepRunner struct{}

func (resumeStreamStepRunner) RunStep(_ context.Context, _ executor.Request, step planner.PlanStep) (executor.StepResult, error) {
	return executor.StepResult{
		StepID:  step.StepID,
		Status:  runtime.StepCompleted,
		Summary: "service expert completed deployment",
	}, nil
}

func TestOrchestratorResumeStreamEmitsTurnLinkedContinuation(t *testing.T) {
	prevAI := config.CFG.AI
	config.CFG.AI.UseTurnBlockStreaming = true
	t.Cleanup(func() {
		config.CFG.AI = prevAI
	})

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	store := runtime.NewExecutionStore(client, "ai:test:resume:")
	exec := executor.New(store, executor.WithStepRunner(resumeStreamStepRunner{}))

	_, err := exec.Run(context.Background(), executor.Request{
		TraceID:   "trace-1",
		SessionID: "session-1",
		Message:   "deploy payment-api",
		EventMeta: events.EventMeta{TurnID: "turn-1"},
		Plan: planner.ExecutionPlan{
			PlanID: "plan-1",
			Goal:   "deploy payment-api",
			Steps: []planner.PlanStep{
				{
					StepID: "step-1",
					Title:  "发布服务",
					Expert: "service",
					Mode:   "mutating",
					Risk:   "high",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("prepare approval state: %v", err)
	}

	orch := &Orchestrator{
		executions: store,
		executor:   exec,
		summarizer: summarizer.NewWithFunc(func(_ context.Context, _ summarizer.Input, onThinkingDelta func(string), onAnswerDelta func(string)) (string, error) {
			onThinkingDelta("resume thinking")
			onAnswerDelta("resume answer")
			return "resume answer", nil
		}),
	}
	var streamed []StreamEvent
	res, err := orch.ResumeStream(context.Background(), ResumeRequest{
		SessionID: "session-1",
		PlanID:    "plan-1",
		StepID:    "step-1",
		Approved:  true,
	}, func(evt StreamEvent) bool {
		streamed = append(streamed, evt)
		return true
	})
	if err != nil {
		t.Fatalf("ResumeStream() error = %v", err)
	}
	if res == nil {
		t.Fatal("ResumeStream() returned nil result")
	}
	if res.TurnID != "turn-1" {
		t.Fatalf("res.TurnID = %q, want turn-1", res.TurnID)
	}

	assertEventSeen(t, streamed, events.Meta)
	assertEventSeen(t, streamed, events.TurnStarted)
	assertEventSeen(t, streamed, events.StepUpdate)
	assertEventSeen(t, streamed, events.ToolCall)
	assertEventSeen(t, streamed, events.ToolResult)
	assertEventSeen(t, streamed, events.ThinkingDelta)
	assertEventSeen(t, streamed, events.Summary)
	assertEventSeen(t, streamed, events.Done)
	assertEventSeen(t, streamed, events.TurnDone)

	for _, evt := range streamed {
		if evt.Type == events.Meta || evt.Type == events.StepUpdate || evt.Type == events.ToolCall || evt.Type == events.ToolResult || evt.Type == events.Done {
			if got := stringValue(evt.Data["turn_id"]); got != "turn-1" {
				t.Fatalf("%s turn_id = %q, want turn-1", evt.Type, got)
			}
		}
	}
}

func TestOrchestratorResumeStreamRejectsWithoutSummary(t *testing.T) {
	prevAI := config.CFG.AI
	config.CFG.AI.UseTurnBlockStreaming = true
	t.Cleanup(func() {
		config.CFG.AI = prevAI
	})

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	store := runtime.NewExecutionStore(client, "ai:test:resume:")
	exec := executor.New(store, executor.WithStepRunner(resumeStreamStepRunner{}))

	_, err := exec.Run(context.Background(), executor.Request{
		TraceID:   "trace-reject",
		SessionID: "session-reject",
		Message:   "deploy payment-api",
		EventMeta: events.EventMeta{TurnID: "turn-reject"},
		Plan: planner.ExecutionPlan{
			PlanID: "plan-reject",
			Goal:   "deploy payment-api",
			Steps: []planner.PlanStep{
				{
					StepID: "step-1",
					Title:  "发布服务",
					Expert: "service",
					Mode:   "mutating",
					Risk:   "high",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("prepare approval state: %v", err)
	}

	orch := &Orchestrator{
		executions: store,
		executor:   exec,
		summarizer: summarizer.NewWithFunc(func(_ context.Context, _ summarizer.Input, onThinkingDelta func(string), onAnswerDelta func(string)) (string, error) {
			onThinkingDelta("should not run")
			onAnswerDelta("should not run")
			return "should not run", nil
		}),
	}
	var streamed []StreamEvent
	res, err := orch.ResumeStream(context.Background(), ResumeRequest{
		SessionID: "session-reject",
		PlanID:    "plan-reject",
		StepID:    "step-1",
		Approved:  false,
	}, func(evt StreamEvent) bool {
		streamed = append(streamed, evt)
		return true
	})
	if err != nil {
		t.Fatalf("ResumeStream() error = %v", err)
	}
	if res == nil {
		t.Fatal("ResumeStream() returned nil result")
	}
	if res.TurnID != "turn-reject" {
		t.Fatalf("res.TurnID = %q, want turn-reject", res.TurnID)
	}
	assertEventSeen(t, streamed, events.Meta)
	assertEventSeen(t, streamed, events.StepUpdate)
	assertEventSeen(t, streamed, events.Done)
	assertEventSeen(t, streamed, events.TurnDone)
	if assertHasEvent(streamed, events.ThinkingDelta) || assertHasEvent(streamed, events.Summary) {
		t.Fatalf("rejected resume should not emit summary continuation: %#v", eventTypes(streamed))
	}
}

func assertHasEvent(seen []StreamEvent, want events.Name) bool {
	return findEvent(seen, want) != nil
}
