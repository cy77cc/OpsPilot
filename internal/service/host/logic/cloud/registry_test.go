// Package cloud_test 测试云厂商注册表功能。
package cloud_test

import (
	"context"
	"testing"

	cloudpkg "github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud/volcengine"
)

// mockProvider 用于测试的 Mock 适配器。
type mockProvider struct {
	name        string
	displayName string
}

func (m *mockProvider) Name() string        { return m.name }
func (m *mockProvider) DisplayName() string { return m.displayName }
func (m *mockProvider) ValidateCredential(ctx context.Context, ak, sk, region string) error {
	return nil
}
func (m *mockProvider) ListInstances(ctx context.Context, req cloudpkg.ListInstancesRequest) ([]cloudpkg.CloudInstance, error) {
	return nil, nil
}

func TestRegistry_RegisterAndGetProvider(t *testing.T) {
	// 使用新的注册表实例，避免污染全局注册表
	registry := cloudpkg.NewRegistry()

	// 注册火山云适配器
	provider := volcengine.New()
	registry.Register(provider)

	// 测试获取已注册的适配器
	got, err := registry.GetProvider("volcengine")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}
	if got.Name() != "volcengine" {
		t.Errorf("Name = %s, want volcengine", got.Name())
	}
	if got.DisplayName() != "火山云" {
		t.Errorf("DisplayName = %s, want 火山云", got.DisplayName())
	}

	// 测试获取不存在的适配器
	_, err = registry.GetProvider("not-exist")
	if err == nil {
		t.Error("GetProvider should return error for non-existent provider")
	}
}

func TestRegistry_ListProviders(t *testing.T) {
	registry := cloudpkg.NewRegistry()

	// 注册多个适配器
	registry.Register(volcengine.New())
	registry.Register(cloudpkg.NewMockProvider("alicloud", "阿里云"))
	registry.Register(cloudpkg.NewMockProvider("tencent", "腾讯云"))

	// 获取列表
	list := registry.ListProviders()

	if len(list) != 3 {
		t.Fatalf("ListProviders returned %d providers, want 3", len(list))
	}

	// 验证列表包含所有已注册的云厂商
	providerMap := make(map[string]string)
	for _, p := range list {
		providerMap[p.Name] = p.DisplayName
	}

	if providerMap["volcengine"] != "火山云" {
		t.Error("Missing volcengine provider")
	}
	if providerMap["alicloud"] != "阿里云" {
		t.Error("Missing alicloud provider")
	}
	if providerMap["tencent"] != "腾讯云" {
		t.Error("Missing tencent provider")
	}
}

func TestGlobalRegistry(t *testing.T) {
	// 手动注册适配器（因为 init 函数只在主程序运行时执行）
	cloudpkg.Register(volcengine.New())
	cloudpkg.Register(cloudpkg.NewMockProvider("alicloud", "阿里云"))
	cloudpkg.Register(cloudpkg.NewMockProvider("tencent", "腾讯云"))

	// 测试全局注册表函数
	providers := cloudpkg.ListProviders()

	// 验证全局注册表不为空
	if len(providers) == 0 {
		t.Error("Global registry should not be empty")
	}

	// 验证包含火山云
	found := false
	for _, p := range providers {
		if p.Name == "volcengine" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Global registry should contain volcengine provider")
	}
}
