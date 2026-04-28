package tools

import (
	"context"
	"strings"
	"testing"
)

func TestSearchToolsTool_FindsByName(t *testing.T) {
	catalog := NewCatalog([]ToolMetadata{
		{ToolName: "k8s_query", Description: "Query Kubernetes resources with SQL-like syntax", Domain: "kubernetes"},
		{ToolName: "k8s_list_resources", Description: "List Kubernetes resources in a namespace", Domain: "kubernetes"},
		{ToolName: "monitor_metric", Description: "Query Prometheus metrics", Domain: "monitoring"},
	})

	searchTool, err := NewSearchToolsTool(catalog)
	if err != nil {
		t.Fatalf("unexpected error creating search tool: %v", err)
	}

	info, _ := searchTool.Info(context.Background())
	if info.Name != "search_tools" {
		t.Errorf("expected tool name search_tools, got %q", info.Name)
	}

	result, err := searchTool.InvokableRun(context.Background(), `{"query": "kubernetes"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "k8s_query") {
		t.Errorf("expected result to contain k8s_query, got %q", result)
	}
	if strings.Contains(result, "monitor_metric") {
		t.Errorf("kubernetes query should not return monitoring tools")
	}
}

func TestSearchToolsTool_FindsByDescription(t *testing.T) {
	catalog := NewCatalog([]ToolMetadata{
		{ToolName: "host_exec", Description: "Execute shell commands on remote hosts", Domain: "host"},
		{ToolName: "cicd_pipeline_trigger", Description: "Trigger a CI/CD pipeline run", Domain: "cicd"},
	})

	searchTool, err := NewSearchToolsTool(catalog)
	if err != nil {
		t.Fatalf("unexpected error creating search tool: %v", err)
	}

	result, err := searchTool.InvokableRun(context.Background(), `{"query": "pipeline"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "cicd_pipeline_trigger") {
		t.Errorf("expected cicd_pipeline_trigger in results")
	}
	if strings.Contains(result, "host_exec") {
		t.Errorf("pipeline query should not return host_exec")
	}
}

func TestSearchToolsTool_EmptyQuery(t *testing.T) {
	catalog := NewCatalog([]ToolMetadata{
		{ToolName: "k8s_query", Description: "Query Kubernetes resources", Domain: "kubernetes"},
	})

	searchTool, err := NewSearchToolsTool(catalog)
	if err != nil {
		t.Fatalf("unexpected error creating search tool: %v", err)
	}

	result, err := searchTool.InvokableRun(context.Background(), `{"query": ""}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "k8s_query") {
		t.Errorf("empty query should return all tools")
	}
}

func TestSearchToolsTool_NoMatch(t *testing.T) {
	catalog := NewCatalog([]ToolMetadata{
		{ToolName: "k8s_query", Description: "Query Kubernetes resources", Domain: "kubernetes"},
	})

	searchTool, err := NewSearchToolsTool(catalog)
	if err != nil {
		t.Fatalf("unexpected error creating search tool: %v", err)
	}

	result, err := searchTool.InvokableRun(context.Background(), `{"query": "nonexistent_xyz"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "k8s_query") {
		t.Errorf("non-matching query should not return results")
	}
}

func TestCatalog_Search(t *testing.T) {
	catalog := NewCatalog([]ToolMetadata{
		{ToolName: "k8s_query", Description: "Query Kubernetes resources", Domain: "kubernetes"},
		{ToolName: "k8s_list", Description: "List Kubernetes pods", Domain: "kubernetes"},
		{ToolName: "monitor_alert", Description: "Get monitoring alerts", Domain: "monitoring"},
	})

	results := catalog.Search("kubernetes", 10, "")
	if len(results) != 2 {
		t.Errorf("expected 2 kubernetes results, got %d", len(results))
	}

	results = catalog.Search("monitoring", 10, "")
	if len(results) != 1 {
		t.Errorf("expected 1 monitoring result, got %d", len(results))
	}
}

func TestCatalog_All(t *testing.T) {
	entries := []ToolMetadata{
		{ToolName: "k8s_query", Description: "Query Kubernetes resources", Domain: "kubernetes"},
		{ToolName: "monitor_alert", Description: "Get monitoring alerts", Domain: "monitoring"},
	}
	catalog := NewCatalog(entries)

	all := catalog.All()
	if len(all) != 2 {
		t.Errorf("expected 2 entries, got %d", len(all))
	}

	// Verify it's a copy — modifying the returned slice doesn't affect the catalog.
	all[0].ToolName = "modified"
	if catalog.All()[0].ToolName == "modified" {
		t.Error("All() should return a copy, not a reference to internal slice")
	}
}

func TestCatalog_Len(t *testing.T) {
	catalog := NewCatalog([]ToolMetadata{
		{ToolName: "a", Domain: "x"},
		{ToolName: "b", Domain: "y"},
		{ToolName: "c", Domain: "z"},
	})
	if got := catalog.Len(); got != 3 {
		t.Errorf("expected Len() == 3, got %d", got)
	}

	empty := NewCatalog(nil)
	if got := empty.Len(); got != 0 {
		t.Errorf("expected empty Len() == 0, got %d", got)
	}
}
