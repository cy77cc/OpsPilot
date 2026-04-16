package cicd

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

func BuildCICDSummary(workerSummary contracts.DelegationSummary, pipeline string) contracts.DelegationSummary {
	summaryText := strings.TrimSpace(workerSummary.Summary)
	if summaryText == "" {
		switch workerSummary.Status {
		case contracts.StatusFailed:
			summaryText = "CI/CD specialist could not complete pipeline diagnostics for the requested scope."
		default:
			summaryText = "CI/CD specialist completed pipeline diagnostics for the requested scope."
		}
	}

	nextAction := strings.TrimSpace(workerSummary.RecommendedNextAction)
	if nextAction == "" {
		switch workerSummary.Status {
		case contracts.StatusFailed:
			nextAction = "Ask deep_main to inspect pipeline error details or retry with narrowed filters."
		default:
			nextAction = "Ask deep_main whether to continue failure triage or prepare a governed pipeline action."
		}
	}

	findings := append([]string{}, workerSummary.KeyFindings...)
	if p := strings.TrimSpace(pipeline); p != "" {
		findings = append([]string{"pipeline: " + p}, findings...)
	}

	return contracts.DelegationSummary{
		TaskID:                strings.TrimSpace(workerSummary.TaskID),
		AgentName:             name,
		Status:                workerSummary.Status,
		Summary:               summaryText,
		KeyFindings:           findings,
		RiskLevel:             workerSummary.RiskLevel,
		Confidence:            workerSummary.Confidence,
		RecommendedNextAction: nextAction,
		ArtifactRefs:          workerSummary.ArtifactRefs,
		Metrics:               workerSummary.Metrics,
	}
}
