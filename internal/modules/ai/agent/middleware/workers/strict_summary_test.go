package workers

import (
	"strings"
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

func TestValidateStrictSummary_Table(t *testing.T) {
	validBase := contracts.DelegationSummary{
		TaskID:    "task-valid",
		AgentName: "isolation_worker",
		Status:    contracts.StatusReturned,
		Summary:   "condensed summary",
	}

	tests := []struct {
		name      string
		summary   contracts.DelegationSummary
		wantError bool
		errLike   string
	}{
		{
			name: "accepts valid summary",
			summary: contracts.DelegationSummary{
				TaskID:    validBase.TaskID,
				AgentName: validBase.AgentName,
				Status:    validBase.Status,
				Summary:   validBase.Summary,
				Metrics: map[string]any{
					"ratio": "0.91",
				},
			},
			wantError: false,
		},
		{
			name: "rejects exact raw key",
			summary: contracts.DelegationSummary{
				TaskID:    validBase.TaskID,
				AgentName: validBase.AgentName,
				Status:    validBase.Status,
				Summary:   validBase.Summary,
				Metrics: map[string]any{
					"raw": "x",
				},
			},
			wantError: true,
			errLike:   "raw metric payloads",
		},
		{
			name: "rejects raw prefix key",
			summary: contracts.DelegationSummary{
				TaskID:    validBase.TaskID,
				AgentName: validBase.AgentName,
				Status:    validBase.Status,
				Summary:   validBase.Summary,
				Metrics: map[string]any{
					"raw_payload": "x",
				},
			},
			wantError: true,
			errLike:   "raw metric payloads",
		},
		{
			name: "rejects raw suffix key",
			summary: contracts.DelegationSummary{
				TaskID:    validBase.TaskID,
				AgentName: validBase.AgentName,
				Status:    validBase.Status,
				Summary:   validBase.Summary,
				Metrics: map[string]any{
					"payload_raw": "x",
				},
			},
			wantError: true,
			errLike:   "raw metric payloads",
		},
		{
			name: "does not reject substring only",
			summary: contracts.DelegationSummary{
				TaskID:    validBase.TaskID,
				AgentName: validBase.AgentName,
				Status:    validBase.Status,
				Summary:   validBase.Summary,
				Metrics: map[string]any{
					"drawdown": "small",
				},
			},
			wantError: false,
		},
		{
			name: "rejects oversized metric string",
			summary: contracts.DelegationSummary{
				TaskID:    validBase.TaskID,
				AgentName: validBase.AgentName,
				Status:    validBase.Status,
				Summary:   validBase.Summary,
				Metrics: map[string]any{
					"note": strings.Repeat("x", maxWorkerMetricTextBytes+1),
				},
			},
			wantError: true,
			errLike:   "payload too large",
		},
		{
			name: "delegates summary validate failures",
			summary: contracts.DelegationSummary{
				TaskID:    "",
				AgentName: validBase.AgentName,
				Status:    validBase.Status,
				Summary:   validBase.Summary,
			},
			wantError: true,
			errLike:   "task_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStrictSummary(tt.summary)
			if tt.wantError && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantError && tt.errLike != "" && !strings.Contains(err.Error(), tt.errLike) {
				t.Fatalf("expected error containing %q, got %q", tt.errLike, err.Error())
			}
		})
	}
}
