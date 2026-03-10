package summarizer

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/ai/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

func TestSummarizerMarksNeedMoreInvestigation(t *testing.T) {
	s := New(nil)
	out, err := s.Summarize(context.Background(), Input{
		Message: "check payment-api",
		Plan: &planner.ExecutionPlan{
			PlanID: "plan-1",
			Goal:   "check payment-api",
		},
		State: runtime.ExecutionState{
			PlanID: "plan-1",
		},
		Steps: []executor.StepResult{
			{StepID: "step-1", Status: runtime.StepFailed, Summary: "service check failed"},
		},
	})
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if !out.NeedMoreInvestigation {
		t.Fatalf("NeedMoreInvestigation = false, want true")
	}
	if out.ReplanHint == nil {
		t.Fatalf("ReplanHint = nil, want non-nil")
	}
}

func TestSummarizerExplainsApprovalWait(t *testing.T) {
	s := New(nil)
	out, err := s.Summarize(context.Background(), Input{
		Message: "deploy payment-api",
		State: runtime.ExecutionState{
			PlanID: "plan-2",
			PendingApproval: &runtime.PendingApproval{
				PlanID: "plan-2",
				StepID: "step-2",
				Title:  "发布服务",
				Status: "pending",
			},
		},
	})
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if out.NeedMoreInvestigation {
		t.Fatalf("NeedMoreInvestigation = true, want false")
	}
	if out.Conclusion == "" {
		t.Fatalf("Conclusion is empty")
	}
}

func TestSummarizerRequestsInvestigationWhenCompletedStepsLackEvidence(t *testing.T) {
	s := New(nil)
	out, err := s.Summarize(context.Background(), Input{
		Message: "查看 cilium 日志",
		Plan: &planner.ExecutionPlan{
			PlanID: "plan-3",
			Goal:   "查看 cilium 日志",
		},
		State: runtime.ExecutionState{
			PlanID: "plan-3",
		},
		Steps: []executor.StepResult{
			{StepID: "step-1", Status: runtime.StepCompleted, Summary: "成功读取最近 100 条日志"},
		},
	})
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if !out.NeedMoreInvestigation {
		t.Fatalf("NeedMoreInvestigation = false, want true")
	}
	if out.ReplanHint == nil || out.ReplanHint.Reason != "completed_steps_without_evidence" {
		t.Fatalf("ReplanHint = %#v", out.ReplanHint)
	}
	if out.Conclusion == "" || out.Narrative == "" {
		t.Fatalf("Conclusion/Narrative should not be empty: %#v", out)
	}
}

func TestNormalizeSummaryQualifiesUncertainConclusions(t *testing.T) {
	base := SummaryOutput{
		Summary:               "证据不足",
		Conclusion:            "当前只能给出初步判断，仍需进一步调查。",
		Narrative:             "当前叙述仅基于已完成步骤和现有证据，不足部分仍待补充确认。",
		NeedMoreInvestigation: true,
		ReplanHint: &ReplanHint{
			Reason:          "evidence_missing",
			Focus:           "补充证据",
			MissingEvidence: []string{"step_evidence"},
		},
	}

	out := normalizeSummary(base, SummaryOutput{
		Summary:    "日志显示正常",
		Conclusion: "服务已经恢复",
		Narrative:  "根据结果可以确认服务恢复",
	})

	if !out.NeedMoreInvestigation {
		t.Fatalf("NeedMoreInvestigation = false, want true")
	}
	if out.ReplanHint == nil {
		t.Fatalf("ReplanHint = nil")
	}
	if out.Conclusion == "服务已经恢复" {
		t.Fatalf("Conclusion should be qualified, got %q", out.Conclusion)
	}
	if out.Narrative == "根据结果可以确认服务恢复" {
		t.Fatalf("Narrative should be qualified, got %q", out.Narrative)
	}
}
