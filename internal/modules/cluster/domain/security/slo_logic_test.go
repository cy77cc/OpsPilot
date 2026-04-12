package security

import (
	"testing"
	"time"

	clusterpolicy "github.com/cy77cc/OpsPilot/internal/modules/cluster/domain/policy"
)

func TestPhase3SLO_AdmissionLatencyBudget(t *testing.T) {
	samples := []time.Duration{120 * time.Millisecond, 240 * time.Millisecond, 350 * time.Millisecond, 560 * time.Millisecond, 790 * time.Millisecond}
	result := EvaluateAdmissionLatencyBudget(samples)
	if !result.Passed {
		t.Fatalf("expected admission latency budget pass, got %#v", result)
	}
	if result.Unit != "ms" {
		t.Fatalf("expected ms unit, got %s", result.Unit)
	}
}

func TestPhase3SLO_RuntimeDetectToAlertP95(t *testing.T) {
	samples := []time.Duration{5 * time.Second, 8 * time.Second, 12 * time.Second, 33 * time.Second, 36 * time.Second}
	result := EvaluateRuntimeDetectToAlertP95(samples)
	if result.Passed {
		t.Fatalf("expected runtime detect-to-alert SLO to fail, got %#v", result)
	}
	if result.Grade == "healthy" {
		t.Fatalf("expected degraded grade for failing slo, got %s", result.Grade)
	}
}

func TestPolicyMetrics_Phase3Recorders(t *testing.T) {
	clusterpolicy.ObservePhase3AdmissionLatency(420 * time.Millisecond)
	clusterpolicy.ObservePhase3RuntimeDetectToAlert(12 * time.Second)
	clusterpolicy.RecordPhase3ContainmentOutcome("auto", true)
	clusterpolicy.RecordPhase3ContainmentOutcome("suggest_only", false)
	clusterpolicy.SetPhase3SLOGrade("runtime_detect_to_alert_p95", "warning")

	result := EvaluateContainmentSuccessRate(10, 9)
	if result.Passed {
		t.Fatalf("expected containment success SLO fail at 90%%, got %#v", result)
	}
}
