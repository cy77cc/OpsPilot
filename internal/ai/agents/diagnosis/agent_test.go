package diagnosis

import (
	"context"
	"testing"
)

func TestAgent_DiagnoseReturnsProgressAndReportMetadata(t *testing.T) {
	agent := NewAgent()

	result, err := agent.Diagnose(context.Background(), Request{
		Message: "Diagnose why the rollout is failing",
	})
	if err != nil {
		t.Fatalf("diagnose message: %v", err)
	}
	if len(result.Progress) == 0 {
		t.Fatalf("expected progress updates, got %#v", result)
	}
	if result.Report.Summary == "" {
		t.Fatalf("expected report summary, got %#v", result.Report)
	}
}

func TestAgent_UsesOnlyReadonlyKubernetesTools(t *testing.T) {
	agent := NewAgent()

	tools := agent.ToolNames()
	if len(tools) == 0 {
		t.Fatalf("expected kubernetes tools to be available")
	}
	for _, name := range tools {
		switch name {
		case "k8s_query", "k8s_list_resources", "k8s_events", "k8s_get_events", "k8s_logs", "k8s_get_pod_logs":
		default:
			t.Fatalf("unexpected non-readonly tool %q", name)
		}
	}
}
