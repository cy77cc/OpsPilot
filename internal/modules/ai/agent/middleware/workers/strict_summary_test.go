package workers

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

func TestStrictSummary_RejectsRawPayload(t *testing.T) {
	summary := contracts.DelegationSummary{
		TaskID:    "task-1",
		AgentName: "isolation_worker",
		Status:    contracts.StatusReturned,
		Summary:   "ok",
		Metrics: map[string]any{
			"raw_json": `{"huge":"payload"}`,
		},
	}
	if err := ValidateStrictSummary(summary); err == nil {
		t.Fatal("expected strict summary validation error")
	}
}
