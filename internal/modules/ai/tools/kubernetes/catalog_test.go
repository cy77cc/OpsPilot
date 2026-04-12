package kubernetes

import "testing"

func TestCatalogMetadataList_UsesCanonicalKubernetesToolNames(t *testing.T) {
	metadata := CatalogMetadataList()
	names := make(map[string]struct{}, len(metadata))
	for _, item := range metadata {
		names[item.ToolName] = struct{}{}
	}

	for _, required := range []string{"k8s_query", "k8s_events", "k8s_logs", "k8s_scale_deployment"} {
		if _, ok := names[required]; !ok {
			t.Fatalf("expected %s in kubernetes catalog, got %#v", required, names)
		}
	}

	for _, legacy := range []string{"k8s_get_events", "k8s_get_pod_logs", "kubectl_get_pods", "kubectl_logs"} {
		if _, ok := names[legacy]; ok {
			t.Fatalf("did not expect legacy kubernetes tool %s in catalog", legacy)
		}
	}
}
