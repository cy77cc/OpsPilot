// Package model 提供数据库模型定义。
//
// 本文件定义主控台可观测性缓存相关的数据模型。
package model

import "time"

// ClusterResourceSnapshot 集群资源快照表，存储定期采集的集群资源使用数据。
//
// 表名: cluster_resource_snapshots
// 用途: 缓存集群 CPU/内存/Pod 资源数据，避免频繁调用 K8s API
// 采集频率: 每 5 分钟
type ClusterResourceSnapshot struct {
	ID                  uint64    `gorm:"primaryKey;column:id;autoIncrement" json:"id"`                                      // 快照 ID
	ClusterID           uint      `gorm:"column:cluster_id;not null;index:idx_cluster_collected,priority:1" json:"cluster_id"` // 集群 ID
	CPUAllocatableCores float64   `gorm:"column:cpu_allocatable_cores;type:decimal(10,2);not null;default:0" json:"cpu_allocatable_cores"`
	CPURequestedCores   float64   `gorm:"column:cpu_requested_cores;type:decimal(10,2);not null;default:0" json:"cpu_requested_cores"`
	CPULimitCores       float64   `gorm:"column:cpu_limit_cores;type:decimal(10,2);not null;default:0" json:"cpu_limit_cores"`
	CPUUsageCores       float64   `gorm:"column:cpu_usage_cores;type:decimal(10,2);not null;default:0" json:"cpu_usage_cores"`
	MemoryAllocatableMB int64     `gorm:"column:memory_allocatable_mb;not null;default:0" json:"memory_allocatable_mb"`
	MemoryRequestedMB   int64     `gorm:"column:memory_requested_mb;not null;default:0" json:"memory_requested_mb"`
	MemoryLimitMB       int64     `gorm:"column:memory_limit_mb;not null;default:0" json:"memory_limit_mb"`
	MemoryUsageMB       int64     `gorm:"column:memory_usage_mb;not null;default:0" json:"memory_usage_mb"`
	PodTotal            int       `gorm:"column:pod_total;not null;default:0" json:"pod_total"`         // Pod 总数
	PodRunning          int       `gorm:"column:pod_running;not null;default:0" json:"pod_running"`     // 运行中 Pod 数
	PodPending          int       `gorm:"column:pod_pending;not null;default:0" json:"pod_pending"`     // 等待中 Pod 数
	PodFailed           int       `gorm:"column:pod_failed;not null;default:0" json:"pod_failed"`       // 失败 Pod 数
	PVCount             int       `gorm:"column:pv_count;not null;default:0" json:"pv_count"`           // PV 数量
	PVCCount            int       `gorm:"column:pvc_count;not null;default:0" json:"pvc_count"`         // PVC 数量
	StorageUsedGB       float64   `gorm:"column:storage_used_gb;type:decimal(10,2);not null;default:0" json:"storage_used_gb"`
	CollectedAt         time.Time `gorm:"column:collected_at;not null;index:idx_cluster_collected,priority:2" json:"collected_at"` // 采集时间
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                                      // 创建时间
}

// TableName 返回集群资源快照表名。
func (ClusterResourceSnapshot) TableName() string { return "cluster_resource_snapshots" }

