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
// 覆盖同名适配器，用于测试或更新实现。
func Register(p CloudProvider) {
	globalRegistry.Register(p)
}

// GetProvider 从全局注册表获取指定云厂商适配器。
//
// 参数:
//   - name: 云厂商标识（如 "volcengine"）
//
// 返回:
//   - 找到返回适配器实例
//   - 未找到返回 ErrProviderNotFound 错误
func GetProvider(name string) (CloudProvider, error) {
	return globalRegistry.GetProvider(name)
}

// ListProviders 列出全局注册表中所有云厂商信息。
//
// 返回所有已注册的云厂商列表，用于前端下拉选项展示。
func ListProviders() []ProviderInfo {
	return globalRegistry.ListProviders()
}

// Register 注册云厂商适配器。
func (r *Registry) Register(p CloudProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// GetProvider 获取指定云厂商适配器。
func (r *Registry) GetProvider(name string) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.providers[name]; ok {
		return p, nil
	}
	return nil, ErrProviderNotFound
}

// ListProviders 列出所有云厂商信息。
func (r *Registry) ListProviders() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]ProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, ProviderInfo{
			Name:        p.Name(),
			DisplayName: p.DisplayName(),
		})
	}
	return list
}
