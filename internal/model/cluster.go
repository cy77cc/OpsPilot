// Package model 提供数据库模型定义。
//
// 本文件定义集群相关的数据模型，包括集群主体和启动配置模板。
package model

import (
	"time"
)

// Cluster 是集群表模型，存储 Kubernetes/OpenShift 集群信息。
//
// 表名: clusters
// 关联:
//   - Node (一对多，非外键)
//   - ClusterBootstrapProfile (多对一，非外键)
//
// 类型说明:
//   - Type: kubernetes / openshift
//   - Source: platform_managed (平台管理) / external_managed (外部托管)
//   - EnvType: development / staging / production
type Cluster struct {
	ID             uint       `gorm:"primaryKey;column:id" json:"id"`                                               // 集群 ID
	Name           string     `gorm:"column:name;type:varchar(64);not null;uniqueIndex" json:"name"`               // 集群名称 (唯一)
	Description    string     `gorm:"column:description;type:varchar(256)" json:"description"`                      // 集群描述
	Version        string     `gorm:"column:version;type:varchar(64)" json:"version"`                              // 集群版本
	Status         string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`                 // 状态: active/inactive/error
	Type           string     `gorm:"column:type;type:varchar(32);not null" json:"type"`                           // 类型: kubernetes/openshift
	Source         string     `gorm:"column:source;type:varchar(32);default:'platform_managed';index" json:"source"` // 来源: platform_managed/external_managed
	EnvType        string     `gorm:"column:env_type;type:varchar(32);not null;default:'development';index" json:"env_type"` // 环境: development/staging/production
	Endpoint       string     `gorm:"column:endpoint;type:varchar(256)" json:"endpoint"`                           // API Server 地址
	KubeConfig     string     `gorm:"column:kubeconfig;type:mediumtext" json:"-"`                                 // kubeconfig 内容 (逐步废弃)
	CACert         string     `gorm:"column:ca_cert;type:text" json:"-"`                                          // CA 证书
	Token          string     `gorm:"column:token;type:text" json:"-"`                                            // 认证 Token
	Nodes          string     `gorm:"column:nodes;type:json" json:"nodes"`                                         // 节点列表 (JSON)
	AuthMethod     string     `gorm:"column:auth_method;type:varchar(32)" json:"auth_method"`                      // 认证方式: kubeconfig/cert/token
	CredentialID   *uint      `gorm:"column:credential_id;index" json:"credential_id"`                            // 凭证 ID
	K8sVersion     string     `gorm:"column:k8s_version;type:varchar(32)" json:"k8s_version"`                     // Kubernetes 版本
	PodCIDR        string     `gorm:"column:pod_cidr;type:varchar(32)" json:"pod_cidr"`                           // Pod 网段
	ServiceCIDR    string     `gorm:"column:service_cidr;type:varchar(32)" json:"service_cidr"`                   // Service 网段
	ManagementMode string     `gorm:"column:management_mode;type:varchar(32);default:'k8s-only'" json:"management_mode"` // 管理模式
	LastSyncAt     *time.Time `gorm:"column:last_sync_at" json:"last_sync_at"`                                    // 最后同步时间
	CreatedBy      string     `gorm:"column:created_by;type:varchar(64)" json:"created_by"`                        // 创建人
	UpdatedBy      string     `gorm:"column:updated_by;type:varchar(64)" json:"updated_by"`                        // 更新人
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`                         // 创建时间
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                         // 更新时间
}

// TableName 返回集群表名。
func (Cluster) TableName() string {
	return "clusters"
}

// ClusterBootstrapProfile 是集群启动配置表模型，定义可复用的集群部署默认值。
//
// 表名: cluster_bootstrap_profiles
// 用途: 为新集群创建提供预设配置，简化部署流程
type ClusterBootstrapProfile struct {
	ID                   uint      `gorm:"primaryKey;column:id" json:"id"`                                           // 配置 ID
	Name                 string    `gorm:"column:name;type:varchar(128);not null;uniqueIndex" json:"name"`           // 配置名称 (唯一)
	Description          string    `gorm:"column:description;type:varchar(512)" json:"description"`                  // 配置描述
	VersionChannel       string    `gorm:"column:version_channel;type:varchar(32);default:'stable-1'" json:"version_channel"` // 版本渠道
	K8sVersion           string    `gorm:"column:k8s_version;type:varchar(32)" json:"k8s_version"`                  // Kubernetes 版本
	RepoMode             string    `gorm:"column:repo_mode;type:varchar(16);default:'online'" json:"repo_mode"`      // 仓库模式: online/offline
	RepoURL              string    `gorm:"column:repo_url;type:varchar(512)" json:"repo_url"`                       // 仓库地址
	ImageRepository      string    `gorm:"column:image_repository;type:varchar(256)" json:"image_repository"`       // 镜像仓库
	EndpointMode         string    `gorm:"column:endpoint_mode;type:varchar(16);default:'nodeIP'" json:"endpoint_mode"` // 端点模式: nodeIP/vip
	ControlPlaneEndpoint string    `gorm:"column:control_plane_endpoint;type:varchar(256)" json:"control_plane_endpoint"` // 控制面端点
	VIPProvider          string    `gorm:"column:vip_provider;type:varchar(32)" json:"vip_provider"`                // VIP 提供者: kube-vip/keepalived
	EtcdMode             string    `gorm:"column:etcd_mode;type:varchar(16);default:'stacked'" json:"etcd_mode"`    // etcd 模式: stacked/external
	ExternalEtcdJSON     string    `gorm:"column:external_etcd_json;type:longtext" json:"external_etcd_json"`       // 外部 etcd 配置 (JSON)
	CreatedBy            uint64    `gorm:"column:created_by;index" json:"created_by"`                               // 创建人 ID
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`               // 创建时间
	UpdatedAt            time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                      // 更新时间
}

// TableName 返回集群启动配置表名。
func (ClusterBootstrapProfile) TableName() string {
	return "cluster_bootstrap_profiles"
}
