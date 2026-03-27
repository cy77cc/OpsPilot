// Package prometheus 提供 Prometheus HTTP API 客户端实现。
//
// 本文件实现指标推送到 Pushgateway 的功能，
// 用于将主机和集群状态指标推送到 Prometheus 生态系统。
package prometheus

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

// MetricsPusher 推送指标到 Pushgateway。
type MetricsPusher struct {
	gatewayURL string
	registry   *prometheus.Registry
	mu         sync.Mutex

	// 主机指标
	hostCPULoad        *prometheus.GaugeVec
	hostMemoryUsedMB   *prometheus.GaugeVec
	hostMemoryTotalMB  *prometheus.GaugeVec
	hostMemoryUsage    *prometheus.GaugeVec
	hostDiskUsage      *prometheus.GaugeVec
	hostInodeUsage     *prometheus.GaugeVec
	hostHealthState    *prometheus.GaugeVec
	hostConnectivity   *prometheus.GaugeVec
	hostCollectionTime *prometheus.GaugeVec

	// 集群指标
	clusterNodeCount    *prometheus.GaugeVec
	clusterNodeReady    *prometheus.GaugeVec
	clusterNodeNotReady *prometheus.GaugeVec
	clusterPodCount     *prometheus.GaugeVec
	clusterPodRunning   *prometheus.GaugeVec
	clusterPodPending   *prometheus.GaugeVec
	clusterPodFailed    *prometheus.GaugeVec
	clusterCollectTime  *prometheus.GaugeVec
}

// HostMetricSnapshot 主机指标快照。
type HostMetricSnapshot struct {
	HostID             uint64
	HostName           string
	HostIP             string
	CPULoad            float64
	MemoryUsedMB       int
	MemoryTotalMB      int
	DiskUsagePercent   float64
	InodeUsagePercent  float64
	HealthState        string // healthy, degraded, critical, unknown
	ConnectivityStatus string
}

// ClusterMetricSnapshot 集群指标快照。
type ClusterMetricSnapshot struct {
	ClusterID    uint64
	ClusterName  string
	NodeTotal    int
	NodeReady    int
	NodeNotReady int
	PodTotal     int
	PodRunning   int
	PodPending   int
	PodFailed    int
}

// NewMetricsPusher 创建指标推送器。
func NewMetricsPusher(gatewayURL string) (*MetricsPusher, error) {
	if gatewayURL == "" {
		return nil, fmt.Errorf("pushgateway url is empty")
	}

	p := &MetricsPusher{
		gatewayURL: gatewayURL,
		registry:   prometheus.NewRegistry(),
	}

	// 初始化主机指标
	p.hostCPULoad = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "host_cpu_load",
		Help: "Host CPU load average (1 minute)",
	}, []string{"host_id", "host_name", "host_ip"})

	p.hostMemoryUsedMB = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "host_memory_used_mb",
		Help: "Host memory used in MB",
	}, []string{"host_id", "host_name", "host_ip"})

	p.hostMemoryTotalMB = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "host_memory_total_mb",
		Help: "Host total memory in MB",
	}, []string{"host_id", "host_name", "host_ip"})

	p.hostMemoryUsage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "host_memory_usage_percent",
		Help: "Host memory usage percentage (0-100)",
	}, []string{"host_id", "host_name", "host_ip"})

	p.hostDiskUsage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "host_disk_usage_percent",
		Help: "Host disk usage percentage (0-100)",
	}, []string{"host_id", "host_name", "host_ip"})

	p.hostInodeUsage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "host_inode_usage_percent",
		Help: "Host inode usage percentage (0-100)",
	}, []string{"host_id", "host_name", "host_ip"})

	p.hostHealthState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "host_health_state",
		Help: "Host health state: 0=healthy, 1=degraded, 2=critical, 3=unknown",
	}, []string{"host_id", "host_name", "host_ip"})

	p.hostConnectivity = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "host_connectivity_status",
		Help: "Host connectivity status: 0=healthy, 1=degraded, 2=critical, 3=unknown",
	}, []string{"host_id", "host_name", "host_ip"})

	p.hostCollectionTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "host_collection_timestamp",
		Help: "Unix timestamp of last collection",
	}, []string{"host_id", "host_name", "host_ip"})

	// 初始化集群指标
	p.clusterNodeCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8s_cluster_node_count",
		Help: "Total number of nodes in the cluster",
	}, []string{"cluster_id", "cluster_name"})

	p.clusterNodeReady = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8s_cluster_node_ready",
		Help: "Number of ready nodes in the cluster",
	}, []string{"cluster_id", "cluster_name"})

	p.clusterNodeNotReady = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8s_cluster_node_not_ready",
		Help: "Number of not ready nodes in the cluster",
	}, []string{"cluster_id", "cluster_name"})

	p.clusterPodCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8s_cluster_pod_count",
		Help: "Total number of pods in the cluster",
	}, []string{"cluster_id", "cluster_name"})

	p.clusterPodRunning = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8s_cluster_pod_running",
		Help: "Number of running pods in the cluster",
	}, []string{"cluster_id", "cluster_name"})

	p.clusterPodPending = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8s_cluster_pod_pending",
		Help: "Number of pending pods in the cluster",
	}, []string{"cluster_id", "cluster_name"})

	p.clusterPodFailed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8s_cluster_pod_failed",
		Help: "Number of failed pods in the cluster",
	}, []string{"cluster_id", "cluster_name"})

	p.clusterCollectTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8s_cluster_collection_timestamp",
		Help: "Unix timestamp of last cluster collection",
	}, []string{"cluster_id", "cluster_name"})

	// 注册所有指标
	p.registry.MustRegister(
		p.hostCPULoad,
		p.hostMemoryUsedMB,
		p.hostMemoryTotalMB,
		p.hostMemoryUsage,
		p.hostDiskUsage,
		p.hostInodeUsage,
		p.hostHealthState,
		p.hostConnectivity,
		p.hostCollectionTime,
		p.clusterNodeCount,
		p.clusterNodeReady,
		p.clusterNodeNotReady,
		p.clusterPodCount,
		p.clusterPodRunning,
		p.clusterPodPending,
		p.clusterPodFailed,
		p.clusterCollectTime,
	)

	return p, nil
}

