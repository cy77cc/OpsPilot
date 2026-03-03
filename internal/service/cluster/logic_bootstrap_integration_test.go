package cluster

import (
	"strings"
	"testing"

	"github.com/cy77cc/k8s-manage/internal/model"
)

func TestBuildBootstrapSteps_ContainsNewPrechecksAndEndpointSteps(t *testing.T) {
	steps := buildBootstrapSteps("1.28.2")
	names := make(map[string]struct{}, len(steps))
	for _, step := range steps {
		names[step.Name] = struct{}{}
	}
	for _, required := range []string{"bootstrap-prechecks", "vip-provider", "endpoint-health"} {
		if _, ok := names[required]; !ok {
			t.Fatalf("missing step: %s", required)
		}
	}
}

func TestBuildKubeadmInitConfigYAML_MirrorAndVIP(t *testing.T) {
	task := &model.ClusterBootstrapTask{
		Name:                 "prod-a",
		K8sVersion:           "1.28.3",
		PodCIDR:              "10.244.0.0/16",
		ServiceCIDR:          "10.96.0.0/12",
		ControlPlaneEndpoint: "10.0.0.10:6443",
		ImageRepository:      "registry.aliyuncs.com/google_containers",
		EtcdMode:             "stacked",
	}

	yml, err := buildKubeadmInitConfigYAML(task, "172.16.1.11")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustContain := []string{
		"kubernetesVersion: v1.28.3",
		"controlPlaneEndpoint: 10.0.0.10:6443",
		"imageRepository: registry.aliyuncs.com/google_containers",
		"etcd:",
		"local: {}",
	}
	for _, s := range mustContain {
		if !strings.Contains(yml, s) {
			t.Fatalf("expected yaml contains %q, got:\n%s", s, yml)
		}
	}
}

func TestBuildKubeadmInitConfigYAML_ExternalEtcd(t *testing.T) {
	task := &model.ClusterBootstrapTask{
		Name:             "prod-b",
		K8sVersion:       "1.28.3",
		PodCIDR:          "10.244.0.0/16",
		ServiceCIDR:      "10.96.0.0/12",
		EtcdMode:         "external",
		ExternalEtcdJSON: `{"endpoints":["https://10.0.0.21:2379"],"ca_cert":"-----BEGIN CERTIFICATE-----\\nca\\n-----END CERTIFICATE-----","cert":"-----BEGIN CERTIFICATE-----\\ncert\\n-----END CERTIFICATE-----","key":"-----BEGIN PRIVATE KEY-----\\nkey\\n-----END PRIVATE KEY-----"}`,
	}
	yml, err := buildKubeadmInitConfigYAML(task, "172.16.1.11")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(yml, "external:") || !strings.Contains(yml, "https://10.0.0.21:2379") {
		t.Fatalf("expected external etcd config in yaml, got:\n%s", yml)
	}
}
