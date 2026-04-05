package cluster

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	policyHitTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "policy_hit_total",
			Help: "Total number of policy hits grouped by policy and traffic intent.",
		},
		[]string{"policy_name", "action", "direction", "namespace"},
	)
	policyDenyTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "policy_deny_total",
			Help: "Total number of policy deny decisions grouped by policy and namespace.",
		},
		[]string{"policy_name", "namespace"},
	)
	policyReleaseDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "policy_release_duration_seconds",
			Help:    "Observed duration for policy release phases.",
			Buckets: []float64{0.5, 1, 2, 5, 10},
		},
		[]string{"phase"},
	)
	simulationEvaluationDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "simulation_evaluation_duration_seconds",
			Help:    "Observed duration for policy simulation evaluation.",
			Buckets: []float64{0.1, 0.5, 1, 2},
		},
	)
	cniAdapterTranslationErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cni_adapter_translation_errors_total",
			Help: "Total number of CNI adapter translation errors.",
		},
		[]string{"cni_type", "error_code"},
	)
	phase3AdmissionLatencySeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "phase3_admission_latency_seconds",
			Help:    "Observed admission gate latency for phase-3 security controls.",
			Buckets: []float64{0.05, 0.1, 0.2, 0.5, 0.8, 1.5},
		},
	)
	phase3RuntimeDetectToAlertSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "phase3_runtime_detect_to_alert_seconds",
			Help:    "Observed runtime detection to alert delay for phase-3 security controls.",
			Buckets: []float64{1, 3, 5, 10, 20, 30, 60},
		},
	)
	phase3ContainmentOutcomeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "phase3_containment_outcome_total",
			Help: "Containment outcomes grouped by execution mode and result.",
		},
		[]string{"mode", "result"},
	)
	phase3SLOGradeGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "phase3_slo_grade",
			Help: "Current SLO grade score by metric: healthy=2, warning=1, critical=0.",
		},
		[]string{"metric"},
	)
)

func RecordPolicyHit(policyName, action, direction, namespace string) {
	policyHitTotal.WithLabelValues(
		policyMetricLabel(policyName, "unknown_policy"),
		policyMetricLabel(action, "unknown_action"),
		policyMetricLabel(direction, "unknown_direction"),
		policyMetricLabel(namespace, "cluster"),
	).Inc()
}

func RecordPolicyDeny(policyName, namespace string) {
	policyDenyTotal.WithLabelValues(
		policyMetricLabel(policyName, "unknown_policy"),
		policyMetricLabel(namespace, "cluster"),
	).Inc()
}

func ObservePolicyReleaseDuration(phase string, duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	policyReleaseDurationSeconds.WithLabelValues(
		policyMetricLabel(phase, "unknown"),
	).Observe(duration.Seconds())
}

func ObservePolicyReleaseDurationSince(phase string, startedAt time.Time) {
	if startedAt.IsZero() {
		return
	}
	ObservePolicyReleaseDuration(phase, time.Since(startedAt))
}

func ObserveSimulationEvaluationDuration(duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	simulationEvaluationDurationSeconds.Observe(duration.Seconds())
}

func ObserveSimulationEvaluationDurationSince(startedAt time.Time) {
	if startedAt.IsZero() {
		return
	}
	ObserveSimulationEvaluationDuration(time.Since(startedAt))
}

func RecordCNIAdapterTranslationError(cniType, errorCode string) {
	cniAdapterTranslationErrorsTotal.WithLabelValues(
		policyMetricLabel(cniType, "unknown"),
		policyMetricLabel(errorCode, "unknown"),
	).Inc()
}

func ObservePhase3AdmissionLatency(duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	phase3AdmissionLatencySeconds.Observe(duration.Seconds())
}

func ObservePhase3RuntimeDetectToAlert(duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	phase3RuntimeDetectToAlertSeconds.Observe(duration.Seconds())
}

func RecordPhase3ContainmentOutcome(mode string, success bool) {
	result := "failed"
	if success {
		result = "success"
	}
	phase3ContainmentOutcomeTotal.WithLabelValues(
		policyMetricLabel(mode, "unknown"),
		result,
	).Inc()
}

func SetPhase3SLOGrade(metric, grade string) {
	phase3SLOGradeGauge.WithLabelValues(policyMetricLabel(metric, "unknown")).Set(gradeToScore(grade))
}

func gradeToScore(grade string) float64 {
	switch strings.ToLower(strings.TrimSpace(grade)) {
	case "healthy":
		return 2
	case "warning":
		return 1
	default:
		return 0
	}
}

func policyMetricLabel(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
