package ai

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/ai/events"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
)

func TestAIMetricsRecordRewriteQuality(t *testing.T) {
	metrics := NewAIMetrics()

	metrics.RecordRewrite(rewrite.Output{
		NormalizedGoal: "inspect pod logs",
		OperationMode:  "investigate",
		Narrative:      "structured rewrite",
	})
	metrics.RecordRewrite(rewrite.Output{
		NormalizedGoal: "incident lookup",
		OperationMode:  "investigate",
		Narrative:      "incident lookup rewrite",
		RetrievalQueries: []string{
			"pod logs restart incidents",
		},
		AmbiguityFlags: []string{"namespace"},
	})

	snapshot := metrics.Snapshot()
	if snapshot.Rewrite.Total != 2 {
		t.Fatalf("rewrite total = %d, want 2", snapshot.Rewrite.Total)
	}
	if snapshot.Rewrite.StructuredOutputs != 2 {
		t.Fatalf("structured outputs = %d, want 2", snapshot.Rewrite.StructuredOutputs)
	}
	if snapshot.Rewrite.Fallbacks != 0 {
		t.Fatalf("fallbacks = %d, want 0 after removing rewrite semantic fallback", snapshot.Rewrite.Fallbacks)
	}
	if snapshot.Rewrite.AmbiguousOutputs != 1 {
		t.Fatalf("ambiguous outputs = %d, want 1", snapshot.Rewrite.AmbiguousOutputs)
	}
	if snapshot.Rewrite.QualityRate != 1 {
		t.Fatalf("quality rate = %v, want 1", snapshot.Rewrite.QualityRate)
	}
}

func TestAIMetricsRecordPlannerRates(t *testing.T) {
	metrics := NewAIMetrics()

	metrics.RecordPlanner(planner.Decision{Type: planner.DecisionClarify})
	metrics.RecordPlanner(planner.Decision{
		Type: planner.DecisionPlan,
		Plan: &planner.ExecutionPlan{
			PlanID: "plan-1",
			Steps: []planner.PlanStep{
				{StepID: "step-1", Expert: "k8s"},
			},
		},
	})
	metrics.RecordPlanner(planner.Decision{Type: planner.DecisionDirectReply})

	snapshot := metrics.Snapshot()
	if snapshot.Planner.Total != 3 {
		t.Fatalf("planner total = %d, want 3", snapshot.Planner.Total)
	}
	if snapshot.Planner.Clarify != 1 {
		t.Fatalf("clarify count = %d, want 1", snapshot.Planner.Clarify)
	}
	if snapshot.Planner.Plans != 1 || snapshot.Planner.ExecutablePlans != 1 {
		t.Fatalf("plan counts = %#v, want one executable plan", snapshot.Planner)
	}
	if snapshot.Planner.ClarifyRate <= 0 || snapshot.Planner.ExecutablePlanRate != 1 {
		t.Fatalf("unexpected planner rates: %#v", snapshot.Planner)
	}
}

func TestAIMetricsRecordResumeRates(t *testing.T) {
	metrics := NewAIMetrics()

	metrics.RecordResume("approved", nil)
	metrics.RecordResume("idempotent", nil)
	metrics.RecordResume("missing", nil)

	snapshot := metrics.Snapshot()
	if snapshot.Resume.Total != 3 {
		t.Fatalf("resume total = %d, want 3", snapshot.Resume.Total)
	}
	if snapshot.Resume.Successful != 2 {
		t.Fatalf("resume successful = %d, want 2", snapshot.Resume.Successful)
	}
	if snapshot.Resume.Failures != 1 {
		t.Fatalf("resume failures = %d, want 1", snapshot.Resume.Failures)
	}
	if snapshot.Resume.DuplicateIntercepted != 1 {
		t.Fatalf("duplicate intercepted = %d, want 1", snapshot.Resume.DuplicateIntercepted)
	}
}

func TestAIMetricsThoughtChainCompleteness(t *testing.T) {
	metrics := NewAIMetrics()
	run := metrics.StartThoughtChainRun()

	run.Observe(StreamEvent{Type: events.RewriteResult})
	run.Observe(StreamEvent{Type: events.StageDelta, Data: map[string]any{"stage": "rewrite"}})
	run.Observe(StreamEvent{Type: events.PlannerState})
	run.Observe(StreamEvent{Type: events.StageDelta, Data: map[string]any{"stage": "plan"}})
	run.Observe(StreamEvent{Type: events.StepUpdate})
	run.Observe(StreamEvent{Type: events.StageDelta, Data: map[string]any{"stage": "execute"}})
	run.Finalize()

	snapshot := metrics.Snapshot()
	if snapshot.ThoughtChain.Runs != 1 {
		t.Fatalf("thought chain runs = %d, want 1", snapshot.ThoughtChain.Runs)
	}
	if snapshot.ThoughtChain.ExpectedStageSignals != 3 {
		t.Fatalf("expected stage signals = %d, want 3", snapshot.ThoughtChain.ExpectedStageSignals)
	}
	if snapshot.ThoughtChain.DeliveredStageSignals != 3 {
		t.Fatalf("delivered stage signals = %d, want 3", snapshot.ThoughtChain.DeliveredStageSignals)
	}
	if snapshot.ThoughtChain.RunsWithMissingSignals != 0 {
		t.Fatalf("runs with missing signals = %d, want 0", snapshot.ThoughtChain.RunsWithMissingSignals)
	}
}
