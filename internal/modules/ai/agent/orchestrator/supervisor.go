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

type delegationSemantics struct {
	intent         contracts.Intent
	expectedOutput contracts.ExpectedOutput
}

func NewSupervisor(registry *Registry) *Supervisor {
	return &Supervisor{registry: registry}
}

func (s *Supervisor) BuildDelegationTask(
	runID, targetAgent, question string,
	scope contracts.Scope,
) contracts.DelegationTask {
	intent, expectedOutput := s.deriveDelegationSemantics(targetAgent)
	return contracts.DelegationTask{
		TaskID:         uuid.NewString(),
		ParentRunID:    runID,
		DelegationID:   fmt.Sprintf("delegation:%s", uuid.NewString()),
		TargetAgent:    targetAgent,
		Intent:         intent,
		Question:       question,
		Scope:          scope,
		ExpectedOutput: expectedOutput,
	}
}

func (s *Supervisor) deriveDelegationSemantics(targetAgent string) (contracts.Intent, contracts.ExpectedOutput) {
	if s != nil && s.registry != nil {
		if spec, ok := s.registry.Lookup(targetAgent); ok {
			semantics := semanticsForTarget(spec.Name)
			return semantics.intent, semantics.expectedOutput
		}
	}
	semantics := semanticsForTarget(targetAgent)
	return semantics.intent, semantics.expectedOutput
}

func semanticsForTarget(target string) delegationSemantics {
	switch normalizeScene(target) {
	case "monitor", "monitoring":
		return delegationSemantics{
			intent:         contracts.IntentMetricAnomalySummary,
			expectedOutput: contracts.ExpectedMetricAnomalySummary,
		}
	case "kubernetes":
		return delegationSemantics{
			intent:         contracts.IntentResourceInventory,
			expectedOutput: contracts.ExpectedResourceInventory,
		}
	case "host":
		return delegationSemantics{
			intent:         contracts.IntentHostHealthSummary,
			expectedOutput: contracts.ExpectedHostHealthSummary,
		}
	case "cicd":
		return delegationSemantics{
			intent:         contracts.IntentPipelineFailure,
			expectedOutput: contracts.ExpectedPipelineFailure,
		}
	default:
		return delegationSemantics{
			intent:         contracts.IntentMetricAnomalySummary,
			expectedOutput: contracts.ExpectedMetricAnomalySummary,
		}
	}
}
