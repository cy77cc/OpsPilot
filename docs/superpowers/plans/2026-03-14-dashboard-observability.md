# Dashboard Observability Enhancement Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enhance the main dashboard with corrected AI activity data, cluster resource monitoring, operation status, and workload health overview.

**Architecture:** Add database models for caching K8s resource data, implement collectors for periodic sync, extend the Overview API with new data dimensions, and update the frontend layout with new cards.

**Tech Stack:** Go 1.26 + Gin + GORM, React 19 + Ant Design 6, Kubernetes client-go, Prometheus

---

## File Structure

### Backend (Go)

| File | Purpose |
|------|---------|
| `internal/model/dashboard_cache.go` | New models: ClusterResourceSnapshot, K8sWorkloadStats, K8sIssuePod |
| `internal/service/dashboard/collector.go` | Data collector for K8s resources |
| `internal/service/dashboard/logic.go` | Modify: refactor GetOverview with new dimensions |
| `api/dashboard/v1/dashboard.go` | Modify: add new response types |
| `storage/migrations/20260314_000037_dashboard_cache_tables.sql` | New: migration for cache tables |

### Frontend (React)

| File | Purpose |
|------|---------|
| `web/src/api/modules/dashboard.ts` | Modify: add new types |
| `web/src/components/Dashboard/WorkloadHealthCard.tsx` | New: workload health card |
| `web/src/components/Dashboard/ClusterResourceCard.tsx` | New: cluster resource card |
| `web/src/components/Dashboard/OperationsCard.tsx` | New: operations status card |
| `web/src/pages/Dashboard/Dashboard.tsx` | Modify: update layout |

---

## Chunk 1: Data Models and Migration

### Task 1.1: Create Dashboard Cache Models

**Files:**
- Create: `internal/model/dashboard_cache.go`

- [ ] **Step 1: Write the model file**

```go
// Package model 提供数据库模型定义。
//
// 本文件定义主控台可观测性缓存相关的数据模型。
package model

import "time"

// ClusterResourceSnapshot 集群资源快照表，存储定期采集的集群资源使用数据。
//
// 表名: cluster_resource_snapshots
// 用途: 缓存集群 CPU/内存/Pod 资源数据，避免频繁调用 K8s API
type ClusterResourceSnapshot struct {
	ID                 uint64    `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	ClusterID          uint      `gorm:"column:cluster_id;not null;index:idx_cluster_collected,priority:1" json:"cluster_id"`
	CPUAllocatableCores float64  `gorm:"column:cpu_allocatable_cores;type:decimal(10,2);not null;default:0" json:"cpu_allocatable_cores"`
	CPURequestedCores  float64  `gorm:"column:cpu_requested_cores;type:decimal(10,2);not null;default:0" json:"cpu_requested_cores"`
	CPULimitCores      float64  `gorm:"column:cpu_limit_cores;type:decimal(10,2);not null;default:0" json:"cpu_limit_cores"`
	CPUUsageCores      float64  `gorm:"column:cpu_usage_cores;type:decimal(10,2);not null;default:0" json:"cpu_usage_cores"`
	MemoryAllocatableMB int64   `gorm:"column:memory_allocatable_mb;not null;default:0" json:"memory_allocatable_mb"`
	MemoryRequestedMB   int64   `gorm:"column:memory_requested_mb;not null;default:0" json:"memory_requested_mb"`
	MemoryLimitMB       int64   `gorm:"column:memory_limit_mb;not null;default:0" json:"memory_limit_mb"`
	MemoryUsageMB       int64   `gorm:"column:memory_usage_mb;not null;default:0" json:"memory_usage_mb"`
	PodTotal            int     `gorm:"column:pod_total;not null;default:0" json:"pod_total"`
	PodRunning          int     `gorm:"column:pod_running;not null;default:0" json:"pod_running"`
	PodPending          int     `gorm:"column:pod_pending;not null;default:0" json:"pod_pending"`
	PodFailed           int     `gorm:"column:pod_failed;not null;default:0" json:"pod_failed"`
	PVCount             int     `gorm:"column:pv_count;not null;default:0" json:"pv_count"`
	PVCCount            int     `gorm:"column:pvc_count;not null;default:0" json:"pvc_count"`
	StorageUsedGB       float64 `gorm:"column:storage_used_gb;type:decimal(10,2);not null;default:0" json:"storage_used_gb"`
	CollectedAt         time.Time `gorm:"column:collected_at;not null;index:idx_cluster_collected,priority:2" json:"collected_at"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 返回集群资源快照表名。
func (ClusterResourceSnapshot) TableName() string { return "cluster_resource_snapshots" }

// K8sWorkloadStats K8s 工作负载统计表，存储 Deployment/StatefulSet/DaemonSet 等统计。
//
// 表名: k8s_workload_stats
// 用途: 缓存工作负载健康状态，用于主控台快速展示
type K8sWorkloadStats struct {
	ID                 uint      `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	ClusterID          uint      `gorm:"column:cluster_id;not null;index:idx_cluster_ns_collected,priority:1" json:"cluster_id"`
	Namespace          string    `gorm:"column:namespace;type:varchar(128);not null;default:'';index:idx_cluster_ns_collected,priority:2" json:"namespace"`
	DeploymentTotal    int       `gorm:"column:deployment_total;not null;default:0" json:"deployment_total"`
	DeploymentHealthy  int       `gorm:"column:deployment_healthy;not null;default:0" json:"deployment_healthy"`
	StatefulSetTotal   int       `gorm:"column:statefulset_total;not null;default:0" json:"statefulset_total"`
	StatefulSetHealthy int       `gorm:"column:statefulset_healthy;not null;default:0" json:"statefulset_healthy"`
	DaemonSetTotal     int       `gorm:"column:daemonset_total;not null;default:0" json:"daemonset_total"`
	DaemonSetHealthy   int       `gorm:"column:daemonset_healthy;not null;default:0" json:"daemonset_healthy"`
	ServiceCount       int       `gorm:"column:service_count;not null;default:0" json:"service_count"`
	IngressCount       int       `gorm:"column:ingress_count;not null;default:0" json:"ingress_count"`
	CollectedAt        time.Time `gorm:"column:collected_at;not null;index:idx_cluster_ns_collected,priority:3" json:"collected_at"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 返回 K8s 工作负载统计表名。
func (K8sWorkloadStats) TableName() string { return "k8s_workload_stats" }

// K8sIssuePod 异常 Pod 缓存表，存储有问题的 Pod 信息。
//
// 表名: k8s_issue_pods
// 用途: 快速展示集群中的异常 Pod 列表
type K8sIssuePod struct {
	ID          uint      `gorm:"primaryKey;column:id;autoIncrement" json:"id"`
	ClusterID   uint      `gorm:"column:cluster_id;not null;uniqueIndex:uk_cluster_ns_pod,priority:1" json:"cluster_id"`
	Namespace   string    `gorm:"column:namespace;type:varchar(128);not null;uniqueIndex:uk_cluster_ns_pod,priority:2" json:"namespace"`
	PodName     string    `gorm:"column:pod_name;type:varchar(256);not null;uniqueIndex:uk_cluster_ns_pod,priority:3" json:"pod_name"`
	IssueType   string    `gorm:"column:issue_type;type:varchar(64);not null;index" json:"issue_type"`
	IssueReason string    `gorm:"column:issue_reason;type:varchar(256);not null" json:"issue_reason"`
	Message     string    `gorm:"column:message;type:text" json:"message"`
	FirstSeenAt time.Time `gorm:"column:first_seen_at;not null" json:"first_seen_at"`
	LastSeenAt  time.Time `gorm:"column:last_seen_at;not null;index" json:"last_seen_at"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 返回异常 Pod 表名。
func (K8sIssuePod) TableName() string { return "k8s_issue_pods" }

// IssueType 常量定义。
const (
	IssueTypeCrashLoopBackOff   = "CrashLoopBackOff"
	IssueTypeImagePullBackOff   = "ImagePullBackOff"
	IssueTypeErrImagePull       = "ErrImagePull"
	IssueTypeCreateContainerErr = "CreateContainerConfigError"
	IssueTypeRunContainerErr    = "RunContainerError"
	IssueTypeOOMKilled          = "OOMKilled"
	IssueTypeEvicted            = "Evicted"
	IssueTypeUnknown            = "Unknown"
)
```

- [ ] **Step 2: Commit**

```bash
git add internal/model/dashboard_cache.go
git commit -m "feat: add dashboard cache models for K8s resource monitoring

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 1.2: Create Migration File

