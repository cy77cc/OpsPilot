package workers

import (
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

const maxWorkerMetricTextBytes = 200
const rawMetricKey = "raw"

var strictRawMetricKeyDenylist = map[string]struct{}{
	"rawjson": {},
}

// ValidateStrictSummary enforces summary-only worker output and rejects raw payloads.
func ValidateStrictSummary(summary contracts.DelegationSummary) error {
	for key, value := range summary.Metrics {
		if isForbiddenRawMetricKey(key) {
			return fmt.Errorf("raw metric payloads are forbidden in worker summaries")
		}

		if text, ok := value.(string); ok && len(text) > maxWorkerMetricTextBytes {
			return fmt.Errorf("worker metric payload too large")
		}
	}

	return summary.Validate()
}

func isForbiddenRawMetricKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}

	if normalized == rawMetricKey || strings.HasPrefix(normalized, rawMetricKey+"_") || strings.HasSuffix(normalized, "_"+rawMetricKey) {
		return true
	}

	_, denied := strictRawMetricKeyDenylist[normalized]
	return denied
}
