package orchestrator

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

// Supervisor owns orchestration decisions and delegation task construction.
type Supervisor struct {
	registry *Registry
}

func NewSupervisor(registry *Registry) *Supervisor {
	return &Supervisor{registry: registry}
}

func (s *Supervisor) BuildDelegationTask(
	runID, targetAgent, question string,
	scope contracts.Scope,
) contracts.DelegationTask {
	return contracts.DelegationTask{
		TaskID:         uuid.NewString(),
		ParentRunID:    runID,
		DelegationID:   fmt.Sprintf("delegation:%s", uuid.NewString()),
		TargetAgent:    targetAgent,
		Intent:         contracts.IntentMetricAnomalySummary,
		Question:       question,
		Scope:          scope,
		ExpectedOutput: contracts.ExpectedMetricAnomalySummary,
	}
}
