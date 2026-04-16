package contracts

import (
	"fmt"
	"strings"
)

type DelegationStatus string

const (
	StatusReturned DelegationStatus = "returned"
	StatusFailed   DelegationStatus = "failed"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type DelegationSummary struct {
	TaskID                string           `json:"task_id"`
	AgentName             string           `json:"agent_name"`
	Status                DelegationStatus `json:"status"`
	Summary               string           `json:"summary"`
	KeyFindings           []string         `json:"key_findings,omitempty"`
	RiskLevel             RiskLevel        `json:"risk_level,omitempty"`
	Confidence            string           `json:"confidence,omitempty"`
	RecommendedNextAction string           `json:"recommended_next_action,omitempty"`
	ArtifactRefs          []ArtifactRef    `json:"artifact_refs,omitempty"`
	Metrics               map[string]any   `json:"metrics,omitempty"`
}

func (s DelegationSummary) Validate() error {
	switch {
	case strings.TrimSpace(s.TaskID) == "":
		return fmt.Errorf("task_id is required")
	case strings.TrimSpace(s.AgentName) == "":
		return fmt.Errorf("agent_name is required")
	case strings.TrimSpace(string(s.Status)) == "":
		return fmt.Errorf("status is required")
	case strings.TrimSpace(s.Summary) == "":
		return fmt.Errorf("summary is required")
	default:
		return nil
	}
}