// PushHostMetrics 推送主机指标到 Pushgateway。
func (p *MetricsPusher) PushHostMetrics(ctx context.Context, snapshot HostMetricSnapshot) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	labels := prometheus.Labels{
		"host_id":   strconv.FormatUint(snapshot.HostID, 10),
		"host_name": snapshot.HostName,
		"host_ip":   snapshot.HostIP,
	}

	// 设置指标值
	p.hostCPULoad.With(labels).Set(snapshot.CPULoad)
	p.hostMemoryUsedMB.With(labels).Set(float64(snapshot.MemoryUsedMB))
	p.hostMemoryTotalMB.With(labels).Set(float64(snapshot.MemoryTotalMB))

	// 计算内存使用率
	if snapshot.MemoryTotalMB > 0 {
		usage := float64(snapshot.MemoryUsedMB) / float64(snapshot.MemoryTotalMB) * 100
		p.hostMemoryUsage.With(labels).Set(usage)
	}

	p.hostDiskUsage.With(labels).Set(snapshot.DiskUsagePercent)
	p.hostInodeUsage.With(labels).Set(snapshot.InodeUsagePercent)
	p.hostHealthState.With(labels).Set(healthStateToFloat(snapshot.HealthState))
	p.hostConnectivity.With(labels).Set(healthStateToFloat(snapshot.ConnectivityStatus))
	p.hostCollectionTime.With(labels).Set(float64(time.Now().Unix()))

	// 推送到 Pushgateway
	jobName := fmt.Sprintf("host_%d", snapshot.HostID)
	return push.New(p.gatewayURL, jobName).
		Gatherer(p.registry).
		PushContext(ctx)
}

// PushClusterMetrics 推送集群指标到 Pushgateway。
func (p *MetricsPusher) PushClusterMetrics(ctx context.Context, snapshot ClusterMetricSnapshot) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	labels := prometheus.Labels{
		"cluster_id":   strconv.FormatUint(snapshot.ClusterID, 10),
		"cluster_name": snapshot.ClusterName,
	}

	// 设置指标值
	p.clusterNodeCount.With(labels).Set(float64(snapshot.NodeTotal))
	p.clusterNodeReady.With(labels).Set(float64(snapshot.NodeReady))
	p.clusterNodeNotReady.With(labels).Set(float64(snapshot.NodeNotReady))
	p.clusterPodCount.With(labels).Set(float64(snapshot.PodTotal))
	p.clusterPodRunning.With(labels).Set(float64(snapshot.PodRunning))
	p.clusterPodPending.With(labels).Set(float64(snapshot.PodPending))
	p.clusterPodFailed.With(labels).Set(float64(snapshot.PodFailed))
	p.clusterCollectTime.With(labels).Set(float64(time.Now().Unix()))

	// 推送到 Pushgateway
	jobName := fmt.Sprintf("cluster_%d", snapshot.ClusterID)
	return push.New(p.gatewayURL, jobName).
		Gatherer(p.registry).
		PushContext(ctx)
}

// healthStateToFloat 将健康状态字符串转换为数值。
func healthStateToFloat(state string) float64 {
	switch state {
	case "healthy":
		return 0
	case "degraded":
		return 1
	case "critical":
		return 2
	default:
		return 3 // unknown
	}
}
