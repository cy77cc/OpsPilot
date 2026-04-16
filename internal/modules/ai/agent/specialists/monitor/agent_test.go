package monitor

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

func TestShouldDelegateToIsolationWorker(t *testing.T) {
	tests := []struct {
		name       string
		scope      contracts.Scope
		pointCount int
		want       bool
	}{
		{
			name:       "long range at 100 does not delegate",
			scope:      contracts.Scope{TimeRange: "24h"},
			pointCount: 100,
			want:       false,
		},
		{
			name:       "long range at 101 delegates",
			scope:      contracts.Scope{TimeRange: "24h"},
			pointCount: 101,
			want:       true,
		},
		{
			name:       "short range at 100 does not delegate",
			scope:      contracts.Scope{TimeRange: "5m"},
			pointCount: 100,
			want:       false,
		},
		{
			name:       "short range at 101 still does not delegate",
			scope:      contracts.Scope{TimeRange: "5m"},
			pointCount: 101,
			want:       false,
		},
		{
			name:       "short range at 500 does not delegate",
			scope:      contracts.Scope{TimeRange: "5m"},
			pointCount: 500,
			want:       false,
		},
		{
			name:       "long range at 500 delegates",
			scope:      contracts.Scope{TimeRange: "24h"},
			pointCount: 500,
			want:       true,
		},
		{
			name:       "short range at 501 delegates",
			scope:      contracts.Scope{TimeRange: "5m"},
			pointCount: 501,
			want:       true,
		},
		{
			name:       "long range at 501 delegates",
			scope:      contracts.Scope{TimeRange: "24h"},
			pointCount: 501,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldDelegateToIsolationWorker(tt.scope, tt.pointCount)
			if got != tt.want {
				t.Fatalf("ShouldDelegateToIsolationWorker(%+v, %d) = %v, want %v", tt.scope, tt.pointCount, got, tt.want)
			}
		})
	}
}

func TestBuildMonitorSummary_PropagatesWorkerStatusAndSummary(t *testing.T) {
	worker := contracts.DelegationSummary{
		TaskID:      "task-failed",
		AgentName:   "isolation_worker",
		Status:      contracts.StatusFailed,
		Summary:     "No metric samples were returned.",
		KeyFindings: []string{"worker observed empty sample set"},
	}

	monitorSummary := BuildMonitorSummary(worker, "api-gateway", "24h")
	if monitorSummary.Status != contracts.StatusFailed {
		t.Fatalf("expected failed status to be preserved, got %q", monitorSummary.Status)
	}
	if monitorSummary.Summary != worker.Summary {
		t.Fatalf("expected worker summary to be preserved, got %q", monitorSummary.Summary)
	}
	if monitorSummary.TaskID != worker.TaskID {
		t.Fatalf("expected task id %q, got %q", worker.TaskID, monitorSummary.TaskID)
	}
	if monitorSummary.AgentName != "monitor" {
		t.Fatalf("expected wrapped agent name monitor, got %q", monitorSummary.AgentName)
	}
}
