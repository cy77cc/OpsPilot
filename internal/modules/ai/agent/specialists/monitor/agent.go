package monitor

import "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"

func ShouldDelegateToIsolationWorker(scope contracts.Scope, pointCount int) bool {
	if pointCount > 500 {
		return true
	}

	switch scope.TimeRange {
	case "6h", "12h", "24h", "48h", "7d":
		return pointCount > 100
	default:
		return false
	}
}
