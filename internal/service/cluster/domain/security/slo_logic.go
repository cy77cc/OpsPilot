package security

import (
	"math"
	"sort"
	"time"
)

type SLOResult struct {
	Name   string  `json:"name"`
	Passed bool    `json:"passed"`
	Target float64 `json:"target"`
	Actual float64 `json:"actual"`
	Unit   string  `json:"unit"`
	Grade  string  `json:"grade"`
}

func EvaluateAdmissionLatencyBudget(samples []time.Duration) SLOResult {
	p95 := percentileDurationMs(samples, 0.95)
	target := 800.0
	return buildSLOResult("admission_latency_p95", p95 <= target, target, p95, "ms")
}

func EvaluateRuntimeDetectToAlertP95(samples []time.Duration) SLOResult {
	p95 := percentileDurationSeconds(samples, 0.95)
	target := 30.0
	return buildSLOResult("runtime_detect_to_alert_p95", p95 <= target, target, p95, "s")
}

func EvaluateContainmentSuccessRate(total, success int) SLOResult {
	target := 0.98
	actual := 0.0
	if total > 0 {
		actual = float64(success) / float64(total)
	}
	return buildSLOResult("runtime_containment_success_rate", actual >= target, target, actual, "ratio")
}

func buildSLOResult(name string, passed bool, target, actual float64, unit string) SLOResult {
	grade := "critical"
	delta := 0.0
	if target != 0 {
		delta = math.Abs(actual-target) / target
	}
	if passed {
		grade = "healthy"
	} else if delta <= 0.1 {
		grade = "warning"
	}
	return SLOResult{Name: name, Passed: passed, Target: target, Actual: actual, Unit: unit, Grade: grade}
}

func percentileDurationMs(samples []time.Duration, p float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	vals := make([]float64, 0, len(samples))
	for _, d := range samples {
		vals = append(vals, d.Seconds()*1000)
	}
	return percentile(vals, p)
}

func percentileDurationSeconds(samples []time.Duration, p float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	vals := make([]float64, 0, len(samples))
	for _, d := range samples {
		vals = append(vals, d.Seconds())
	}
	return percentile(vals, p)
}

func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	if p <= 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	sorted := append([]float64(nil), vals...)
	sort.Float64s(sorted)
	idx := int(math.Ceil(float64(len(sorted))*p)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
