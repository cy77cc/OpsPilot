// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件定义集群服务的缓存策略，包括缓存键模式和 TTL 配置。
package cluster

import "time"

// 缓存键前缀。
const (
	clusterCachePrefix = "cluster" // 集群缓存键前缀

	CacheTTLShort  = 20 * time.Second // 短缓存 TTL (列表类数据)
	CacheTTLMedium = 60 * time.Second // 中等缓存 TTL (详情类数据)
	CacheTTLLong   = 2 * time.Minute  // 长缓存 TTL (配置类数据)
)

// CachePolicy 缓存策略配置。
type CachePolicy struct {
	KeyPattern       string        // 缓存键模式
	TTL              time.Duration // 缓存过期时间
	InvalidatesOnOps []string      // 触发失效的操作列表
}

// ClusterPhase1CachePolicies 定义集群服务的缓存策略映射。
//
// 策略说明:
//   - clusters.list: 集群列表，短 TTL
//   - clusters.detail: 集群详情，中等 TTL
//   - clusters.nodes: 集群节点列表，短 TTL
//   - clusters.bootstrap_profiles: 引导配置列表，长 TTL
var ClusterPhase1CachePolicies = map[string]CachePolicy{
	"clusters.list": {
		KeyPattern:       "cluster:list:{status}:{source}",
		TTL:              CacheTTLShort,
		InvalidatesOnOps: []string{"cluster.create", "cluster.update", "cluster.delete", "cluster.import", "cluster.sync"},
	},
	"clusters.detail": {
		KeyPattern:       "cluster:detail:{id}",
		TTL:              CacheTTLMedium,
		InvalidatesOnOps: []string{"cluster.update", "cluster.delete", "cluster.import", "cluster.sync"},
	},
	"clusters.nodes": {
		KeyPattern:       "cluster:nodes:{id}",
		TTL:              CacheTTLShort,
		InvalidatesOnOps: []string{"cluster.addNode", "cluster.removeNode", "cluster.sync", "cluster.import"},
	},
	"clusters.bootstrap_profiles": {
		KeyPattern:       "cluster:bootstrap:profiles:list",
		TTL:              CacheTTLLong,
		InvalidatesOnOps: []string{"cluster.bootstrap_profile.create", "cluster.bootstrap_profile.update", "cluster.bootstrap_profile.delete"},
	},
}

// CacheKeyClusterList 生成集群列表缓存键。
//
// 参数:
//   - status: 状态筛选
//   - source: 来源筛选
//
// 返回: 缓存键字符串
func CacheKeyClusterList(status, source string) string {
	return clusterCachePrefix + ":list:" + status + ":" + source
}

// CacheKeyClusterDetail 生成集群详情缓存键。
//
// 参数:
//   - id: 集群 ID
//
// 返回: 缓存键字符串
func CacheKeyClusterDetail(id uint) string {
	return clusterCachePrefix + ":detail:" + itoa(id)
}

// CacheKeyClusterNodes 生成集群节点列表缓存键。
//
// 参数:
//   - id: 集群 ID
//
// 返回: 缓存键字符串
func CacheKeyClusterNodes(id uint) string {
	return clusterCachePrefix + ":nodes:" + itoa(id)
}

// CacheKeyBootstrapProfiles 生成引导配置列表缓存键。
//
// 返回: 缓存键字符串
func CacheKeyBootstrapProfiles() string {
	return clusterCachePrefix + ":bootstrap:profiles:list"
}

// itoa 将 uint 转换为字符串 (无分配版本)。
//
// 参数:
//   - v: 待转换值
//
// 返回: 字符串表示
func itoa(v uint) string {
	if v == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	return string(buf[i:])
}
