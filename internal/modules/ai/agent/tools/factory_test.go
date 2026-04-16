package tools

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

func TestBuildToolsForSceneWithMode_ReadOnlySpecialistsExcludeWriteTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		scene        string
		forbidTools  []string
		requiredTool string
	}{
		{
			name:         "cicd specialist stays read only",
			scene:        "cicd",
			forbidTools:  []string{"cicd_pipeline_trigger", "job_run"},
			requiredTool: "cicd_pipeline_list",
		},
		{
			name:         "host specialist stays read only",
			scene:        "host",
			forbidTools:  []string{"host_exec"},
			requiredTool: "host_list_inventory",
		},
		{
			name:         "kubernetes specialist stays read only",
			scene:        "kubernetes",
			forbidTools:  []string{"k8s_scale_deployment", "k8s_restart_deployment", "k8s_delete_pod"},
			requiredTool: "k8s_query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := baseToolNames(t, BuildToolsForSceneWithMode(context.Background(), tt.scene, true))
			if !containsToolName(names, tt.requiredTool) {
				t.Fatalf("expected %s in read-only %s tools, got %v", tt.requiredTool, tt.scene, names)
			}
			for _, forbidden := range tt.forbidTools {
				if containsToolName(names, forbidden) {
					t.Fatalf("did not expect %s in read-only %s tools, got %v", forbidden, tt.scene, names)
				}
			}
		})
	}
}

func baseToolNames(t *testing.T, tools []tool.BaseTool) []string {
	t.Helper()
	result := make([]string, 0, len(tools))
	for _, item := range tools {
		info, err := item.Info(t.Context())
		if err != nil {
			t.Fatalf("get tool info: %v", err)
		}
		result = append(result, info.Name)
	}
	return result
}

func containsToolName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
