package isolation

import (
	"testing"
)

func TestReduceMetricPoints_ReturnsSummaryOnly(t *testing.T) {
	const taskID = "task-123"
	summary := ReduceMetricPoints(taskID, "http_request_duration_seconds", []float64{0.18, 0.21, 1.60})
	if err := summary.Validate(); err != nil {
		t.Fatalf("expected valid summary, got %v", err)
	}
	if summary.TaskID != taskID {
		t.Fatalf("expected task id %q, got %q", taskID, summary.TaskID)
	}
	if len(summary.KeyFindings) == 0 {
		t.Fatal("expected findings")
	}
}

func TestReduceMetricPoints_EmptyPoints(t *testing.T) {
	const taskID = "task-empty"
	summary := ReduceMetricPoints(taskID, "http_request_duration_seconds", nil)
	if err := summary.Validate(); err != nil {
		t.Fatalf("expected valid empty summary, got %v", err)
	}
	if summary.TaskID != taskID {
		t.Fatalf("expected task id %q, got %q", taskID, summary.TaskID)
	}
	if summary.Summary != "No metric samples were returned." {
		t.Fatalf("unexpected empty summary text: %q", summary.Summary)
	}
}

func TestSummarizeMetricPoints_PropagatesTaskID(t *testing.T) {
	const taskID = "task-through-wrapper"
	summary := SummarizeMetricPoints(taskID, "http_request_duration_seconds", []float64{0.2, 0.5})
	if summary.TaskID != taskID {
		t.Fatalf("expected task id %q, got %q", taskID, summary.TaskID)
	}
}