**Files:**
- Create: `storage/migrations/20260314_000037_dashboard_cache_tables.sql`

- [ ] **Step 1: Write the migration file**

```sql
-- +migrate Up
-- 集群资源快照表
CREATE TABLE IF NOT EXISTS cluster_resource_snapshots (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cluster_id INT UNSIGNED NOT NULL,
  cpu_allocatable_cores DECIMAL(10,2) NOT NULL DEFAULT 0,
  cpu_requested_cores DECIMAL(10,2) NOT NULL DEFAULT 0,
  cpu_limit_cores DECIMAL(10,2) NOT NULL DEFAULT 0,
  cpu_usage_cores DECIMAL(10,2) NOT NULL DEFAULT 0,
  memory_allocatable_mb BIGINT NOT NULL DEFAULT 0,
  memory_requested_mb BIGINT NOT NULL DEFAULT 0,
  memory_limit_mb BIGINT NOT NULL DEFAULT 0,
  memory_usage_mb BIGINT NOT NULL DEFAULT 0,
  pod_total INT NOT NULL DEFAULT 0,
  pod_running INT NOT NULL DEFAULT 0,
  pod_pending INT NOT NULL DEFAULT 0,
  pod_failed INT NOT NULL DEFAULT 0,
  pv_count INT NOT NULL DEFAULT 0,
  pvc_count INT NOT NULL DEFAULT 0,
  storage_used_gb DECIMAL(10,2) NOT NULL DEFAULT 0,
  collected_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_cluster_collected (cluster_id, collected_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='集群资源快照表';

-- K8s 工作负载统计表
CREATE TABLE IF NOT EXISTS k8s_workload_stats (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cluster_id INT UNSIGNED NOT NULL,
  namespace VARCHAR(128) NOT NULL DEFAULT '',
  deployment_total INT NOT NULL DEFAULT 0,
  deployment_healthy INT NOT NULL DEFAULT 0,
  statefulset_total INT NOT NULL DEFAULT 0,
  statefulset_healthy INT NOT NULL DEFAULT 0,
  daemonset_total INT NOT NULL DEFAULT 0,
  daemonset_healthy INT NOT NULL DEFAULT 0,
  service_count INT NOT NULL DEFAULT 0,
  ingress_count INT NOT NULL DEFAULT 0,
  collected_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_cluster_ns_collected (cluster_id, namespace, collected_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='K8s 工作负载统计表';

-- 异常 Pod 缓存表
CREATE TABLE IF NOT EXISTS k8s_issue_pods (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  cluster_id INT UNSIGNED NOT NULL,
  namespace VARCHAR(128) NOT NULL,
  pod_name VARCHAR(256) NOT NULL,
  issue_type VARCHAR(64) NOT NULL,
  issue_reason VARCHAR(256) NOT NULL,
  message TEXT,
  first_seen_at TIMESTAMP NOT NULL,
  last_seen_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_cluster_ns_pod (cluster_id, namespace, pod_name),
  INDEX idx_issue_type (issue_type),
  INDEX idx_last_seen (last_seen_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='异常 Pod 缓存表';

-- +migrate Down
DROP TABLE IF EXISTS k8s_issue_pods;
DROP TABLE IF EXISTS k8s_workload_stats;
DROP TABLE IF EXISTS cluster_resource_snapshots;
```

- [ ] **Step 2: Commit**

```bash
git add storage/migrations/20260314_000037_dashboard_cache_tables.sql
git commit -m "feat: add migration for dashboard cache tables

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 2: API Types

### Task 2.1: Add New Response Types

**Files:**
- Modify: `api/dashboard/v1/dashboard.go`

- [ ] **Step 1: Add new types after existing types**

Add after line 95 (after `AIActivity` struct):

```go

// WorkloadStats K8s 工作负载健康统计。
type WorkloadStats struct {
	Deployments  WorkloadHealth `json:"deployments"`
	StatefulSets WorkloadHealth `json:"statefulsets"`
	DaemonSets   WorkloadHealth `json:"daemonsets"`
	Services     int            `json:"services"`
	Ingresses    int            `json:"ingresses"`
}

// WorkloadHealth 单类工作负载健康统计。
type WorkloadHealth struct {
	Total   int `json:"total"`
	Healthy int `json:"healthy"`
}

// ClusterResource 集群资源概览。
type ClusterResource struct {
	ClusterID   uint           `json:"clusterId"`
	ClusterName string         `json:"clusterName"`
	CPU         ResourceMetric `json:"cpu"`
	Memory      ResourceMetric `json:"memory"`
	Pods        PodStats       `json:"pods"`
}

// ResourceMetric 资源指标。
type ResourceMetric struct {
	Allocatable  float64 `json:"allocatable"`
	Requested    float64 `json:"requested"`
	Usage        float64 `json:"usage"`
	UsagePercent float64 `json:"usagePercent"`
}

// PodStats Pod 状态统计。
type PodStats struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Pending int `json:"pending"`
	Failed  int `json:"failed"`
}

// DeploymentStats 部署状态统计。
type DeploymentStats struct {
	Running        int `json:"running"`
	PendingApproval int `json:"pendingApproval"`
	TodayTotal     int `json:"todayTotal"`
	TodaySuccess   int `json:"todaySuccess"`
	TodayFailed    int `json:"todayFailed"`
}

// CICDStats CI/CD 状态统计。
type CICDStats struct {
	Running    int `json:"running"`
	Queued     int `json:"queued"`
	TodayTotal int `json:"todayTotal"`
	Success    int `json:"success"`
	Failed     int `json:"failed"`
}

// IssuePodStats 异常 Pod 统计。
type IssuePodStats struct {
	Total  int            `json:"total"`
	ByType map[string]int `json:"byType"`
}

// HealthOverview 健康概览。
type HealthOverview struct {
	Hosts        HealthStats   `json:"hosts"`
	Clusters     HealthStats   `json:"clusters"`
	Applications HealthStats   `json:"applications"`
	Workloads    WorkloadStats `json:"workloads"`
}

// ResourcesOverview 资源使用概览。
type ResourcesOverview struct {
	Hosts    []MetricSeries    `json:"hosts"`
	Clusters []ClusterResource `json:"clusters"`
}

// OperationsOverview 运行状态概览。
type OperationsOverview struct {
	Deployments DeploymentStats `json:"deployments"`
	CICD        CICDStats       `json:"cicd"`
	IssuePods   IssuePodStats   `json:"issuePods"`
}

// OverviewResponseV2 主控台概览响应 V2（增强版）。
type OverviewResponseV2 struct {
	Health     HealthOverview     `json:"health"`
	Resources  ResourcesOverview  `json:"resources"`
	Operations OperationsOverview `json:"operations"`
	Alerts     AlertSummary       `json:"alerts"`
	Events     []EventItem        `json:"events"`
	AI         AIActivity         `json:"ai"`
}
```

- [ ] **Step 2: Commit**

```bash
git add api/dashboard/v1/dashboard.go
git commit -m "feat: add new response types for dashboard observability

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 3: Data Collector

### Task 3.1: Create Dashboard Collector

**Files:**
- Create: `internal/service/dashboard/collector.go`

- [ ] **Step 1: Write the collector file**

