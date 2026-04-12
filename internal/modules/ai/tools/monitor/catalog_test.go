package monitor

import "testing"

func TestCatalogMetadataList_UsesCanonicalMonitorToolNames(t *testing.T) {
	metadata := CatalogMetadataList()
	names := make(map[string]struct{}, len(metadata))
	for _, item := range metadata {
		names[item.ToolName] = struct{}{}
	}

	for _, required := range []string{"monitor_alert_rule_list", "monitor_alert", "monitor_metric"} {
		if _, ok := names[required]; !ok {
			t.Fatalf("expected %s in monitor catalog, got %#v", required, names)
		}
	}

	for _, legacy := range []string{"monitor_alert_active", "monitor_metric_query"} {
		if _, ok := names[legacy]; ok {
			t.Fatalf("did not expect legacy monitor tool %s in catalog", legacy)
		}
	}
}
