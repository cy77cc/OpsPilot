// Package kubernetes 提供 Kubernetes 集群操作相关的工具实现。
//
// 本文件测试 K8s 写操作工具的输入验证逻辑。
package kubernetes

import (
	"testing"
)

// =============================================================================
// 输入验证测试
// =============================================================================

func TestValidateScaleInput(t *testing.T) {
	tests := []struct {
		name    string
		input   *K8sScaleDeploymentInput
		wantErr bool
	}{
		{
			name: "valid input",
			input: &K8sScaleDeploymentInput{
				ClusterID: 1,
				Namespace: "default",
				Name:      "nginx",
				Replicas:  3,
			},
			wantErr: false,
		},
		{
			name: "zero replicas allowed",
			input: &K8sScaleDeploymentInput{
				ClusterID: 1,
				Namespace: "default",
				Name:      "nginx",
				Replicas:  0,
			},
			wantErr: false,
		},
		{
			name: "missing cluster_id",
			input: &K8sScaleDeploymentInput{
				Namespace: "default",
				Name:      "nginx",
				Replicas:  3,
			},
			wantErr: true,
		},
		{
			name: "missing namespace",
			input: &K8sScaleDeploymentInput{
				ClusterID: 1,
				Name:      "nginx",
				Replicas:  3,
			},
			wantErr: true,
		},
		{
			name: "missing name",
			input: &K8sScaleDeploymentInput{
				ClusterID: 1,
				Namespace: "default",
				Replicas:  3,
			},
			wantErr: true,
		},
		{
			name: "negative replicas not allowed",
			input: &K8sScaleDeploymentInput{
				ClusterID: 1,
				Namespace: "default",
				Name:      "nginx",
				Replicas:  -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScaleInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateScaleInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNamespaceNameInput(t *testing.T) {
	tests := []struct {
		name      string
		clusterID int
		namespace string
		nameInput string
		wantErr   bool
	}{
		{
			name:      "valid input",
			clusterID: 1,
			namespace: "default",
			nameInput: "nginx",
			wantErr:   false,
		},
		{
			name:      "missing cluster_id",
			clusterID: 0,
			namespace: "default",
			nameInput: "nginx",
			wantErr:   true,
		},
		{
			name:      "missing namespace",
			clusterID: 1,
			namespace: "",
			nameInput: "nginx",
			wantErr:   true,
		},
		{
			name:      "missing name",
			clusterID: 1,
			namespace: "default",
			nameInput: "",
			wantErr:   true,
		},
		{
			name:      "whitespace namespace",
			clusterID: 1,
			namespace: "   ",
			nameInput: "nginx",
			wantErr:   true,
		},
		{
			name:      "whitespace name",
			clusterID: 1,
			namespace: "default",
			nameInput: "   ",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNamespaceNameInput(tt.clusterID, tt.namespace, tt.nameInput)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNamespaceNameInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// 工具创建测试
// =============================================================================

func TestK8sScaleDeploymentCreation(t *testing.T) {
	tool := K8sScaleDeployment(nil)
	if tool == nil {
		t.Error("K8sScaleDeployment() returned nil")
	}
}

func TestK8sRestartDeploymentCreation(t *testing.T) {
	tool := K8sRestartDeployment(nil)
	if tool == nil {
		t.Error("K8sRestartDeployment() returned nil")
	}
}

func TestK8sDeletePodCreation(t *testing.T) {
	tool := K8sDeletePod(nil)
	if tool == nil {
		t.Error("K8sDeletePod() returned nil")
	}
}

func TestK8sRollbackDeploymentCreation(t *testing.T) {
	tool := K8sRollbackDeployment(nil)
	if tool == nil {
		t.Error("K8sRollbackDeployment() returned nil")
	}
}

func TestK8sDeleteDeploymentCreation(t *testing.T) {
	tool := K8sDeleteDeployment(nil)
	if tool == nil {
		t.Error("K8sDeleteDeployment() returned nil")
	}
}

func TestNewKubernetesWriteTools(t *testing.T) {
	tools := NewKubernetesWriteTools(nil)
	if len(tools) != 5 {
		t.Errorf("NewKubernetesWriteTools() returned %d tools, want 5", len(tools))
	}
}