```go
// Package dashboard 提供主控台相关的业务逻辑。
//
// 本文件实现主控台数据采集器，定时采集 K8s 集群资源、工作负载状态和异常 Pod。
package dashboard

import (
	"context"
	"sync"
	"time"

	prominfra "github.com/cy77cc/OpsPilot/internal/infra/prometheus"
	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"gorm.io/gorm"
)

// Collector 主控台数据采集器。
type Collector struct {
	svcCtx        *svc.ServiceContext
	db            *gorm.DB
	prometheus    prominfra.Client
	collectorOnce sync.Once
}

// NewCollector 创建主控台数据采集器。
func NewCollector(svcCtx *svc.ServiceContext) *Collector {
	return &Collector{
		svcCtx:     svcCtx,
		db:         svcCtx.DB,
		prometheus: svcCtx.Prometheus,
	}
}

// Start 启动定时采集。
func (c *Collector) Start() {
	c.collectorOnce.Do(func() {
		// 首次立即采集
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		c.Collect(ctx)
		cancel()

		// 启动定时采集
		go func() {
			// 每 5 分钟采集集群资源
			resourceTicker := time.NewTicker(5 * time.Minute)
			defer resourceTicker.Stop()

			// 每 1 分钟采集工作负载状态
			workloadTicker := time.NewTicker(1 * time.Minute)
			defer workloadTicker.Stop()

			// 每 30 秒采集异常 Pod
			issuePodTicker := time.NewTicker(30 * time.Second)
			defer issuePodTicker.Stop()

			for {
				select {
				case <-resourceTicker.C:
					ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					c.collectClusterResources(ctx)
					cancel()
				case <-workloadTicker.C:
					ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					c.collectWorkloadStats(ctx)
					cancel()
				case <-issuePodTicker.C:
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					c.collectIssuePods(ctx)
					cancel()
				}
			}
		}()

		logger.L().Info("Dashboard collector started")
	})
}

// Collect 执行一轮完整采集。
func (c *Collector) Collect(ctx context.Context) {
	c.collectClusterResources(ctx)
	c.collectWorkloadStats(ctx)
	c.collectIssuePods(ctx)
}

// collectClusterResources 采集所有集群的资源使用情况。
func (c *Collector) collectClusterResources(ctx context.Context) {
	var clusters []model.Cluster
	if err := c.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		logger.L().Warn("failed to list clusters for resource collection", logger.Error(err))
		return
	}

	for i := range clusters {
		c.collectClusterResource(ctx, &clusters[i])
	}
}

// collectClusterResource 采集单个集群的资源使用情况。
func (c *Collector) collectClusterResource(ctx context.Context, cluster *model.Cluster) {
	cli, err := c.getK8sClient(cluster)
	if err != nil {
		logger.L().Debug("failed to get k8s client for cluster",
			logger.Error(err),
			logger.Uint("cluster_id", uint(cluster.ID)),
		)
		return
	}

	// 查询节点资源
	nodes, err := cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.L().Warn("failed to list nodes", logger.Error(err), logger.Uint("cluster_id", uint(cluster.ID)))
		return
	}

	var cpuAllocatable, memAllocatable float64
	for _, node := range nodes.Items {
		cpuAllocatable += float64(node.Status.Allocatable.Cpu().MilliValue()) / 1000
		memAllocatable += float64(node.Status.Allocatable.Memory().Value()) / 1024 / 1024
	}

	// 查询 Pod 资源请求
	pods, err := cli.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.L().Warn("failed to list pods", logger.Error(err), logger.Uint("cluster_id", uint(cluster.ID)))
		return
	}

	var cpuRequested, memRequested float64
	var runningCount, pendingCount, failedCount int
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if req := container.Resources.Requests; req != nil {
				cpuRequested += float64(req.Cpu().MilliValue()) / 1000
				memRequested += float64(req.Memory().Value()) / 1024 / 1024
			}
		}
		switch pod.Status.Phase {
		case corev1.PodRunning:
			runningCount++
		case corev1.PodPending:
			pendingCount++
		case corev1.PodFailed:
			failedCount++
		}
	}

	// 从 Prometheus 查询实际使用
	var cpuUsage, memUsage float64
	if c.prometheus != nil {
		cpuUsage = c.queryClusterCPUUsage(ctx, cluster.ID)
		memUsage = c.queryClusterMemoryUsage(ctx, cluster.ID)
	}

	// 写入快照
	snapshot := model.ClusterResourceSnapshot{
		ClusterID:           cluster.ID,
		CPUAllocatableCores: cpuAllocatable,
		CPURequestedCores:   cpuRequested,
		CPUUsageCores:       cpuUsage,
		MemoryAllocatableMB: int64(memAllocatable),
		MemoryRequestedMB:   int64(memRequested),
		MemoryUsageMB:       int64(memUsage),
		PodTotal:            len(pods.Items),
		PodRunning:          runningCount,
		PodPending:          pendingCount,
		PodFailed:           failedCount,
		CollectedAt:         time.Now().UTC(),
	}

	if err := c.db.Create(&snapshot).Error; err != nil {
		logger.L().Warn("failed to save cluster resource snapshot", logger.Error(err))
	}
}

// collectWorkloadStats 采集所有集群的工作负载状态。
func (c *Collector) collectWorkloadStats(ctx context.Context) {
	var clusters []model.Cluster
	if err := c.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		return
	}

	for i := range clusters {
		c.collectClusterWorkload(ctx, &clusters[i])
	}
}

// collectClusterWorkload 采集单个集群的工作负载状态。
func (c *Collector) collectClusterWorkload(ctx context.Context, cluster *model.Cluster) {
	cli, err := c.getK8sClient(cluster)
	if err != nil {
		return
	}

	// 采集 Deployment
	deployments, _ := cli.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	var deployTotal, deployHealthy int
	for _, d := range deployments.Items {
		deployTotal++
		if d.Status.ReadyReplicas == d.Status.Replicas && d.Status.Replicas > 0 {
			deployHealthy++
		}
	}

	// 采集 StatefulSet
	statefulsets, _ := cli.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	var stsTotal, stsHealthy int
	for _, sts := range statefulsets.Items {
		stsTotal++
		if sts.Status.ReadyReplicas == sts.Status.Replicas && sts.Status.Replicas > 0 {
			stsHealthy++
		}
	}

	// 采集 DaemonSet
	daemonsets, _ := cli.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	var dsTotal, dsHealthy int
	for _, ds := range daemonsets.Items {
		dsTotal++
		if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
			dsHealthy++
		}
	}

	// 采集 Service
	services, _ := cli.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	serviceCount := len(services.Items)

	// 采集 Ingress
	ingresses, _ := cli.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	ingressCount := len(ingresses.Items)

	// 按命名空间聚合写入
	nsMap := make(map[string]model.K8sWorkloadStats)
	for _, d := range deployments.Items {
		ns := d.Namespace
		stats := nsMap[ns]
		stats.ClusterID = cluster.ID
		stats.Namespace = ns
		stats.DeploymentTotal++
		if d.Status.ReadyReplicas == d.Status.Replicas && d.Status.Replicas > 0 {
			stats.DeploymentHealthy++
		}
		nsMap[ns] = stats
	}

	// 写入数据库（简化版：只写集群级别的聚合）
	now := time.Now().UTC()
	stats := model.K8sWorkloadStats{
		ClusterID:          cluster.ID,
		Namespace:          "",
		DeploymentTotal:    deployTotal,
		DeploymentHealthy:  deployHealthy,
		StatefulSetTotal:   stsTotal,
		StatefulSetHealthy: stsHealthy,
		DaemonSetTotal:     dsTotal,
		DaemonSetHealthy:   dsHealthy,
		ServiceCount:       serviceCount,
		IngressCount:       ingressCount,
		CollectedAt:        now,
	}
	c.db.Create(&stats)
}

// collectIssuePods 采集所有集群的异常 Pod。
func (c *Collector) collectIssuePods(ctx context.Context) {
	var clusters []model.Cluster
	if err := c.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		return
	}

	for i := range clusters {
		c.collectClusterIssuePods(ctx, &clusters[i])
	}
}

// collectClusterIssuePods 采集单个集群的异常 Pod。
func (c *Collector) collectClusterIssuePods(ctx context.Context, cluster *model.Cluster) {
	cli, err := c.getK8sClient(cluster)
	if err != nil {
		return
	}

	pods, err := cli.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}

	now := time.Now().UTC()
	var issuePods []model.K8sIssuePod

	for _, pod := range pods.Items {
		issueType, issueReason, message := detectPodIssue(&pod)
		if issueType == "" {
			continue
		}

		issuePods = append(issuePods, model.K8sIssuePod{
			ClusterID:   cluster.ID,
			Namespace:   pod.Namespace,
			PodName:     pod.Name,
			IssueType:   issueType,
			IssueReason: issueReason,
			Message:     message,
			FirstSeenAt: now,
			LastSeenAt:  now,
		})
	}

	// 更新或创建异常 Pod 记录
	for _, ip := range issuePods {
		var existing model.K8sIssuePod
		err := c.db.Where("cluster_id = ? AND namespace = ? AND pod_name = ?", ip.ClusterID, ip.Namespace, ip.PodName).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			c.db.Create(&ip)
		} else if err == nil {
			c.db.Model(&existing).Updates(map[string]any{
				"issue_type":   ip.IssueType,
				"issue_reason": ip.IssueReason,
				"message":      ip.Message,
				"last_seen_at": now,
			})
		}
	}

	// 清理已恢复的异常 Pod（超过 5 分钟未更新）
	c.db.Where("cluster_id = ? AND last_seen_at < ?", cluster.ID, now.Add(-5*time.Minute)).Delete(&model.K8sIssuePod{})
}

// detectPodIssue 检测 Pod 是否有问题。
func detectPodIssue(pod *corev1.Pod) (issueType, issueReason, message string) {
	// 检查 Pod 状态
	if pod.Status.Phase == corev1.PodFailed {
		return "Failed", string(pod.Status.Phase), "Pod is in failed state"
	}

	// 检查容器状态
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			switch cs.State.Waiting.Reason {
			case "CrashLoopBackOff":
				return model.IssueTypeCrashLoopBackOff, cs.State.Waiting.Reason, cs.State.Waiting.Message
			case "ImagePullBackOff":
				return model.IssueTypeImagePullBackOff, cs.State.Waiting.Reason, cs.State.Waiting.Message
			case "ErrImagePull":
				return model.IssueTypeErrImagePull, cs.State.Waiting.Reason, cs.State.Waiting.Message
			case "CreateContainerConfigError":
				return model.IssueTypeCreateContainerErr, cs.State.Waiting.Reason, cs.State.Waiting.Message
			}
		}
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			if cs.State.Terminated.Reason == "OOMKilled" {
				return model.IssueTypeOOMKilled, cs.State.Terminated.Reason, cs.State.Terminated.Message
			}
		}
	}

	// 检查 Pod 条件
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse {
			return model.IssueTypeUnknown, "NotReady", condition.Message
		}
	}

	return "", "", ""
}

// getK8sClient 获取集群的 K8s 客户端。
func (c *Collector) getK8sClient(cluster *model.Cluster) (*kubernetes.Clientset, error) {
	if cluster.KubeConfig == "" {
		return nil, ErrKubeConfigNotFound
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// queryClusterCPUUsage 从 Prometheus 查询集群 CPU 使用率。
func (c *Collector) queryClusterCPUUsage(ctx context.Context, clusterID uint) float64 {
	if c.prometheus == nil {
		return 0
	}
	// 简化实现：查询节点 CPU 使用率总和
	result, err := c.prometheus.Query(ctx, `sum(rate(container_cpu_usage_seconds_total{container!=""}[5m]))`, time.Now())
	if err != nil {
		return 0
	}
	if len(result.Vector) > 0 && len(result.Vector[0].Value) >= 2 {
		if v, ok := result.Vector[0].Value[1].(float64); ok {
			return v
		}
	}
	return 0
}

// queryClusterMemoryUsage 从 Prometheus 查询集群内存使用量。
func (c *Collector) queryClusterMemoryUsage(ctx context.Context, clusterID uint) float64 {
	if c.prometheus == nil {
		return 0
	}
	result, err := c.prometheus.Query(ctx, `sum(container_memory_working_set_bytes{container!=""})/1024/1024`, time.Now())
	if err != nil {
		return 0
	}
	if len(result.Vector) > 0 && len(result.Vector[0].Value) >= 2 {
		if v, ok := result.Vector[0].Value[1].(float64); ok {
			return v
		}
	}
	return 0
}

// ErrKubeConfigNotFound 表示未找到 KubeConfig。
var ErrKubeConfigNotFound = &KubeConfigNotFoundError{}

type KubeConfigNotFoundError struct{}

func (e *KubeConfigNotFoundError) Error() string {
	return "kubeconfig not found"
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/service/dashboard/collector.go
git commit -m "feat: add dashboard collector for K8s resource monitoring

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 4: Logic Refactoring

### Task 4.1: Refactor GetOverview Method

**Files:**
- Modify: `internal/service/dashboard/logic.go`

- [ ] **Step 1: Add new aggregation methods**

Add after the existing `parseTimeRange` function (around line 543):

```go
// getClusterResources 获取集群资源概览。
func (l *Logic) getClusterResources(ctx context.Context) ([]dashboardv1.ClusterResource, error) {
	// 从缓存表读取最新的集群资源快照
	type snapshotWithCluster struct {
		model.ClusterResourceSnapshot
		ClusterName string
	}
	var snapshots []snapshotWithCluster
	err := l.svcCtx.DB.WithContext(ctx).
		Table("cluster_resource_snapshots crs").
		Select("crs.*, c.name as cluster_name").
		Joins("JOIN clusters c ON c.id = crs.cluster_id").
		Where("crs.id IN (SELECT MAX(id) FROM cluster_resource_snapshots GROUP BY cluster_id)").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}

	out := make([]dashboardv1.ClusterResource, 0, len(snapshots))
	for _, s := range snapshots {
		cpuUsagePercent := float64(0)
		if s.CPUAllocatableCores > 0 {
			cpuUsagePercent = s.CPUUsageCores / s.CPUAllocatableCores * 100
		}
		memUsagePercent := float64(0)
		if s.MemoryAllocatableMB > 0 {
			memUsagePercent = float64(s.MemoryUsageMB) / float64(s.MemoryAllocatableMB) * 100
		}
		out = append(out, dashboardv1.ClusterResource{
			ClusterID:   s.ClusterID,
			ClusterName: s.ClusterName,
			CPU: dashboardv1.ResourceMetric{
				Allocatable:  s.CPUAllocatableCores,
				Requested:    s.CPURequestedCores,
				Usage:        s.CPUUsageCores,
				UsagePercent: cpuUsagePercent,
			},
			Memory: dashboardv1.ResourceMetric{
				Allocatable:  float64(s.MemoryAllocatableMB),
				Requested:    float64(s.MemoryRequestedMB),
				Usage:        float64(s.MemoryUsageMB),
				UsagePercent: memUsagePercent,
			},
			Pods: dashboardv1.PodStats{
				Total:   s.PodTotal,
				Running: s.PodRunning,
				Pending: s.PodPending,
				Failed:  s.PodFailed,
			},
		})
	}
	return out, nil
}

