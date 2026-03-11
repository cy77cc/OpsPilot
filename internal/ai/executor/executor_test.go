package executor

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

type stubStepRunner struct {
	result StepResult
	err    error
	calls  int
}

func (s *stubStepRunner) RunStep(_ context.Context, _ Request, step planner.PlanStep) (StepResult, error) {
	s.calls++
	if s.err != nil {
		return StepResult{}, s.err
	}
	out := s.result
	if out.StepID == "" {
		out.StepID = step.StepID
	}
	if out.Summary == "" {
		out.Summary = fmt.Sprintf("expert %s executed step %s", step.Expert, step.StepID)
	}
	return out, nil
}

func newExecutionStore(t *testing.T) *runtime.ExecutionStore {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	return runtime.NewExecutionStore(client, "ai:test:execution:")
}

func TestExecutorApprovalResumeFlow(t *testing.T) {
	store := newExecutionStore(t)
	runner := &stubStepRunner{result: StepResult{Summary: "service expert completed deployment"}}
	exec := New(store, WithStepRunner(runner))
	ctx := context.Background()

	result, err := exec.Run(ctx, Request{
		TraceID:   "trace-1",
		SessionID: "session-1",
		Message:   "deploy payment-api",
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
		t.Fatalf("Run() error = %v", err)
	}
	if result.State.PendingApproval == nil {
		t.Fatalf("expected pending approval")
	}
	if got := result.State.Status; got != runtime.ExecutionStatusWaitingApproval {
		t.Fatalf("state status = %s, want %s", got, runtime.ExecutionStatusWaitingApproval)
	}

	resumed, err := exec.Resume(ctx, ResumeRequest{
		SessionID: "session-1",
		PlanID:    "plan-1",
		StepID:    "step-1",
		Approved:  true,
	})
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if got := resumed.State.Status; got != runtime.ExecutionStatusCompleted {
		t.Fatalf("resumed state status = %s, want %s", got, runtime.ExecutionStatusCompleted)
	}
	if got := resumed.State.Steps["step-1"].Status; got != runtime.StepCompleted {
		t.Fatalf("step status = %s, want %s", got, runtime.StepCompleted)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
}

func TestExecutorReadonlyStepRequiresExpertRunner(t *testing.T) {
	store := newExecutionStore(t)
	exec := New(store)
	ctx := context.Background()

	result, err := exec.Run(ctx, Request{
		TraceID:   "trace-2",
		SessionID: "session-2",
		Message:   "inspect payment-api",
		Plan: planner.ExecutionPlan{
			PlanID: "plan-2",
			Goal:   "inspect payment-api",
			Steps: []planner.PlanStep{
				{
					StepID: "step-1",
					Title:  "检查服务状态",
					Expert: "service",
					Mode:   "readonly",
					Risk:   "low",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := result.State.Steps["step-1"].Status; got != runtime.StepFailed {
		t.Fatalf("step status = %s, want %s", got, runtime.StepFailed)
	}
	if got := result.State.Steps["step-1"].ErrorCode; got != "expert_tool_stream_failed" {
		t.Fatalf("error code = %q, want %q", got, "expert_tool_stream_failed")
	}
}

func TestExecutorRejectApprovalUsesCancelledAndBlockedSummaries(t *testing.T) {
	store := newExecutionStore(t)
	exec := New(store, WithStepRunner(&stubStepRunner{}))
	ctx := context.Background()

	result, err := exec.Run(ctx, Request{
		TraceID:   "trace-reject",
		SessionID: "session-reject",
		Message:   "deploy payment-api",
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
				{
					StepID:    "step-2",
					Title:     "验证结果",
					Expert:    "observability",
					Mode:      "readonly",
					Risk:      "low",
					DependsOn: []string{"step-1"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.State.PendingApproval == nil {
		t.Fatalf("expected pending approval")
	}

	rejected, err := exec.Resume(ctx, ResumeRequest{
		SessionID: "session-reject",
		PlanID:    "plan-reject",
		StepID:    "step-1",
		Approved:  false,
	})
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if got := rejected.State.Steps["step-1"].Status; got != runtime.StepCancelled {
		t.Fatalf("step-1 status = %s, want %s", got, runtime.StepCancelled)
	}
	if got := rejected.State.Steps["step-1"].UserVisibleSummary; got != "审批已拒绝，当前步骤不会执行。" {
		t.Fatalf("step-1 summary = %q", got)
	}
	if got := rejected.State.Steps["step-2"].Status; got != runtime.StepBlocked {
		t.Fatalf("step-2 status = %s, want %s", got, runtime.StepBlocked)
	}
	if got := rejected.State.Steps["step-2"].UserVisibleSummary; got != "上游步骤已取消，当前步骤不会继续执行。" {
		t.Fatalf("step-2 summary = %q", got)
	}
}

func TestExecutorInvokesExpertRunnerForReadonlyStep(t *testing.T) {
	store := newExecutionStore(t)
	runner := &stubStepRunner{
		result: StepResult{
			Summary: "service expert inspected runtime state",
			Evidence: []Evidence{
				{Kind: "expert_result", Source: "service"},
			},
		},
	}
	exec := New(store, WithStepRunner(runner))
	ctx := context.Background()

	result, err := exec.Run(ctx, Request{
		TraceID:   "trace-3",
		SessionID: "session-3",
		Message:   "inspect payment-api",
		Plan: planner.ExecutionPlan{
			PlanID: "plan-3",
			Goal:   "inspect payment-api",
			Steps: []planner.PlanStep{
				{
					StepID: "step-1",
					Title:  "检查服务状态",
					Expert: "service",
					Intent: "inspect_service",
					Task:   "inspect payment-api",
					Mode:   "readonly",
					Risk:   "low",
					Input: map[string]any{
						"service_id": 42,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
	if got := result.State.Steps["step-1"].Status; got != runtime.StepCompleted {
		t.Fatalf("step status = %s, want %s", got, runtime.StepCompleted)
	}
	foundEvidence := false
	for _, step := range result.Steps {
		if step.StepID == "step-1" && len(step.Evidence) > 0 {
			foundEvidence = true
		}
	}
	if !foundEvidence {
		t.Fatalf("expected expert evidence in step results")
	}
}

func TestExecutorEmitsRealtimeStepAndToolEvents(t *testing.T) {
	store := newExecutionStore(t)
	runner := &stubStepRunner{result: StepResult{Summary: "k8s expert fetched pod logs"}}
	exec := New(store, WithStepRunner(runner))
	ctx := context.Background()

	var names []string
	result, err := exec.Run(ctx, Request{
		TraceID:   "trace-4",
		SessionID: "session-4",
		Message:   "check mysql-0 logs",
		EventMeta: EventMeta{
			SessionID: "session-4",
			TraceID:   "trace-4",
			PlanID:    "plan-4",
		},
		EmitEvent: func(name string, _ EventMeta, _ map[string]any) bool {
			names = append(names, name)
			return true
		},
		Plan: planner.ExecutionPlan{
			PlanID: "plan-4",
			Goal:   "check mysql-0 logs",
			Steps: []planner.PlanStep{{
				StepID: "step-1",
				Title:  "fetch logs",
				Expert: "k8s",
				Task:   "fetch pod logs",
				Mode:   "readonly",
				Risk:   "low",
			}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.State.Steps["step-1"].Status != runtime.StepCompleted {
		t.Fatalf("step status = %s, want %s", result.State.Steps["step-1"].Status, runtime.StepCompleted)
	}
	want := []string{"step_update", "tool_call", "tool_result", "step_update"}
	if len(names) != len(want) {
		t.Fatalf("event count = %d, want %d (%v)", len(names), len(want), names)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("event[%d] = %s, want %s", i, names[i], name)
		}
	}
}

func TestExecutorClassifiesExpertFailures(t *testing.T) {
	store := newExecutionStore(t)
	runner := &stubStepRunner{err: &ExecutionError{
		Code:        "expert_result_invalid",
		Message:     "expert returned non-JSON output",
		UserSummary: "专家已执行，但返回结果格式不符合协议。",
	}}
	exec := New(store, WithStepRunner(runner))
	ctx := context.Background()

	result, err := exec.Run(ctx, Request{
		TraceID:   "trace-5",
		SessionID: "session-5",
		Message:   "inspect mysql-0",
		Plan: planner.ExecutionPlan{
			PlanID: "plan-5",
			Goal:   "inspect mysql-0",
			Steps: []planner.PlanStep{{
				StepID: "step-1",
				Title:  "analyze pod health",
				Expert: "analysis",
				Mode:   "readonly",
				Risk:   "low",
			}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	step := result.State.Steps["step-1"]
	if step.ErrorCode != "expert_result_invalid" {
		t.Fatalf("error code = %q, want %q", step.ErrorCode, "expert_result_invalid")
	}
	if step.UserVisibleSummary != "专家已执行，但返回结果格式不符合协议。" {
		t.Fatalf("user summary = %q", step.UserVisibleSummary)
	}
}

func TestExecutorCompactsMissingPrerequisiteErrors(t *testing.T) {
	store := newExecutionStore(t)
	runner := &stubStepRunner{err: fmt.Errorf("[NodeRunError] failed to invoke tool[name:k8s_get_pod_logs id:call_1]: [LocalFunc] failed to invoke tool, toolName=k8s_get_pod_logs, err=k8s client unavailable: cluster_id is required ------------------------ node path: [node_1, ToolNode]")}
	exec := New(store, WithStepRunner(runner))
	ctx := context.Background()

	result, err := exec.Run(ctx, Request{
		TraceID:   "trace-6",
		SessionID: "session-6",
		Message:   "查看 mysql-0 的日志",
		Plan: planner.ExecutionPlan{
			PlanID: "plan-6",
			Goal:   "查看 mysql-0 的日志",
			Steps: []planner.PlanStep{{
				StepID: "step-1",
				Title:  "拉取 pod 日志",
				Expert: "k8s",
				Task:   "fetch mysql-0 logs",
				Mode:   "readonly",
				Risk:   "low",
			}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	step := result.State.Steps["step-1"]
	if step.ErrorCode != "missing_execution_prerequisite" {
		t.Fatalf("error code = %q", step.ErrorCode)
	}
	if got := step.UserVisibleSummary; got != "当前没有可执行的集群上下文。缺少前置上下文：cluster_id" {
		t.Fatalf("user summary = %q", got)
	}
	if strings.Contains(step.ErrorMessage, "node path") {
		t.Fatalf("error message should be compacted, got %q", step.ErrorMessage)
	}
}

func TestExecutorCompactsProviderTimeoutErrors(t *testing.T) {
	store := newExecutionStore(t)
	runner := &stubStepRunner{err: fmt.Errorf("[NodeRunError] failed to create chat completion: Post \"https://coding.dashscope.aliyuncs.com/v1/chat/completions\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)")}
	exec := New(store, WithStepRunner(runner))
	ctx := context.Background()

	result, err := exec.Run(ctx, Request{
		TraceID:   "trace-timeout",
		SessionID: "session-timeout",
		Message:   "在火山云服务器上执行 df -h 并完整返回结果",
		Plan: planner.ExecutionPlan{
			PlanID: "plan-timeout",
			Goal:   "在火山云服务器上执行 df -h 并完整返回结果",
			Steps: []planner.PlanStep{{
				StepID: "step-1",
				Title:  "执行主机命令",
				Expert: "hostops",
				Task:   "在火山云服务器上执行 df -h",
				Mode:   "readonly",
				Risk:   "low",
			}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	step := result.State.Steps["step-1"]
	if step.ErrorCode != "expert_tool_stream_failed" {
		t.Fatalf("error code = %q", step.ErrorCode)
	}
	if got := step.UserVisibleSummary; got != "专家 hostops 调用模型超时，请稍后重试。" {
		t.Fatalf("user summary = %q", got)
	}
	if got := step.ErrorMessage; got != "调用模型超时" {
		t.Fatalf("error message = %q, want compact timeout message", got)
	}
}

func TestExecutorSupportsMultiExpertPlanExecution(t *testing.T) {
	store := newExecutionStore(t)
	runner := &stubStepRunner{result: StepResult{Summary: "expert completed step"}}
	exec := New(store, WithStepRunner(runner))
	ctx := context.Background()

	result, err := exec.Run(ctx, Request{
		TraceID:   "trace-7",
		SessionID: "session-7",
		Message:   "inspect service and verify metrics",
		Plan: planner.ExecutionPlan{
			PlanID: "plan-7",
			Goal:   "inspect service and verify metrics",
			Steps: []planner.PlanStep{
				{
					StepID: "step-1",
					Title:  "检查服务状态",
					Expert: "service",
					Task:   "inspect service runtime state",
					Mode:   "readonly",
					Risk:   "low",
					Input: map[string]any{
						"service_id": 42,
					},
				},
				{
					StepID:    "step-2",
					Title:     "分析指标",
					Expert:    "observability",
					Task:      "analyze service metrics",
					Mode:      "readonly",
					Risk:      "low",
					DependsOn: []string{"step-1"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if runner.calls != 2 {
		t.Fatalf("runner calls = %d, want 2", runner.calls)
	}
	if got := result.State.Steps["step-1"].Status; got != runtime.StepCompleted {
		t.Fatalf("step-1 status = %s, want %s", got, runtime.StepCompleted)
	}
	if got := result.State.Steps["step-2"].Status; got != runtime.StepCompleted {
		t.Fatalf("step-2 status = %s, want %s", got, runtime.StepCompleted)
	}
	completed := 0
	for _, step := range result.Steps {
		if step.Status == runtime.StepCompleted {
			completed++
		}
	}
	if completed != 2 {
		t.Fatalf("completed step results = %d, want 2 (%#v)", completed, result.Steps)
	}
}
