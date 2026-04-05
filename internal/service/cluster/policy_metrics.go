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

func policyMetricLabel(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