// getWorkloadStats 获取工作负载健康统计。
func (l *Logic) getWorkloadStats(ctx context.Context) (dashboardv1.WorkloadStats, error) {
	var stats model.K8sWorkloadStats
	err := l.svcCtx.DB.WithContext(ctx).
		Where("id = (SELECT MAX(id) FROM k8s_workload_stats WHERE namespace = '')").
		First(&stats).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return dashboardv1.WorkloadStats{}, err
	}
	return dashboardv1.WorkloadStats{
		Deployments: dashboardv1.WorkloadHealth{
			Total:   stats.DeploymentTotal,
			Healthy: stats.DeploymentHealthy,
		},
		StatefulSets: dashboardv1.WorkloadHealth{
			Total:   stats.StatefulSetTotal,
			Healthy: stats.StatefulSetHealthy,
		},
		DaemonSets: dashboardv1.WorkloadHealth{
			Total:   stats.DaemonSetTotal,
			Healthy: stats.DaemonSetHealthy,
		},
		Services:  stats.ServiceCount,
		Ingresses: stats.IngressCount,
	}, nil
}

// getDeploymentStats 获取部署状态统计。
func (l *Logic) getDeploymentStats(ctx context.Context) (dashboardv1.DeploymentStats, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var stats dashboardv1.DeploymentStats

	// 正在部署的数量
	l.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("status IN ?", []string{"deploying", "applying"}).
		Count(&stats.Running)

	// 待审批的数量
	l.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("status = ?", "pending_approval").
		Count(&stats.PendingApproval)

	// 今日发布统计
	l.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("created_at >= ?", today).
		Count(&stats.TodayTotal)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("created_at >= ? AND status = ?", today, "success").
		Count(&stats.TodaySuccess)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("created_at >= ? AND status = ?", today, "failed").
		Count(&stats.TodayFailed)

	return stats, nil
}

