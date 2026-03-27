// Package model 提供数据库模型定义。
//
// 本文件定义集群节点相关的数据模型。
package model

import "time"

// ClusterNode 集群节点表模型，存储从 K8s API 同步的节点信息。
//
// 表名: cluster_nodes
// 索引:
//   - uk_cluster_node: (cluster_id, name) 唯一索引
// 关联:
//   - Cluster (多对一，非外键)
//   - Node/Host (一对一，非外键)
//
// 字段说明:
//   - Role: control-plane / worker / etcd
//   - Status: ready / notready / unknown
type ClusterNode struct {
	ID               uint       `gorm:"primaryKey;column:id" json:"id"`                                                // 节点 ID
	ClusterID        uint       `gorm:"column:cluster_id;not null;index;uniqueIndex:uk_cluster_node,priority:1" json:"cluster_id"`         // 所属集群 ID
	HostID           *uint      `gorm:"column:host_id;index" json:"host_id"`                                           // 关联主机 ID (可选)
	Name             string     `gorm:"column:name;type:varchar(64);not null;uniqueIndex:uk_cluster_node,priority:2" json:"name"`          // 节点名称 (K8s 节点名)
	IP               string     `gorm:"column:ip;type:varchar(45);not null" json:"ip"`                                 // 节点内网 IP
	Role             string     `gorm:"column:role;type:varchar(32);not null" json:"role"`                             // 节点角色: control-plane/worker/etcd
	Status           string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`                   // 节点状态: ready/notready/unknown
	KubeletVersion   string     `gorm:"column:kubelet_version;type:varchar(32)" json:"kubelet_version"`                // Kubelet 版本
	KubeProxyVersion string     `gorm:"column:kube_proxy_version;type:varchar(32)" json:"kube_proxy_version"`          // Kube-proxy 版本
	ContainerRuntime string     `gorm:"column:container_runtime;type:varchar(32)" json:"container_runtime"`            // 容器运行时 (containerd/docker)
	OSImage          string     `gorm:"column:os_image;type:varchar(128)" json:"os_image"`                             // 操作系统镜像
	KernelVersion    string     `gorm:"column:kernel_version;type:varchar(64)" json:"kernel_version"`                  // 内核版本
	AllocatableCPU   string     `gorm:"column:allocatable_cpu;type:varchar(16)" json:"allocatable_cpu"`                // 可分配 CPU
	AllocatableMem   string     `gorm:"column:allocatable_mem;type:varchar(16)" json:"allocatable_mem"`                // 可分配内存
	AllocatablePods  int        `gorm:"column:allocatable_pods;default:0" json:"allocatable_pods"`                     // 可分配 Pod 数量
	Labels           string     `gorm:"column:labels;type:json" json:"labels"`                                         // 节点标签 (JSON)
	Taints           string     `gorm:"column:taints;type:json" json:"taints"`                                         // 节点污点 (JSON)
	Conditions       string     `gorm:"column:conditions;type:json" json:"conditions"`                                 // 节点条件 (JSON): [{type, status, reason, message}]
	JoinedAt         *time.Time `gorm:"column:joined_at" json:"joined_at"`                                              // 加入集群时间
	LastSeenAt       *time.Time `gorm:"column:last_seen_at" json:"last_seen_at"`                                        // 最后心跳时间
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`                             // 创建时间
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                             // 更新时间
}

// TableName 返回集群节点表名。
func (ClusterNode) TableName() string {
	return "cluster_nodes"
}
