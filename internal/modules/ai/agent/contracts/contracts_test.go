package contracts

import "testing"

func TestDelegationTask_ValidateRequiresCoreFields(t *testing.T) {
	task := DelegationTask{
		TaskID:         "task-1",
		ParentRunID:    "run-1",
		DelegationID:   "delegation-1",
		TargetAgent:    "monitor",
		Intent:         IntentMetricAnomalySummary,
		Question:       "Summarize the latency spike.",
		ExpectedOutput: ExpectedMetricAnomalySummary,
	}
	if err := task.Validate(); err != nil {
		t.Fatalf("expected valid task, got %v", err)
	}
}

func TestDelegationSummary_ValidateRejectsEmptySummary(t *testing.T) {
	summary := DelegationSummary{
		TaskID:    "task-1",
		AgentName: "monitor",
		Status:    StatusReturned,
	}
	if err := summary.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestFailureCode_IsKnown(t *testing.T) {
	if !FailureCodeTimeout.Known() {
		t.Fatal("expected timeout to be known")
	}
	if FailureCode("mystery").Known() {
		t.Fatal("did not expect mystery code to be known")
	}
}
