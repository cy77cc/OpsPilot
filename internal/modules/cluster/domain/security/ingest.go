package security

import (
	"encoding/json"
	"strings"

	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
)

type RuntimeIngestEvent struct {
	Namespace string
	Workload  string
	RuleID    string
	Severity  string
	Source    string
}

func ParseFalcoEvent(raw []byte) (RuntimeIngestEvent, error) {
	var payload struct {
		Rule         string `json:"rule"`
		Priority     string `json:"priority"`
		OutputFields struct {
			Namespace string `json:"k8s.ns.name"`
			PodName   string `json:"k8s.pod.name"`
		} `json:"output_fields"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RuntimeIngestEvent{}, err
	}
	return RuntimeIngestEvent{
		Namespace: payload.OutputFields.Namespace,
		Workload:  payload.OutputFields.PodName,
		RuleID:    strings.TrimSpace(payload.Rule),
		Severity:  normalizeSeverity(payload.Priority),
		Source:    clustermodel.SecurityEventSourceFalco,
	}, nil
}

func ParseTetragonEvent(raw []byte) (RuntimeIngestEvent, error) {
	var payload struct {
		PolicyName string `json:"policy_name"`
		Severity   string `json:"severity"`
		Namespace  string `json:"namespace"`
		Pod        string `json:"pod"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return RuntimeIngestEvent{}, err
	}
	return RuntimeIngestEvent{
		Namespace: payload.Namespace,
		Workload:  payload.Pod,
		RuleID:    strings.TrimSpace(payload.PolicyName),
		Severity:  normalizeSeverity(payload.Severity),
		Source:    clustermodel.SecurityEventSourceTetragon,
	}, nil
}

func normalizeSeverity(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "critical":
		return clustermodel.SecuritySeverityCritical
	case "high":
		return clustermodel.SecuritySeverityHigh
	case "medium":
		return clustermodel.SecuritySeverityMedium
	default:
		return clustermodel.SecuritySeverityLow
	}
}
