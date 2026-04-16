package isolation

import (
	"testing"
)

func TestReduceMetricPoints_ReturnsSummaryOnly(t *testing.T) {
	summary := ReduceMetricPoints("http_request_duration_seconds", []float64{0.18, 0.21, 1.60})
	if err := summary.Validate(); err != nil {
		t.Fatalf("expected valid summary, got %v", err)
	}
	if len(summary.KeyFindings) == 0 {
		t.Fatal("expected findings")
	}
}
