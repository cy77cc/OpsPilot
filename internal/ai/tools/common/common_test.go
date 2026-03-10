package common

import "testing"

func TestResolveK8sClientRequiresExplicitClusterID(t *testing.T) {
	cli, source, err := ResolveK8sClient(PlatformDeps{}, map[string]any{})
	if err == nil {
		t.Fatalf("ResolveK8sClient() error = nil, want error")
	}
	if cli != nil {
		t.Fatalf("client = %#v, want nil", cli)
	}
	if source != "missing_cluster_id" {
		t.Fatalf("source = %q, want %q", source, "missing_cluster_id")
	}
	if err.Error() != "k8s client unavailable: cluster_id is required" {
		t.Fatalf("error = %q", err.Error())
	}
}
