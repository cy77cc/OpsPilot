package shared

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

// ApplySummaryDefaults normalizes summary fields for summary-only child agent returns.
func ApplySummaryDefaults(summary contracts.DelegationSummary, fallbackSummary string, fallbackNextAction string) contracts.DelegationSummary {
	normalized := summary

	normalized.Summary = strings.TrimSpace(normalized.Summary)
	if normalized.Summary == "" {
		normalized.Summary = strings.TrimSpace(fallbackSummary)
	}

	normalized.RecommendedNextAction = strings.TrimSpace(normalized.RecommendedNextAction)
	if normalized.RecommendedNextAction == "" {
		normalized.RecommendedNextAction = strings.TrimSpace(fallbackNextAction)
	}

	if len(normalized.KeyFindings) > 0 {
		findings := make([]string, 0, len(normalized.KeyFindings))
		for _, finding := range normalized.KeyFindings {
			finding = strings.TrimSpace(finding)
			if finding != "" {
				findings = append(findings, finding)
			}
		}
		normalized.KeyFindings = findings
	}

	return normalized
}
