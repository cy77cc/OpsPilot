package monitor

import "testing"

func TestCatalogMetadataList_ContainsExpectedMonitorTools(t *testing.T) {
	metadata := CatalogMetadataList()
	names := make([]string, 0, len(metadata))
	for _, item := range metadata {
		names = append(names, item.ToolName)
	}
	for _, required := range []string{"monitor_alert_rule_list", "monitor_alert", "monitor_metric"} {
		if !containsTool(names, required) {
			t.Fatalf("expected %s in catalog metadata, got %v", required, names)
		}
	}
	for _, legacy := range []string{"monitor_alert_active", "monitor_metric_query"} {
		if containsTool(names, legacy) {
			t.Fatalf("did not expect legacy monitor tool %s in catalog metadata, got %v", legacy, names)
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
