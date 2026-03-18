// Package platform 提供平台资源发现相关的工具实现。
//
// 本文件测试资源发现工具的创建逻辑。
package platform

import (
	"testing"
)

// =============================================================================
// 工具创建测试
// =============================================================================

func TestPlatformDiscoverResourcesCreation(t *testing.T) {
	tool := PlatformDiscoverResources(nil)
	if tool == nil {
		t.Error("PlatformDiscoverResources() returned nil")
	}
}

// =============================================================================
// 输入验证测试
// =============================================================================

func TestPlatformDiscoverInputValidation(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		clusterID    int
	}{
		{"empty resource type (overview)", "", 0},
		{"clusters", "clusters", 0},
		{"hosts", "hosts", 0},
		{"services", "services", 0},
		{"namespaces without cluster_id", "namespaces", 0},
		{"namespaces with cluster_id", "namespaces", 1},
		{"metrics", "metrics", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the input struct can be created
			input := &PlatformDiscoverInput{
				ResourceType: tt.resourceType,
				ClusterID:    tt.clusterID,
			}
			if input.ResourceType != tt.resourceType {
				t.Errorf("ResourceType = %v, want %v", input.ResourceType, tt.resourceType)
			}
			if input.ClusterID != tt.clusterID {
				t.Errorf("ClusterID = %v, want %v", input.ClusterID, tt.clusterID)
			}
		})
	}
}
