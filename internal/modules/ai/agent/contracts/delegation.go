package contracts

import (
	"fmt"
	"strings"
)

type Intent string

const (
	IntentMetricAnomalySummary Intent = "metric_anomaly_summary"
	IntentResourceInventory    Intent = "resource_inventory_summary"
	IntentHostHealthSummary    Intent = "host_health_summary"
	IntentPipelineFailure      Intent = "pipeline_failure_summary"
	IntentReleaseReadiness     Intent = "release_readiness_summary"
)

type ExpectedOutput string

const (
	ExpectedMetricAnomalySummary ExpectedOutput = "metric_anomaly_summary"
	ExpectedResourceInventory    ExpectedOutput = "resource_inventory_summary"
	ExpectedHostHealthSummary    ExpectedOutput = "host_health_summary"
	ExpectedPipelineFailure      ExpectedOutput = "pipeline_failure_summary"
	ExpectedReleaseReadiness     ExpectedOutput = "release_readiness_summary"
)

type DelegationTask struct {
	TaskID         string         `json:"task_id"`
	ParentRunID    string         `json:"parent_run_id"`
	DelegationID   string         `json:"delegation_id"`
	TargetAgent    string         `json:"target_agent"`
	Intent         Intent         `json:"intent"`
	Question       string         `json:"question"`
	Scope          Scope          `json:"scope"`
	Constraints    []string       `json:"constraints,omitempty"`
	InputArtifacts []ArtifactRef  `json:"input_artifacts,omitempty"`
	ExpectedOutput ExpectedOutput `json:"expected_output"`
	DeadlineHint   string         `json:"deadline_hint,omitempty"`
}

func (t DelegationTask) Validate() error {
	switch {
	case strings.TrimSpace(t.TaskID) == "":
		return fmt.Errorf("task_id is required")
	case strings.TrimSpace(t.ParentRunID) == "":
		return fmt.Errorf("parent_run_id is required")
	case strings.TrimSpace(t.DelegationID) == "":
		return fmt.Errorf("delegation_id is required")
	case strings.TrimSpace(t.TargetAgent) == "":
		return fmt.Errorf("target_agent is required")
	case strings.TrimSpace(string(t.Intent)) == "":
		return fmt.Errorf("intent is required")
	case strings.TrimSpace(t.Question) == "":
		return fmt.Errorf("question is required")
	case strings.TrimSpace(string(t.ExpectedOutput)) == "":
		return fmt.Errorf("expected_output is required")
	default:
		return nil
	}
}
