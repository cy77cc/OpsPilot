// Package cloud 提供云厂商主机导入的统一接口和适配器管理。
package cloud

import (
	"errors"
	"sync"
)

// ErrProviderNotFound 云厂商适配器未找到错误。
var ErrProviderNotFound = errors.New("cloud provider not found")

// Registry 云厂商适配器注册表。
//
// 管理所有已注册的云厂商适配器，支持动态注册和查询。
// 使用读写锁保证并发安全。
type Registry struct {
	// providers 已注册的云厂商适配器映射。
	providers map[string]CloudProvider

	// mu 读写锁，保护 providers 并发访问。
	mu sync.RWMutex
}

// globalRegistry 全局注册表实例。
//
// 应用启动时通过 Register 注册所有云厂商适配器，
// 运行时通过 GetProvider 和 ListProviders 访问。
var globalRegistry = NewRegistry()

// NewRegistry 创建新的注册表实例。
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]CloudProvider),
	}
}

// Register 注册云厂商适配器到全局注册表。
//
// 参数:
//   - p: 要注册的云厂商适配器
//
// 注册键格式: provider:product_type (如 "ucloud:uhost", "ucloud:ulighthost")
func Register(p CloudProvider) {
	globalRegistry.Register(p)
}

// GetProvider 从全局注册表获取指定云厂商适配器。
//
// 参数:
//   - key: 适配器键，格式为 "provider:product_type" 或 "provider" (向后兼容)
//
// 返回适配器实例或 ErrProviderNotFound 错误
func GetProvider(key string) (CloudProvider, error) {
	return globalRegistry.GetProvider(key)
}

// ListProviders 列出全局注册表中所有云厂商信息。
func ListProviders() []ProviderInfo {
	return globalRegistry.ListProviders()
}

// Register 注册云厂商适配器。
func (r *Registry) Register(p CloudProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// 使用 provider:product_type 作为键
	key := p.Name() + ":" + p.ProductType()
	r.providers[key] = p
}

// GetProvider 获取指定云厂商适配器。
func (r *Registry) GetProvider(key string) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 首先尝试精确匹配 provider:product_type
	if p, ok := r.providers[key]; ok {
		return p, nil
	}

	// 向后兼容：如果 key 不包含 ":"，尝试查找默认产品类型
	// 例如 "ucloud" -> 查找 "ucloud:uhost"
	if !containsColon(key) {
		defaultKey := key + ":uhost"
		if p, ok := r.providers[defaultKey]; ok {
			return p, nil
		}
		defaultKey = key + ":ecs"
		if p, ok := r.providers[defaultKey]; ok {
			return p, nil
		}
	}

	return nil, ErrProviderNotFound
}

// containsColon 检查字符串是否包含冒号。
func containsColon(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return true
		}
	}
	return false
}

// ListProviders 列出所有云厂商信息。
func (r *Registry) ListProviders() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]ProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, ProviderInfo{
			Name:            p.Name(),
			DisplayName:     p.DisplayName(),
			ProductType:     p.ProductType(),
			ProductTypeName: p.ProductTypeName(),
		})
	}
	return list
}