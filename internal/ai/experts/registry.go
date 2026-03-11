// Package experts 提供专家注册和管理功能。
//
// 本文件实现专家注册表，用于管理和发现所有可用的运维专家。
// 专家是具备特定领域工具集的智能代理，如主机运维、K8s 集群管理、服务部署等。
package experts

import (
	"sort"

	deliveryexpert "github.com/cy77cc/OpsPilot/internal/ai/experts/delivery"
	hostopsexpert "github.com/cy77cc/OpsPilot/internal/ai/experts/hostops"
	k8sexpert "github.com/cy77cc/OpsPilot/internal/ai/experts/k8s"
	observabilityexpert "github.com/cy77cc/OpsPilot/internal/ai/experts/observability"
	serviceexpert "github.com/cy77cc/OpsPilot/internal/ai/experts/service"
	expertspec "github.com/cy77cc/OpsPilot/internal/ai/experts/spec"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

// Registry 专家注册表，管理所有可用的专家实例。
//
// 注册表提供专家的注册、查找和列表功能，是专家系统的核心容器。
type Registry struct {
	experts map[string]expertspec.Expert // 专家映射，按名称索引
}

// NewRegistry 创建新的专家注册表。
//
// 参数:
//   - experts: 初始专家列表
//
// 返回:
//   - 包含指定专家的注册表实例
func NewRegistry(experts ...expertspec.Expert) *Registry {
	items := make(map[string]expertspec.Expert, len(experts))
	for _, exp := range experts {
		if exp == nil {
			continue
		}
		items[exp.Name()] = exp
	}
	return &Registry{experts: items}
}

// DefaultRegistry 创建默认的专家注册表，包含所有内置专家。
//
// 内置专家包括:
//   - hostops: 主机运维专家
//   - k8s: Kubernetes 集群专家
//   - service: 服务管理专家
//   - delivery: 交付/CI-CD 专家
//   - observability: 可观测性专家
//
// 参数:
//   - deps: 平台依赖（数据库、配置、客户端等）
func DefaultRegistry(deps common.PlatformDeps) *Registry {
	return NewRegistry(
		hostopsexpert.New(deps),
		k8sexpert.New(deps),
		serviceexpert.New(deps),
		deliveryexpert.New(deps),
		observabilityexpert.New(deps),
	)
}

// Get 根据名称获取专家实例。
//
// 参数:
//   - name: 专家名称（如 "hostops", "k8s"）
//
// 返回:
//   - 专家实例和是否存在的布尔值
func (r *Registry) Get(name string) (expertspec.Expert, bool) {
	if r == nil {
		return nil, false
	}
	exp, ok := r.experts[name]
	return exp, ok
}

// List 获取所有注册的专家列表，按名称排序。
//
// 返回:
//   - 按名称排序的专家列表
func (r *Registry) List() []expertspec.Expert {
	if r == nil {
		return nil
	}
	keys := make([]string, 0, len(r.experts))
	for key := range r.experts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]expertspec.Expert, 0, len(keys))
	for _, key := range keys {
		out = append(out, r.experts[key])
	}
	return out
}

// ToolDirectory 获取所有专家的工具目录，用于规划器决策。
//
// 返回:
//   - 工具导出列表，每个条目包含专家名称、描述和能力清单
func (r *Registry) ToolDirectory() []expertspec.ToolExport {
	exps := r.List()
	out := make([]expertspec.ToolExport, 0, len(exps))
	for _, exp := range exps {
		out = append(out, exp.AsTool())
	}
	return out
}
