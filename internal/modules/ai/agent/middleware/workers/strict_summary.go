package workers

import (
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

const maxWorkerMetricTextBytes = 200

// ValidateStrictSummary enforces summary-only worker output and rejects raw payloads.
func ValidateStrictSummary(summary contracts.DelegationSummary) error {
	for key, value := range summary.Metrics {
		if strings.Contains(strings.ToLower(strings.TrimSpace(key)), "raw") {
			return fmt.Errorf("raw metric payloads are forbidden in worker summaries")
		}

		if text, ok := value.(string); ok && len(text) > maxWorkerMetricTextBytes {
			return fmt.Errorf("worker metric payload too large")
		}
	}

	return summary.Validate()
}
