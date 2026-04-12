package kubernetes

import "testing"

func TestCatalogMetadataList_ContainsCoreToolsAndNoLegacyNames(t *testing.T) {
	metadata := CatalogMetadataList()
	names := make([]string, 0, len(metadata))
	for _, item := range metadata {
		names = append(names, item.ToolName)
	}
	for _, required := range []string{"k8s_query", "k8s_logs", "k8s_scale_deployment"} {
		if !containsTool(names, required) {
			t.Fatalf("expected %s in catalog metadata, got %v", required, names)
		}
	}
	for _, legacy := range []string{"kubectl_get_pods", "kubectl_logs"} {
		if containsTool(names, legacy) {
			t.Fatalf("did not expect legacy tool %s in catalog metadata, got %v", legacy, names)
		}
	}
}

func containsTool(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