// getCICDStats 获取 CI/CD 状态统计。
func (l *Logic) getCICDStats(ctx context.Context) (dashboardv1.CICDStats, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var stats dashboardv1.CICDStats

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.CICDServiceCIRun{}).
		Where("status = ?", "running").
		Count(&stats.Running)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.CICDServiceCIRun{}).
		Where("status = ?", "queued").
		Count(&stats.Queued)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.CICDServiceCIRun{}).
		Where("triggered_at >= ?", today).
		Count(&stats.TodayTotal)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.CICDServiceCIRun{}).
		Where("triggered_at >= ? AND status = ?", today, "success").
		Count(&stats.Success)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.CICDServiceCIRun{}).
		Where("triggered_at >= ? AND status = ?", today, "failed").
		Count(&stats.Failed)

	return stats, nil
}

// getIssuePodStats 获取异常 Pod 统计。
func (l *Logic) getIssuePodStats(ctx context.Context) (dashboardv1.IssuePodStats, error) {
	var stats dashboardv1.IssuePodStats
	stats.ByType = make(map[string]int)

	var total int64
	l.svcCtx.DB.WithContext(ctx).
		Model(&model.K8sIssuePod{}).
		Count(&total)
	stats.Total = int(total)

	var byType []struct {
		IssueType string
		Count     int64
	}
	l.svcCtx.DB.WithContext(ctx).
		Model(&model.K8sIssuePod{}).
		Select("issue_type, COUNT(*) as count").
		Group("issue_type").
		Find(&byType)

	for _, bt := range byType {
		stats.ByType[bt.IssueType] = int(bt.Count)
	}

	return stats, nil
}

