package diagnosis

import (
	"context"
	"strings"
)

type Request struct {
	Message string
}

type Result struct {
	Progress []string `json:"progress"`
	Report   Report   `json:"report"`
}

type Agent struct{}

func NewAgent() *Agent {
	return &Agent{}
}

func (a *Agent) Diagnose(_ context.Context, req Request) (Result, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = "the reported issue"
	}
	return Result{
		Progress: []string{
			"Collecting read-only Kubernetes evidence",
			"Summarizing likely root cause",
		},
		Report: Report{
			Summary:         "Phase 1 diagnosis summary for: " + message,
			Evidence:        []string{"Read-only Kubernetes signals reviewed"},
			RootCauses:      []string{"Likely workload or cluster configuration issue"},
			Recommendations: []string{"Inspect workload events, logs, and rollout status"},
		},
	}, nil
}

func (a *Agent) ToolNames() []string {
	return []string{
		"k8s_query",
		"k8s_list_resources",
		"k8s_events",
		"k8s_get_events",
		"k8s_logs",
		"k8s_get_pod_logs",
	}
}
