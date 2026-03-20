package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/kubernetes"
)

func TestDiagnosisAndChangeTools_IncludePlatformDiscovery(t *testing.T) {
	for _, tt := range []struct {
		name  string
		build func(context.Context) []tool.BaseTool
	}{
		{
			name:  "diagnosis",
			build: NewDiagnosisTools,
		},
		{
			name:  "change",
			build: NewChangeTools,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, item := range tt.build(context.Background()) {
				info, err := item.Info(t.Context())
				if err != nil {
					t.Fatalf("get tool info: %v", err)
				}
				switch info.Name {
				case "platform_discover_resources", "load_session_history":
					if info.Name == "load_session_history" {
						found = true
					}
				}
			}
			if !found {
				t.Fatal("expected load_session_history in toolset")
			}
		})
	}
}

func TestKubernetesReadonlyToolDescriptions_RequireClusterResolution(t *testing.T) {
	targets := map[string]struct{}{
		"k8s_query":          {},
		"k8s_list_resources": {},
		"k8s_events":         {},
		"k8s_get_events":     {},
		"k8s_logs":           {},
		"k8s_get_pod_logs":   {},
	}

	for _, item := range kubernetes.NewKubernetesTools(context.Background()) {
		info, err := item.Info(t.Context())
		if err != nil {
			t.Fatalf("get tool info: %v", err)
		}
		if _, ok := targets[info.Name]; !ok {
			continue
		}
		if !strings.Contains(info.Desc, "cluster_id is required") {
			t.Fatalf("expected %s description to say cluster_id is required, got %q", info.Name, info.Desc)
		}
		if !strings.Contains(info.Desc, "resolve it first") {
			t.Fatalf("expected %s description to tell model to resolve cluster_id first, got %q", info.Name, info.Desc)
		}
		delete(targets, info.Name)
	}

	if len(targets) != 0 {
		t.Fatalf("missing expected tools from readonly toolset: %v", targets)
	}
}