// getEnrichedEvents 获取增强版事件流（包含部署事件）。
func (l *Logic) getEnrichedEvents(ctx context.Context) ([]dashboardv1.EventItem, error) {
	events := make([]dashboardv1.EventItem, 0, 20)

	// 主机事件
	nodeEvents := make([]model.NodeEvent, 0, 10)
	l.svcCtx.DB.WithContext(ctx).
		Order("created_at DESC").
		Limit(10).
		Find(&nodeEvents)
	for _, e := range nodeEvents {
		events = append(events, dashboardv1.EventItem{
			ID:        fmt.Sprintf("node-%d", e.ID),
			Type:      "host_event",
			Message:   e.Message,
			CreatedAt: e.CreatedAt,
		})
	}

	// 告警事件
	alertEvents := make([]model.AlertEvent, 0, 10)
	l.svcCtx.DB.WithContext(ctx).
		Order("created_at DESC").
		Limit(10).
		Find(&alertEvents)
	for _, e := range alertEvents {
		events = append(events, dashboardv1.EventItem{
			ID:        fmt.Sprintf("alert-%d", e.ID),
			Type:      "alert",
			Message:   e.Title,
			CreatedAt: e.CreatedAt,
		})
	}

	// 部署事件
	releaseEvents := make([]model.DeploymentRelease, 0, 10)
	l.svcCtx.DB.WithContext(ctx).
		Order("created_at DESC").
		Limit(10).
		Find(&releaseEvents)
	for _, r := range releaseEvents {
		events = append(events, dashboardv1.EventItem{
			ID:        fmt.Sprintf("release-%d", r.ID),
			Type:      "deployment",
			Message:   fmt.Sprintf("发布状态: %s", r.Status),
			CreatedAt: r.CreatedAt,
		})
	}

	// 按时间排序
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})

	// 取最近的 20 条
	if len(events) > 20 {
		events = events[:20]
	}

	return events, nil
}
```

- [ ] **Step 2: Modify GetOverview method to return new structure**

Replace the existing `GetOverview` method with:

```go
func (l *Logic) GetOverview(ctx context.Context, timeRange string) (*dashboardv1.OverviewResponse, error) {
	now := time.Now()
	since, err := parseTimeRange(now, timeRange)
	if err != nil {
		return nil, err
	}

	out := &dashboardv1.OverviewResponse{}
	group, gctx := errgroup.WithContext(ctx)

	// 健康概览
	group.Go(func() error {
		hostStats, err := l.aggregateHostStats(gctx)
		if err != nil {
			return err
		}
		out.Hosts = hostStats
		return nil
	})

	group.Go(func() error {
		clusterStats, err := l.aggregateClusterStats(gctx)
		if err != nil {
			return err
		}
		out.Clusters = clusterStats
		return nil
	})

	group.Go(func() error {
		serviceStats, err := l.aggregateServiceStats(gctx, now)
		if err != nil {
			return err
		}
		out.Services = serviceStats
		return nil
	})

	// 资源使用
	group.Go(func() error {
		metrics, err := l.getMetricsSeries(gctx, since, now)
		if err != nil {
			return err
		}
		out.Metrics = metrics
		return nil
	})

	// 告警事件
	group.Go(func() error {
		alerts, err := l.getRecentAlerts(gctx)
		if err != nil {
			return err
		}
		out.Alerts = alerts
		return nil
	})

	// 事件流
	group.Go(func() error {
		events, err := l.getRecentEvents(gctx)
		if err != nil {
			return err
		}
		out.Events = events
		return nil
	})

	// AI 活动（修正数据源）
	group.Go(func() error {
		aiActivity, err := l.getAIActivity(gctx, since, now)
		if err != nil {
			return err
		}
		out.AI = aiActivity
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetOverviewV2 返回增强版主控台概览。
func (l *Logic) GetOverviewV2(ctx context.Context, timeRange string) (*dashboardv1.OverviewResponseV2, error) {
	now := time.Now()
	since, err := parseTimeRange(now, timeRange)
	if err != nil {
		return nil, err
	}

	out := &dashboardv1.OverviewResponseV2{}
	group, gctx := errgroup.WithContext(ctx)

	// 健康概览
	group.Go(func() error {
		hostStats, err := l.aggregateHostStats(gctx)
		if err != nil {
			return err
		}
		out.Health.Hosts = hostStats
		return nil
	})

	group.Go(func() error {
		clusterStats, err := l.aggregateClusterStats(gctx)
		if err != nil {
			return err
		}
		out.Health.Clusters = clusterStats
		return nil
	})

	group.Go(func() error {
		appStats, err := l.aggregateServiceStats(gctx, now)
		if err != nil {
			return err
		}
		out.Health.Applications = appStats
		return nil
	})

	group.Go(func() error {
		workloadStats, err := l.getWorkloadStats(gctx)
		if err != nil {
			return err
		}
		out.Health.Workloads = workloadStats
		return nil
	})

	// 资源使用
	group.Go(func() error {
		metrics, err := l.getMetricsSeries(gctx, since, now)
		if err != nil {
			return err
		}
		out.Resources.Hosts = metrics.CPUUsage
		// 复用 CPU usage 的数据结构
		return nil
	})

	group.Go(func() error {
		clusterResources, err := l.getClusterResources(gctx)
		if err != nil {
			return err
		}
		out.Resources.Clusters = clusterResources
		return nil
	})

	// 运行状态
	group.Go(func() error {
		deployStats, err := l.getDeploymentStats(gctx)
		if err != nil {
			return err
		}
		out.Operations.Deployments = deployStats
		return nil
	})

	group.Go(func() error {
		cicdStats, err := l.getCICDStats(gctx)
		if err != nil {
			return err
		}
		out.Operations.CICD = cicdStats
		return nil
	})

	group.Go(func() error {
		issueStats, err := l.getIssuePodStats(gctx)
		if err != nil {
			return err
		}
		out.Operations.IssuePods = issueStats
		return nil
	})

	// 告警事件
	group.Go(func() error {
		alerts, err := l.getRecentAlerts(gctx)
		if err != nil {
			return err
		}
		out.Alerts = alerts
		return nil
	})

	// 事件流（增强版）
	group.Go(func() error {
		events, err := l.getEnrichedEvents(gctx)
		if err != nil {
			return err
		}
		out.Events = events
		return nil
	})

	// AI 活动
	group.Go(func() error {
		aiActivity, err := l.getAIActivity(gctx, since, now)
		if err != nil {
			return err
		}
		out.AI = aiActivity
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}
```

- [ ] **Step 3: Add necessary imports**

Ensure these imports are present:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	dashboardv1 "github.com/cy77cc/OpsPilot/api/dashboard/v1"
	prominfra "github.com/cy77cc/OpsPilot/internal/infra/prometheus"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)
```

- [ ] **Step 4: Commit**

```bash
git add internal/service/dashboard/logic.go
git commit -m "feat: refactor dashboard logic with new aggregation methods

- Add cluster resources overview
- Add workload health stats
- Add deployment/CI status
- Add issue pod stats
- Fix AI activity data source

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 5: Handler and Routes

### Task 5.1: Add V2 Handler

**Files:**
- Modify: `internal/service/dashboard/handler.go`

- [ ] **Step 1: Add GetOverviewV2 handler**

Add after the existing `GetOverview` handler:

```go
// GetOverviewV2 返回增强版主控台概览。
func (h *HTTPHandler) GetOverviewV2(c *gin.Context) {
	timeRange := c.Query("time_range")
	if timeRange == "" {
		timeRange = "1h"
	}

	ctx := c.Request.Context()
	response, err := h.logic.GetOverviewV2(ctx, timeRange)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, response)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/service/dashboard/handler.go
git commit -m "feat: add GetOverviewV2 handler

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 5.2: Update Routes

**Files:**
- Modify: `internal/service/dashboard/routes.go`

- [ ] **Step 1: Add V2 route**

Add after the existing overview route:

```go
v1.GET("/dashboard/overview/v2", h.GetOverviewV2)
```

- [ ] **Step 2: Commit**

```bash
git add internal/service/dashboard/routes.go
git commit -m "feat: add dashboard overview v2 route

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 6: Frontend Types

### Task 6.1: Update Frontend Types

**Files:**
- Modify: `web/src/api/modules/dashboard.ts`

- [ ] **Step 1: Add new type definitions**

Add after the existing `AIActivity` interface:

```typescript
// 新增类型定义

export interface WorkloadHealth {
  total: number;
  healthy: number;
}

export interface WorkloadStats {
  deployments: WorkloadHealth;
  statefulsets: WorkloadHealth;
  daemonsets: WorkloadHealth;
  services: number;
  ingresses: number;
}

export interface ResourceMetric {
  allocatable: number;
  requested: number;
  usage: number;
  usagePercent: number;
}

export interface PodStats {
  total: number;
  running: number;
  pending: number;
  failed: number;
}

export interface ClusterResource {
  clusterId: number;
  clusterName: string;
  cpu: ResourceMetric;
  memory: ResourceMetric;
  pods: PodStats;
}

export interface DeploymentStats {
  running: number;
  pendingApproval: number;
  todayTotal: number;
  todaySuccess: number;
  todayFailed: number;
}

export interface CICDStats {
  running: number;
  queued: number;
  todayTotal: number;
  success: number;
  failed: number;
}

export interface IssuePodStats {
  total: number;
  byType: Record<string, number>;
}

export interface HealthOverview {
  hosts: HealthStats;
  clusters: HealthStats;
  applications: HealthStats;
  workloads: WorkloadStats;
}

export interface ResourcesOverview {
  hosts: MetricSeries[];
  clusters: ClusterResource[];
}

export interface OperationsOverview {
  deployments: DeploymentStats;
  cicd: CICDStats;
  issuePods: IssuePodStats;
}

export interface OverviewResponseV2 {
  health: HealthOverview;
  resources: ResourcesOverview;
  operations: OperationsOverview;
  alerts: {
    firing: number;
    recent: AlertItem[];
  };
  events: EventItem[];
  ai: AIActivity;
}
```

- [ ] **Step 2: Add normalizer functions and API method**

Add after `normalizeAIActivity`:

```typescript
const normalizeWorkloadHealth = (data: any): WorkloadHealth => ({
  total: Number(data?.total || 0),
  healthy: Number(data?.healthy || 0),
});

const normalizeWorkloadStats = (data: any): WorkloadStats => ({
  deployments: normalizeWorkloadHealth(data?.deployments || {}),
  statefulsets: normalizeWorkloadHealth(data?.statefulsets || {}),
  daemonsets: normalizeWorkloadHealth(data?.daemonsets || {}),
  services: Number(data?.services || 0),
  ingresses: Number(data?.ingresses || 0),
});

const normalizeResourceMetric = (data: any): ResourceMetric => ({
  allocatable: Number(data?.allocatable || 0),
  requested: Number(data?.requested || 0),
  usage: Number(data?.usage || 0),
  usagePercent: Number(data?.usagePercent || 0),
});

const normalizeClusterResource = (data: any): ClusterResource => ({
  clusterId: Number(data?.clusterId || 0),
  clusterName: String(data?.clusterName || ''),
  cpu: normalizeResourceMetric(data?.cpu || {}),
  memory: normalizeResourceMetric(data?.memory || {}),
  pods: {
    total: Number(data?.pods?.total || 0),
    running: Number(data?.pods?.running || 0),
    pending: Number(data?.pods?.pending || 0),
    failed: Number(data?.pods?.failed || 0),
  },
});

const normalizeDeploymentStats = (data: any): DeploymentStats => ({
  running: Number(data?.running || 0),
  pendingApproval: Number(data?.pendingApproval || 0),
  todayTotal: Number(data?.todayTotal || 0),
  todaySuccess: Number(data?.todaySuccess || 0),
  todayFailed: Number(data?.todayFailed || 0),
});

const normalizeCICDStats = (data: any): CICDStats => ({
  running: Number(data?.running || 0),
  queued: Number(data?.queued || 0),
  todayTotal: Number(data?.todayTotal || 0),
  success: Number(data?.success || 0),
  failed: Number(data?.failed || 0),
});

const normalizeIssuePodStats = (data: any): IssuePodStats => ({
  total: Number(data?.total || 0),
  byType: data?.byType || {},
});

export const dashboardApi = {
  // ... existing getOverview method ...

  async getOverviewV2(timeRange: TimeRange = '1h'): Promise<ApiResponse<OverviewResponseV2>> {
    const response = await apiService.get<any>('/dashboard/overview/v2', {
      params: { time_range: timeRange },
    });

    const raw = response.data || {};
    return {
      ...response,
      data: {
        health: {
          hosts: normalizeHealthStats(raw?.health?.hosts),
          clusters: normalizeHealthStats(raw?.health?.clusters),
          applications: normalizeHealthStats(raw?.health?.applications),
          workloads: normalizeWorkloadStats(raw?.health?.workloads),
        },
        resources: {
          hosts: Array.isArray(raw?.resources?.hosts)
            ? raw.resources.hosts.map(normalizeMetricSeries)
            : [],
          clusters: Array.isArray(raw?.resources?.clusters)
            ? raw.resources.clusters.map(normalizeClusterResource)
            : [],
        },
        operations: {
          deployments: normalizeDeploymentStats(raw?.operations?.deployments),
          cicd: normalizeCICDStats(raw?.operations?.cicd),
          issuePods: normalizeIssuePodStats(raw?.operations?.issuePods),
        },
        alerts: {
          firing: Number(raw?.alerts?.firing || 0),
          recent: Array.isArray(raw?.alerts?.recent)
            ? raw.alerts.recent.map(normalizeAlertItem)
            : [],
        },
        events: Array.isArray(raw?.events)
          ? raw.events.map(normalizeEventItem)
          : [],
        ai: normalizeAIActivity(raw?.ai),
      },
    };
  },
};
```

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/dashboard.ts
git commit -m "feat: add frontend types for dashboard V2

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 7: Frontend Components

### Task 7.1: Create WorkloadHealthCard Component

**Files:**
- Create: `web/src/components/Dashboard/WorkloadHealthCard.tsx`

- [ ] **Step 1: Write the component**

```tsx
import React from 'react';
import { Card, Progress, Space, Typography } from 'antd';
import {
  AppstoreOutlined,
  CloudServerOutlined,
  RocketOutlined,
} from '@ant-design/icons';
import type { WorkloadStats } from '../../api/modules/dashboard';

const { Text } = Typography;

interface Props {
  data: WorkloadStats;
  loading?: boolean;
}

const WorkloadHealthCard: React.FC<Props> = ({ data, loading }) => {
  const items = [
    {
      key: 'deployments',
      icon: <RocketOutlined className="text-blue-500" />,
      label: 'Deployment',
      total: data.deployments.total,
      healthy: data.deployments.healthy,
    },
    {
      key: 'statefulsets',
      icon: <CloudServerOutlined className="text-purple-500" />,
      label: 'StatefulSet',
      total: data.statefulsets.total,
      healthy: data.statefulsets.healthy,
    },
    {
      key: 'daemonsets',
      icon: <AppstoreOutlined className="text-cyan-500" />,
      label: 'DaemonSet',
      total: data.daemonsets.total,
      healthy: data.daemonsets.healthy,
    },
  ];

  return (
    <Card title="工作负载健康" loading={loading} size="small">
      <div className="space-y-3">
        {items.map((item) => {
          const percent = item.total > 0 ? Math.round((item.healthy / item.total) * 100) : 100;
          const status = percent === 100 ? 'success' : percent >= 80 ? 'normal' : 'exception';

          return (
            <div key={item.key} className="flex items-center gap-3">
              <div className="w-6">{item.icon}</div>
              <div className="flex-1">
                <div className="flex justify-between mb-1">
                  <Text type="secondary" className="text-xs">{item.label}</Text>
                  <Text className="text-xs">{item.healthy}/{item.total}</Text>
                </div>
                <Progress
                  percent={percent}
                  size="small"
                  showInfo={false}
                  status={status}
                />
              </div>
            </div>
          );
        })}
        <div className="flex justify-between pt-2 border-t border-gray-100">
          <Space>
            <Text type="secondary" className="text-xs">Service</Text>
            <Text strong>{data.services}</Text>
          </Space>
          <Space>
            <Text type="secondary" className="text-xs">Ingress</Text>
            <Text strong>{data.ingresses}</Text>
          </Space>
        </div>
      </div>
    </Card>
  );
};

export default WorkloadHealthCard;
```

- [ ] **Step 2: Commit**

```bash
git add web/src/components/Dashboard/WorkloadHealthCard.tsx
git commit -m "feat: add WorkloadHealthCard component

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 7.2: Create ClusterResourceCard Component

**Files:**
- Create: `web/src/components/Dashboard/ClusterResourceCard.tsx`

- [ ] **Step 1: Write the component**

```tsx
import React from 'react';
import { Card, Progress, Select, Empty, Typography } from 'antd';
import {
  DashboardOutlined,
  MemoryOutlined,
  CloudOutlined,
} from '@ant-design/icons';
import type { ClusterResource } from '../../api/modules/dashboard';

const { Text } = Typography;

interface Props {
  data: ClusterResource[];
  loading?: boolean;
}

const ClusterResourceCard: React.FC<Props> = ({ data, loading }) => {
  const [selectedCluster, setSelectedCluster] = React.useState<number | null>(
    data.length > 0 ? data[0].clusterId : null
  );

  const cluster = data.find((c) => c.clusterId === selectedCluster);

  if (data.length === 0 && !loading) {
    return (
      <Card title="集群资源" size="small">
        <Empty description="暂无集群数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      </Card>
    );
  }

  return (
    <Card
      title="集群资源"
      size="small"
      extra={
        data.length > 1 && (
          <Select
            value={selectedCluster}
            onChange={setSelectedCluster}
            style={{ width: 120 }}
            size="small"
            options={data.map((c) => ({
              label: c.clusterName,
              value: c.clusterId,
            }))}
          />
        )
      }
      loading={loading}
    >
      {cluster && (
        <div className="space-y-4">
          {/* CPU 使用率 */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <DashboardOutlined className="text-blue-500" />
                <Text strong>CPU</Text>
              </div>
              <Text type="secondary" className="text-xs">
                {cluster.cpu.usagePercent.toFixed(1)}%
              </Text>
            </div>
            <Progress
              percent={cluster.cpu.usagePercent}
              size="small"
              format={() => `${cluster.cpu.usage.toFixed(1)} / ${cluster.cpu.allocatable.toFixed(1)} 核`}
            />
            <div className="flex justify-between mt-1">
              <Text type="secondary" className="text-xs">
                已请求: {cluster.cpu.requested.toFixed(1)} 核
              </Text>
            </div>
          </div>

          {/* 内存使用率 */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <MemoryOutlined className="text-green-500" />
                <Text strong>内存</Text>
              </div>
              <Text type="secondary" className="text-xs">
                {cluster.memory.usagePercent.toFixed(1)}%
              </Text>
            </div>
            <Progress
              percent={cluster.memory.usagePercent}
              size="small"
              strokeColor="#52c41a"
              format={() => `${(cluster.memory.usage / 1024).toFixed(1)} / ${(cluster.memory.allocatable / 1024).toFixed(1)} GB`}
            />
            <div className="flex justify-between mt-1">
              <Text type="secondary" className="text-xs">
                已请求: {(cluster.memory.requested / 1024).toFixed(1)} GB
              </Text>
            </div>
          </div>

          {/* Pod 统计 */}
          <div className="pt-2 border-t border-gray-100">
            <div className="flex items-center gap-2 mb-2">
              <CloudOutlined className="text-purple-500" />
              <Text strong>Pod</Text>
            </div>
            <div className="grid grid-cols-4 gap-2 text-center">
              <div>
                <Text strong className="text-lg">{cluster.pods.total}</Text>
                <br />
                <Text type="secondary" className="text-xs">总数</Text>
              </div>
              <div>
                <Text strong className="text-lg text-green-500">{cluster.pods.running}</Text>
                <br />
                <Text type="secondary" className="text-xs">运行中</Text>
              </div>
              <div>
                <Text strong className="text-lg text-yellow-500">{cluster.pods.pending}</Text>
                <br />
                <Text type="secondary" className="text-xs">等待中</Text>
              </div>
              <div>
                <Text strong className="text-lg text-red-500">{cluster.pods.failed}</Text>
                <br />
                <Text type="secondary" className="text-xs">失败</Text>
              </div>
            </div>
          </div>
        </div>
      )}
    </Card>
  );
};

export default ClusterResourceCard;
```

- [ ] **Step 2: Commit**

```bash
git add web/src/components/Dashboard/ClusterResourceCard.tsx
git commit -m "feat: add ClusterResourceCard component

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 7.3: Create OperationsCard Component

**Files:**
- Create: `web/src/components/Dashboard/OperationsCard.tsx`

- [ ] **Step 1: Write the component**

```tsx
import React from 'react';
import { Card, Badge, List, Typography, Tag, Space } from 'antd';
import {
  RocketOutlined,
  ThunderboltOutlined,
  WarningOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import type { OperationsOverview } from '../../api/modules/dashboard';

const { Text } = Typography;

interface Props {
  data: OperationsOverview;
  loading?: boolean;
}

const OperationsCard: React.FC<Props> = ({ data, loading }) => {
  const items = [
    {
      key: 'deployments',
      icon: <RocketOutlined />,
      label: '部署状态',
      badge: data.deployments.running + data.deployments.pendingApproval,
      details: [
        { label: '进行中', value: data.deployments.running },
        { label: '待审批', value: data.deployments.pendingApproval },
        { label: '今日成功', value: data.deployments.todaySuccess, type: 'success' },
        { label: '今日失败', value: data.deployments.todayFailed, type: 'error' },
      ],
    },
    {
      key: 'cicd',
      icon: <ThunderboltOutlined />,
      label: 'CI/CD',
      badge: data.cicd.running + data.cicd.queued,
      details: [
        { label: '运行中', value: data.cicd.running },
        { label: '排队中', value: data.cicd.queued },
        { label: '今日成功', value: data.cicd.success, type: 'success' },
        { label: '今日失败', value: data.cicd.failed, type: 'error' },
      ],
    },
    {
      key: 'issue_pods',
      icon: <WarningOutlined />,
      label: '异常 Pod',
      badge: data.issuePods.total,
      badgeStatus: data.issuePods.total > 0 ? 'error' : 'success',
      details: Object.entries(data.issuePods.byType).map(([type, count]) => ({
        label: type,
        value: count,
      })),
    },
  ];

  return (
    <Card title="运行状态" size="small" loading={loading}>
      <List
        dataSource={items}
        renderItem={(item) => (
          <List.Item className="!py-2 !px-0">
            <div className="w-full">
              <div className="flex items-center justify-between mb-2">
                <Space>
                  {item.icon}
                  <Text strong>{item.label}</Text>
                </Space>
                <Badge
                  count={item.badge}
                  status={item.badgeStatus || 'processing'}
                  showZero
                />
              </div>
              <div className="grid grid-cols-2 gap-2 pl-6">
                {item.details.slice(0, 4).map((detail, idx) => (
                  <div key={idx} className="flex justify-between">
                    <Text type="secondary" className="text-xs">{detail.label}</Text>
                    <Text
                      strong
                      className="text-xs"
                      style={detail.type === 'success' ? { color: '#52c41a' } : detail.type === 'error' ? { color: '#ff4d4f' } : {}}
                    >
                      {detail.value}
                    </Text>
                  </div>
                ))}
              </div>
            </div>
          </List.Item>
        )}
      />
    </Card>
  );
};

export default OperationsCard;
```

- [ ] **Step 2: Commit**

```bash
git add web/src/components/Dashboard/OperationsCard.tsx
git commit -m "feat: add OperationsCard component

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 8: Dashboard Page Update

### Task 8.1: Update Dashboard Page Layout

**Files:**
- Modify: `web/src/pages/Dashboard/Dashboard.tsx`

- [ ] **Step 1: Update imports and state**

Replace the imports section with:

```tsx
import React, { useCallback, useEffect, useState } from 'react';
import { Col, Row, message } from 'antd';
import { useInterval } from 'ahooks';
import { useNavigate } from 'react-router-dom';
import { Api } from '../../api';
import type { OverviewResponseV2, TimeRange } from '../../api/modules/dashboard';
import TimeRangeSelector from '../../components/Dashboard/TimeRangeSelector';
import HealthCard from '../../components/Dashboard/HealthCard';
import TimeseriesChart from '../../components/Dashboard/TimeseriesChart';
import AlertPanel from '../../components/Dashboard/AlertPanel';
import EventStream from '../../components/Dashboard/EventStream';
import AIActivityCard from '../../components/Dashboard/AIActivityCard';
import WorkloadHealthCard from '../../components/Dashboard/WorkloadHealthCard';
import ClusterResourceCard from '../../components/Dashboard/ClusterResourceCard';
import OperationsCard from '../../components/Dashboard/OperationsCard';
```

- [ ] **Step 2: Update empty state and component**

Replace the `emptyOverview` and component body with:

```tsx
const emptyOverview: OverviewResponseV2 = {
  health: {
    hosts: { total: 0, healthy: 0, degraded: 0, unhealthy: 0, offline: 0 },
    clusters: { total: 0, healthy: 0, degraded: 0, unhealthy: 0, offline: 0 },
    applications: { total: 0, healthy: 0, degraded: 0, unhealthy: 0, offline: 0 },
    workloads: {
      deployments: { total: 0, healthy: 0 },
      statefulsets: { total: 0, healthy: 0 },
      daemonsets: { total: 0, healthy: 0 },
      services: 0,
      ingresses: 0,
    },
  },
  resources: {
    hosts: [],
    clusters: [],
  },
  operations: {
    deployments: { running: 0, pendingApproval: 0, todayTotal: 0, todaySuccess: 0, todayFailed: 0 },
    cicd: { running: 0, queued: 0, todayTotal: 0, success: 0, failed: 0 },
    issuePods: { total: 0, byType: {} },
  },
  alerts: { firing: 0, recent: [] },
  events: [],
  ai: {
    stats: { sessionCount: 0, tokenCount: 0, avgDurationMs: 0, successRate: 0 },
    sessions: [],
    byScene: {},
  },
};

const Dashboard: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [timeRange, setTimeRange] = useState<TimeRange>('1h');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [overview, setOverview] = useState<OverviewResponseV2>(emptyOverview);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await Api.dashboard.getOverviewV2(timeRange);
      setOverview(response.data || emptyOverview);
    } catch (error) {
      message.error('加载主控台概览失败');
    } finally {
      setLoading(false);
    }
  }, [timeRange]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    const handler = () => {
      load();
    };
    window.addEventListener('project:changed', handler as EventListener);
    return () => window.removeEventListener('project:changed', handler as EventListener);
  }, [load]);

  useInterval(() => {
    load();
  }, autoRefresh ? 60000 : undefined);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">主控台</h1>
          <p className="text-sm text-gray-500 mt-1">实时监控系统运行状态</p>
        </div>
        <TimeRangeSelector
          value={timeRange}
          autoRefresh={autoRefresh}
          loading={loading}
          onChange={setTimeRange}
          onRefresh={load}
          onAutoRefreshChange={setAutoRefresh}
        />
      </div>

      {/* 健康概览 - 4 列 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <HealthCard title="主机健康" data={overview.health.hosts} onClick={() => navigate('/hosts')} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <HealthCard title="集群健康" data={overview.health.clusters} onClick={() => navigate('/deployment/infrastructure/clusters')} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <HealthCard title="应用健康" data={overview.health.applications} onClick={() => navigate('/services')} />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <WorkloadHealthCard data={overview.health.workloads} loading={loading} />
        </Col>
      </Row>

      {/* 资源使用 - 2 列 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} xl={12}>
          <TimeseriesChart title="CPU 使用率" series={overview.resources.hosts} loading={loading} />
        </Col>
        <Col xs={24} xl={12}>
          <ClusterResourceCard data={overview.resources.clusters} loading={loading} />
        </Col>
      </Row>

      {/* 运行状态 + 告警 + AI - 3 列 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}>
          <OperationsCard data={overview.operations} loading={loading} />
        </Col>
        <Col xs={24} md={8}>
          <AlertPanel alerts={overview.alerts} loading={loading} />
        </Col>
        <Col xs={24} md={8}>
          <AIActivityCard data={overview.ai} loading={loading} />
        </Col>
      </Row>

      {/* 事件流 - 全宽 */}
      <Row gutter={[16, 16]}>
        <Col xs={24}>
          <EventStream events={overview.events} loading={loading} />
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
```

- [ ] **Step 3: Commit**

```bash
git add web/src/pages/Dashboard/Dashboard.tsx
git commit -m "feat: update Dashboard page with new layout and components

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 9: Integration

### Task 9.1: Initialize Collector in ServiceContext

**Files:**
- Modify: `internal/svc/svc.go`

- [ ] **Step 1: Add collector initialization**

Find where other services are initialized and add:

```go
// 在 svc.go 中导入 dashboard 包
import "github.com/cy77cc/OpsPilot/internal/service/dashboard"

// 在 ServiceContext 初始化后添加
if c.Metrics.Enable {
    collector := dashboard.NewCollector(svcCtx)
    go collector.Start()
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/svc/svc.go
git commit -m "feat: initialize dashboard collector on startup

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 9.2: Run Tests and Fix Issues

- [ ] **Step 1: Run backend tests**

```bash
make test
```

Expected: All tests pass

- [ ] **Step 2: Run frontend build**

```bash
cd web && pnpm build
```

Expected: Build succeeds

- [ ] **Step 3: Run database migration**

```bash
make migrate-up
```

Expected: Tables created successfully

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete dashboard observability enhancement

- Fix AI activity data source
- Add cluster resource monitoring
- Add workload health overview
- Add deployment/CI status
- Add issue pod tracking

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Summary

This plan implements the dashboard observability enhancement in 9 chunks:

1. **Chunk 1**: Database models and migration
2. **Chunk 2**: API type definitions
3. **Chunk 3**: Data collector implementation
4. **Chunk 4**: Logic refactoring
5. **Chunk 5**: Handler and routes
6. **Chunk 6**: Frontend types
7. **Chunk 7**: Frontend components
8. **Chunk 8**: Dashboard page update
9. **Chunk 9**: Integration and testing
