package tools

import "testing"

func TestCatalogSearchReturnsTopDomainMatches(t *testing.T) {
	catalog := NewCatalog([]ToolMetadata{
		{ToolName: "host_exec", Domain: "host", Capability: "command_execution"},
		{ToolName: "k8s_query", Domain: "kubernetes", Capability: "query"},
	})
	results := catalog.Search("host exec", 1, "")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Domain != "host" {
		t.Fatalf("expected host domain, got %q", results[0].Domain)
	}
	if results[0].ToolName != "host_exec" {
		t.Fatalf("expected host_exec result, got %q", results[0].ToolName)
	}
}

func TestCatalogSearchRespectsLimit(t *testing.T) {
	catalog := NewCatalog([]ToolMetadata{
		{ToolName: "k8s_query", Domain: "kubernetes", Capability: "query"},
		{ToolName: "k8s_logs", Domain: "kubernetes", Capability: "logs"},
		{ToolName: "k8s_events", Domain: "kubernetes", Capability: "events"},
	})
	results := catalog.Search("k8s", 2, "")
	if len(results) > 2 {
		t.Fatalf("expected at most 2 results, got %d", len(results))
	}
}

func TestCatalogSearchFiltersDomainBeforeLimit(t *testing.T) {
	catalog := NewCatalog([]ToolMetadata{
		{ToolName: "host_exec", Domain: "host", Capability: "command_execution"},
		{ToolName: "host_logs", Domain: "host", Capability: "logs"},
		{ToolName: "k8s_query", Domain: "kubernetes", Capability: "query"},
	})
	results := catalog.Search("query", 5, "kubernetes")
	if len(results) != 1 || results[0].ToolName != "k8s_query" {
		t.Fatalf("expected kubernetes-only result, got %#v", results)
	}
}
