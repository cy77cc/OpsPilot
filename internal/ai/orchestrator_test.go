package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/cy77cc/OpsPilot/internal/ai/events"
	"github.com/cy77cc/OpsPilot/internal/ai/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/ai/state"
	"github.com/cy77cc/OpsPilot/internal/ai/summarizer"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

type orchestratorStubStepRunner struct {
	result executor.StepResult
	err    error
	calls  int
}

func (s *orchestratorStubStepRunner) RunStep(_ context.Context, _ executor.Request, step planner.PlanStep) (executor.StepResult, error) {
	s.calls++
	if s.err != nil {
		return executor.StepResult{}, s.err
	}
	out := s.result
	if out.StepID == "" {
		out.StepID = step.StepID
	}
	if out.Summary == "" {
		out.Summary = "expert step completed"
	}
	return out, nil
}

func newExecutionStoreForOrchestrator(t *testing.T) *runtime.ExecutionStore {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	return runtime.NewExecutionStore(client, "ai:test:execution:")
}

func newSessionStateForOrchestrator(t *testing.T) *state.SessionState {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	return state.NewSessionState(client, "ai:test:session:")
}

func TestResumeReturnsIdempotentStatus(t *testing.T) {
	store := newExecutionStoreForOrchestrator(t)
	exec := executor.New(store)
	ctx := context.Background()

	_, err := exec.Run(ctx, executor.Request{
		TraceID:   "trace-2",
		SessionID: "session-2",
		Message:   "deploy payment-api",
		Plan: planner.ExecutionPlan{
			PlanID: "plan-2",
			Goal:   "deploy payment-api",
			Steps: []planner.PlanStep{
				{
					StepID: "step-2",
					Title:  "发布服务",
					Expert: "service",
					Mode:   "mutating",
					Risk:   "high",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	orch := NewOrchestrator(nil, store, common.PlatformDeps{})
	first, err := orch.Resume(ctx, ResumeRequest{
		SessionID: "session-2",
		PlanID:    "plan-2",
		StepID:    "step-2",
		Approved:  true,
	})
	if err != nil {
		t.Fatalf("first Resume() error = %v", err)
	}
	if first.Status == "idempotent" {
		t.Fatalf("first resume unexpectedly idempotent")
	}

	second, err := orch.Resume(ctx, ResumeRequest{
		SessionID: "session-2",
		PlanID:    "plan-2",
		StepID:    "step-2",
		Approved:  true,
	})
	if err != nil {
		t.Fatalf("second Resume() error = %v", err)
	}
	if second.Status != "idempotent" {
		t.Fatalf("second resume status = %s, want idempotent", second.Status)
	}
}

func TestResumeDoesNotPanicWhenPendingApprovalAlreadyCleared(t *testing.T) {
	store := newExecutionStoreForOrchestrator(t)
	ctx := context.Background()
	if err := store.Save(ctx, runtime.ExecutionState{
		SessionID: "session-3",
		PlanID:    "plan-3",
		Status:    runtime.ExecutionStatusCompleted,
		Phase:     "executor_completed",
		Steps: map[string]runtime.StepState{
			"step-3": {
				StepID: "step-3",
				Status: runtime.StepCompleted,
			},
		},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	orch := NewOrchestrator(nil, store, common.PlatformDeps{})
	res, err := orch.Resume(ctx, ResumeRequest{
		SessionID: "session-3",
		PlanID:    "plan-3",
		StepID:    "step-3",
		Approved:  true,
	})
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if res == nil {
		t.Fatalf("Resume() returned nil result")
	}
	if res.StepID != "step-3" {
		t.Fatalf("StepID = %q, want step-3", res.StepID)
	}
}

func TestResumeRejectReturnsRejectedMessage(t *testing.T) {
	store := newExecutionStoreForOrchestrator(t)
	exec := executor.New(store)
	ctx := context.Background()

	_, err := exec.Run(ctx, executor.Request{
		TraceID:   "trace-4",
		SessionID: "session-4",
		Message:   "deploy payment-api",
		Plan: planner.ExecutionPlan{
			PlanID: "plan-4",
			Goal:   "deploy payment-api",
			Steps: []planner.PlanStep{
				{
					StepID: "step-4",
					Title:  "发布服务",
					Expert: "service",
					Mode:   "mutating",
					Risk:   "high",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	orch := NewOrchestrator(nil, store, common.PlatformDeps{})
	res, err := orch.Resume(ctx, ResumeRequest{
		SessionID: "session-4",
		PlanID:    "plan-4",
		StepID:    "step-4",
		Approved:  false,
	})
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if res.Status != "rejected" {
		t.Fatalf("status = %q, want rejected", res.Status)
	}
	if res.Message != "审批已拒绝，待审批步骤不会执行，相关下游步骤已被取消或阻断。" {
		t.Fatalf("message = %q", res.Message)
	}
}

func TestOrchestratorRunMainFlowEmitsExpectedEvents(t *testing.T) {
	sessions := newSessionStateForOrchestrator(t)
	store := newExecutionStoreForOrchestrator(t)
	runner := &orchestratorStubStepRunner{result: executor.StepResult{
		Summary: "service status collected",
		Evidence: []executor.Evidence{
			{Kind: "tool_result", Source: "service"},
		},
	}}
	orch := &Orchestrator{
		sessions:   sessions,
		executions: store,
		rewriter:   rewrite.New(nil),
		planner:    planner.New(nil),
		executor:   executor.New(store, executor.WithStepRunner(runner)),
		summarizer: summarizer.New(nil),
		renderer:   newFinalAnswerRenderer(),
		maxIters:   2,
	}

	var names []string
	var stages []string
	var deltas []string
	err := orch.Run(context.Background(), RunRequest{
		Message: "查看 payment-api 的状态",
		RuntimeContext: RuntimeContext{
			Scene: "scene:service",
			SelectedResources: []SelectedResource{
				{Type: "service", ID: "42", Name: "payment-api"},
			},
		},
	}, func(evt StreamEvent) bool {
		names = append(names, string(evt.Type))
		if evt.Type == events.StageDelta {
			stages = append(stages, strings.TrimSpace(stringValue(evt.Data["stage"])))
		}
		if evt.Type == events.Delta {
			deltas = append(deltas, stringValue(evt.Data["content_chunk"]))
		}
		return true
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertEventOrder(t, names, []string{
		"meta",
		"rewrite_result",
		"planner_state",
		"plan_created",
		"step_update",
		"summary",
		"done",
	})
	assertStageSeen(t, stages, "rewrite")
	assertStageSeen(t, stages, "plan")
	assertStageSeen(t, stages, "execute")
	assertStageSeen(t, stages, "summary")
	if len(deltas) == 0 {
		t.Fatalf("delta events = %v, want streamed final answer", deltas)
	}
	if deltaIndex, doneIndex := firstEventIndex(names, "delta"), firstEventIndex(names, "done"); deltaIndex < 0 || doneIndex < 0 || deltaIndex > doneIndex {
		t.Fatalf("events = %v, want delta before done", names)
	}
}

func TestOrchestratorPlanAndReplyClarifyFlow(t *testing.T) {
	orch := &Orchestrator{
		planner:  planner.New(nil),
		maxIters: 2,
	}
	meta := events.EventMeta{
		SessionID: "session-clarify",
		TraceID:   "trace-clarify",
	}
	var names []string
	var content strings.Builder
	reply, err := orch.planAndReply(context.Background(), "帮我看看状态", rewrite.Output{
		AmbiguityFlags: []string{"resource_target_not_explicit"},
	}, RuntimeContext{}, meta, func(evt StreamEvent) bool {
		names = append(names, string(evt.Type))
		if evt.Type == events.Delta {
			content.WriteString(stringValue(evt.Data["content_chunk"]))
		}
		return true
	}, "session-clarify")
	if err != nil {
		t.Fatalf("planAndReply() error = %v", err)
	}
	if !strings.Contains(reply, "确认") && !strings.Contains(reply, "明确") && !strings.Contains(reply, "补充") {
		t.Fatalf("reply = %q, want clarify content", reply)
	}
	assertEventOrder(t, names, []string{"planner_state", "clarify_required", "delta"})
}

func TestOrchestratorApprovalResumeFlow(t *testing.T) {
	store := newExecutionStoreForOrchestrator(t)
	runner := &orchestratorStubStepRunner{result: executor.StepResult{Summary: "deployment completed"}}
	orch := &Orchestrator{
		executions: store,
		planner:    planner.New(nil),
		executor:   executor.New(store, executor.WithStepRunner(runner)),
		maxIters:   2,
	}
	meta := events.EventMeta{
		SessionID: "session-approval",
		TraceID:   "trace-approval",
	}

	var names []string
	reply, err := orch.planAndReply(context.Background(), "发布 payment-api 到 prod", rewrite.Output{
		NormalizedGoal: "发布 payment-api 到 prod",
		OperationMode:  "mutate",
		ResourceHints: rewrite.ResourceHints{
			ServiceName: "payment-api",
			ServiceID:   42,
			ClusterName: "prod",
			ClusterID:   9,
		},
		NormalizedRequest: rewrite.NormalizedRequest{
			Targets: []rewrite.RequestTarget{{Type: "service", Name: "payment-api"}},
		},
	}, RuntimeContext{}, meta, func(evt StreamEvent) bool {
		names = append(names, string(evt.Type))
		return true
	}, "session-approval")
	if err != nil {
		t.Fatalf("planAndReply() error = %v", err)
	}
	if !containsEvent(names, "approval_required") {
		t.Fatalf("events = %v, want approval_required", names)
	}
	if reply == "" {
		t.Fatalf("reply is empty")
	}

	res, err := orch.Resume(context.Background(), ResumeRequest{
		SessionID: "session-approval",
		StepID:    "step-1",
		Approved:  true,
	})
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if res == nil || res.Status == "" {
		t.Fatalf("Resume() returned invalid result: %#v", res)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
}

func TestWaitingApprovalStateSurvivesOrchestratorRestart(t *testing.T) {
	store := newExecutionStoreForOrchestrator(t)
	orch := &Orchestrator{
		executions: store,
		planner:    planner.New(nil),
		executor:   executor.New(store, executor.WithStepRunner(&orchestratorStubStepRunner{result: executor.StepResult{Summary: "deployment completed"}})),
		maxIters:   2,
	}
	meta := events.EventMeta{
		SessionID: "session-restart",
		TraceID:   "trace-restart",
	}

	_, err := orch.planAndReply(context.Background(), "发布 payment-api 到 prod", rewrite.Output{
		NormalizedGoal: "发布 payment-api 到 prod",
		OperationMode:  "mutate",
		ResourceHints: rewrite.ResourceHints{
			ServiceName: "payment-api",
			ServiceID:   42,
			ClusterName: "prod",
			ClusterID:   9,
		},
		NormalizedRequest: rewrite.NormalizedRequest{
			Targets: []rewrite.RequestTarget{{Type: "service", Name: "payment-api"}},
		},
	}, RuntimeContext{}, meta, func(StreamEvent) bool { return true }, "session-restart")
	if err != nil {
		t.Fatalf("planAndReply() error = %v", err)
	}

	st, err := store.Load(context.Background(), "session-restart")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if st == nil || st.PendingApproval == nil {
		t.Fatalf("expected persisted waiting approval state, got %#v", st)
	}
	if st.Status != runtime.ExecutionStatusWaitingApproval {
		t.Fatalf("status = %s, want %s", st.Status, runtime.ExecutionStatusWaitingApproval)
	}

	restarted := &Orchestrator{
		executions: store,
		executor:   executor.New(store, executor.WithStepRunner(&orchestratorStubStepRunner{result: executor.StepResult{Summary: "deployment completed"}})),
		maxIters:   2,
	}
	res, err := restarted.Resume(context.Background(), ResumeRequest{
		SessionID: "session-restart",
		StepID:    "step-1",
		Approved:  false,
	})
	if err != nil {
		t.Fatalf("Resume() error after restart = %v", err)
	}
	if res == nil || res.Status != "rejected" {
		t.Fatalf("Resume() status = %#v, want rejected", res)
	}
}

func TestOrchestratorEmitsReplanStartedWhenSummaryNeedsMoreInvestigation(t *testing.T) {
	sessions := newSessionStateForOrchestrator(t)
	store := newExecutionStoreForOrchestrator(t)
	runner := &orchestratorStubStepRunner{result: executor.StepResult{
		Summary: "service status collected without evidence",
	}}
	orch := &Orchestrator{
		sessions:   sessions,
		executions: store,
		rewriter:   rewrite.New(nil),
		planner:    planner.New(nil),
		executor:   executor.New(store, executor.WithStepRunner(runner)),
		summarizer: summarizer.New(nil),
		renderer:   newFinalAnswerRenderer(),
		maxIters:   2,
	}

	var names []string
	var replanReason string
	err := orch.Run(context.Background(), RunRequest{
		Message: "查看 payment-api 的状态",
		RuntimeContext: RuntimeContext{
			Scene: "scene:service",
			SelectedResources: []SelectedResource{
				{Type: "service", ID: "42", Name: "payment-api"},
			},
		},
	}, func(evt StreamEvent) bool {
		names = append(names, string(evt.Type))
		if evt.Type == events.ReplanStarted {
			replanReason = stringValue(evt.Data["reason"])
		}
		return true
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !containsEvent(names, "replan_started") {
		t.Fatalf("events = %v, want replan_started", names)
	}
	if replanReason == "" {
		t.Fatalf("replan reason is empty")
	}
}

func assertEventOrder(t *testing.T, have []string, wantInOrder []string) {
	t.Helper()
	index := 0
	for _, name := range have {
		if index < len(wantInOrder) && name == wantInOrder[index] {
			index++
		}
	}
	if index != len(wantInOrder) {
		t.Fatalf("events = %v, want ordered subsequence %v", have, wantInOrder)
	}
}

func assertStageSeen(t *testing.T, stages []string, want string) {
	t.Helper()
	for _, stage := range stages {
		if stage == want {
			return
		}
	}
	t.Fatalf("stages = %v, want %s", stages, want)
}

func containsEvent(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func firstEventIndex(items []string, target string) int {
	for i, item := range items {
		if item == target {
			return i
		}
	}
	return -1
}