// K8sWorkloadStats K8s 工作负载统计表，存储 Deployment/StatefulSet/DaemonSet 等统计。
//
// 表名: k8s_workload_stats
// 用途: 缓存工作负载健康状态，用于主控台快速展示
// 采集频率: 每 1 分钟
type K8sWorkloadStats struct {
	ID                 uint      `gorm:"primaryKey;column:id;autoIncrement" json:"id"`                                           // 统计 ID
	ClusterID          uint      `gorm:"column:cluster_id;not null;index:idx_cluster_ns_collected,priority:1" json:"cluster_id"`  // 集群 ID
	Namespace          string    `gorm:"column:namespace;type:varchar(128);not null;default:'';index:idx_cluster_ns_collected,priority:2" json:"namespace"`
	DeploymentTotal    int       `gorm:"column:deployment_total;not null;default:0" json:"deployment_total"`     // Deployment 总数
	DeploymentHealthy  int       `gorm:"column:deployment_healthy;not null;default:0" json:"deployment_healthy"` // 健康 Deployment 数
	StatefulSetTotal   int       `gorm:"column:statefulset_total;not null;default:0" json:"statefulset_total"`   // StatefulSet 总数
	StatefulSetHealthy int       `gorm:"column:statefulset_healthy;not null;default:0" json:"statefulset_healthy"`
	DaemonSetTotal     int       `gorm:"column:daemonset_total;not null;default:0" json:"daemonset_total"`     // DaemonSet 总数
	DaemonSetHealthy   int       `gorm:"column:daemonset_healthy;not null;default:0" json:"daemonset_healthy"` // 健康 DaemonSet 数
	ServiceCount       int       `gorm:"column:service_count;not null;default:0" json:"service_count"`         // Service 数量
	IngressCount       int       `gorm:"column:ingress_count;not null;default:0" json:"ingress_count"`         // Ingress 数量
	CollectedAt        time.Time `gorm:"column:collected_at;not null;index:idx_cluster_ns_collected,priority:3" json:"collected_at"` // 采集时间
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                                        // 创建时间
}

// TableName 返回 K8s 工作负载统计表名。
func (K8sWorkloadStats) TableName() string { return "k8s_workload_stats" }

// K8sIssuePod 异常 Pod 缓存表，存储有问题的 Pod 信息。
//
// 表名: k8s_issue_pods
// 用途: 快速展示集群中的异常 Pod 列表
// 采集频率: 每 30 秒
type K8sIssuePod struct {
	ID          uint      `gorm:"primaryKey;column:id;autoIncrement" json:"id"`                                              // 记录 ID
	ClusterID   uint      `gorm:"column:cluster_id;not null;uniqueIndex:uk_cluster_ns_pod,priority:1" json:"cluster_id"`      // 集群 ID
	Namespace   string    `gorm:"column:namespace;type:varchar(128);not null;uniqueIndex:uk_cluster_ns_pod,priority:2" json:"namespace"`
	PodName     string    `gorm:"column:pod_name;type:varchar(256);not null;uniqueIndex:uk_cluster_ns_pod,priority:3" json:"pod_name"`
	IssueType   string    `gorm:"column:issue_type;type:varchar(64);not null;index" json:"issue_type"`   // 问题类型
	IssueReason string    `gorm:"column:issue_reason;type:varchar(256);not null" json:"issue_reason"`    // 问题原因
	Message     string    `gorm:"column:message;type:text" json:"message"`                               // 详细信息
	FirstSeenAt time.Time `gorm:"column:first_seen_at;not null" json:"first_seen_at"`                    // 首次发现时间
	LastSeenAt  time.Time `gorm:"column:last_seen_at;not null;index" json:"last_seen_at"`                // 最后发现时间
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                    // 创建时间
}

// TableName 返回异常 Pod 表名。
func (K8sIssuePod) TableName() string { return "k8s_issue_pods" }

// IssueType 常量定义，用于标识异常 Pod 的问题类型。
const (
	IssueTypeCrashLoopBackOff   = "CrashLoopBackOff"      // 容器崩溃循环
	IssueTypeImagePullBackOff   = "ImagePullBackOff"      // 镜像拉取失败
	IssueTypeErrImagePull       = "ErrImagePull"          // 镜像拉取错误
	IssueTypeCreateContainerErr = "CreateContainerConfigError" // 容器创建配置错误
	IssueTypeRunContainerErr    = "RunContainerError"     // 容器运行错误
	IssueTypeOOMKilled          = "OOMKilled"             // 内存溢出被杀
	IssueTypeEvicted            = "Evicted"               // Pod 被驱逐
	IssueTypeUnknown            = "Unknown"               // 未知问题
)
