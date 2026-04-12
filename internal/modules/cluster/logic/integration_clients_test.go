package logic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	clustersecurity "github.com/cy77cc/OpsPilot/internal/modules/cluster/domain/security"
	clusterintegration "github.com/cy77cc/OpsPilot/internal/modules/cluster/integration"
)

func TestHarborClient_ListProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/projects" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"core"},{"name":"ai"}]`))
	}))
	defer server.Close()

	client := clusterintegration.NewHTTPHarborClient(server.URL, "")
	projects, err := client.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("list harbor projects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
}

func TestTrivyClient_ScanImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scan" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"summary":{"critical":1,"high":2}}`))
	}))
	defer server.Close()

	client := clusterintegration.NewHTTPTrivyClient(server.URL)
	result, err := client.ScanImage(context.Background(), "registry.local/nginx:1.0")
	if err != nil {
		t.Fatalf("scan image: %v", err)
	}
	if result.Summary.Critical != 1 {
		t.Fatalf("expected critical=1, got %d", result.Summary.Critical)
	}
}

func TestArgoCDClient_SyncApplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/applications/payments/sync" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"succeeded","revision":"abc123"}`))
	}))
	defer server.Close()

	client := clusterintegration.NewHTTPArgoCDClient(server.URL, "")
	result, err := client.Sync(context.Background(), "payments")
	if err != nil {
		t.Fatalf("sync application: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("expected succeeded, got %s", result.Status)
	}
}

func TestRuntimeIngest_ParseFalcoAndTetragon(t *testing.T) {
	falcoRaw := []byte(`{"rule":"Terminal shell in container","priority":"Critical","output_fields":{"k8s.ns.name":"prod","k8s.pod.name":"api-1"}}`)
	falcoEvent, err := clustersecurity.ParseFalcoEvent(falcoRaw)
	if err != nil {
		t.Fatalf("parse falco event: %v", err)
	}
	if falcoEvent.Severity != "critical" {
		t.Fatalf("expected falco severity critical, got %s", falcoEvent.Severity)
	}

	tetragonRaw := []byte(`{"policy_name":"exec-detect","severity":"high","namespace":"prod","pod":"api-1"}`)
	tetragonEvent, err := clustersecurity.ParseTetragonEvent(tetragonRaw)
	if err != nil {
		t.Fatalf("parse tetragon event: %v", err)
	}
	if tetragonEvent.RuleID != "exec-detect" {
		t.Fatalf("expected rule exec-detect, got %s", tetragonEvent.RuleID)
	}
}
