// Package cloud 提供云厂商主机导入的统一接口和适配器管理。
package cloud

import (
	"context"
	"strings"
)

// MockProvider 云厂商 Mock 适配器。
//
// 用于阿里云和腾讯云的占位实现，返回模拟数据。
// 后续可替换为真实的 SDK 集成。
type MockProvider struct {
	// name 云厂商标识。
	name string

	// displayName 云厂商显示名称。
	displayName string
}

// NewMockProvider 创建 Mock 适配器。
//
// 参数:
//   - name: 云厂商标识（如 "alicloud"、"tencent"）
//   - displayName: 显示名称（如 "阿里云"、"腾讯云"）
func NewMockProvider(name, displayName string) *MockProvider {
	return &MockProvider{
		name:        name,
		displayName: displayName,
	}
}

// Name 返回云厂商标识。
func (p *MockProvider) Name() string {
	return p.name
}

// DisplayName 返回云厂商显示名称。
func (p *MockProvider) DisplayName() string {
	return p.displayName
}

// ValidateCredential 验证凭证（Mock 实现）。
//
// Mock 实现始终返回成功，仅验证参数非空。
func (p *MockProvider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
	return nil
}

// ListInstances 查询实例列表（Mock 实现）。
//
// 返回模拟的实例数据，支持关键词过滤。
func (p *MockProvider) ListInstances(ctx context.Context, req ListInstancesRequest) ([]CloudInstance, error) {
	// 默认地域
	region := req.Region
	if region == "" {
		region = "cn-hangzhou"
	}

	// 模拟实例数据
	instances := []CloudInstance{
		{
			InstanceID: p.name + "-i-001",
			Name:       p.name + "-web-01",
			IP:         "10.0.10.11",
			PrivateIP:  "10.0.10.11",
			Region:     region,
			Zone:       region + "-a",
			Status:     "running",
			OS:         "Ubuntu 22.04",
			CPU:        4,
			MemoryMB:   8192,
			DiskGB:     100,
		},
		{
			InstanceID: p.name + "-i-002",
			Name:       p.name + "-api-01",
			IP:         "10.0.10.12",
			PrivateIP:  "10.0.10.12",
			Region:     region,
			Zone:       region + "-a",
			Status:     "running",
			OS:         "CentOS 7",
			CPU:        8,
			MemoryMB:   16384,
			DiskGB:     200,
		},
	}

	// 关键词过滤
	if kw := strings.ToLower(strings.TrimSpace(req.Keyword)); kw != "" {
		filtered := make([]CloudInstance, 0)
		for _, inst := range instances {
			if strings.Contains(strings.ToLower(inst.Name), kw) ||
				strings.Contains(strings.ToLower(inst.InstanceID), kw) ||
				strings.Contains(inst.IP, kw) {
				filtered = append(filtered, inst)
			}
		}
		return filtered, nil
	}

	return instances, nil
}
