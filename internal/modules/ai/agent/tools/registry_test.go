package tools

import "testing"

func TestAllCatalogEntries_CoversCoreDomainsWithoutDuplicateNames(t *testing.T) {
	entries := AllCatalogEntries()
	if len(entries) == 0 {
		t.Fatal("expected non-empty catalog entries")
	}

	seen := make(map[string]struct{}, len(entries))
	domains := map[string]bool{
		"host":           false,
		"kubernetes":     false,
		"monitoring":     false,
		"service":        false,
		"deployment":     false,
		"cicd":           false,
		"infrastructure": false,
		"governance":     false,
	}

	for _, entry := range entries {
		if _, ok := seen[entry.ToolName]; ok {
			t.Fatalf("duplicate tool name found in catalog: %s", entry.ToolName)
		}
		seen[entry.ToolName] = struct{}{}
		if _, ok := domains[entry.Domain]; ok {
			domains[entry.Domain] = true
		}
	}

	for domain, found := range domains {
		if !found {
			t.Fatalf("expected catalog coverage for domain %s", domain)
		}
	}
}

func TestAllCatalogEntries_HaveNonEmptyDescriptions(t *testing.T) {
	for _, entry := range AllCatalogEntries() {
		if entry.Description == "" {
			t.Errorf("catalog entry %q (domain=%s) has empty description", entry.ToolName, entry.Domain)
		}
	}
}
