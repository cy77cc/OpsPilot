// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件定义集群服务的数据传输对象 (DTO) 和请求/响应结构。
package cluster

import (
	"time"
)

// ClusterNode 集群节点响应结构。
type ClusterNode struct {
	ID               uint      `json:"id"`                // 节点 ID
	ClusterID        uint      `json:"cluster_id"`        // 所属集群 ID
	HostID           *uint     `json:"host_id"`           // 关联主机 ID
	Name             string    `json:"name"`              // 节点名称
	IP               string    `json:"ip"`                // 节点 IP
	Role             string    `json:"role"`              // 节点角色: control-plane/worker
	Status           string    `json:"status"`            // 节点状态
	KubeletVersion   string    `json:"kubelet_version"`   // Kubelet 版本
	ContainerRuntime string    `json:"container_runtime"` // 容器运行时
	OSImage          string    `json:"os_image"`          // 操作系统镜像
	KernelVersion    string    `json:"kernel_version"`    // 内核版本
	AllocatableCPU   string    `json:"allocatable_cpu"`   // 可分配 CPU
	AllocatableMem   string    `json:"allocatable_mem"`   // 可分配内存
	Labels           string    `json:"labels"`            // 节点标签
	CreatedAt        time.Time `json:"created_at"`        // 创建时间
	UpdatedAt        time.Time `json:"updated_at"`        // 更新时间
}

// ClusterDetail 集群详情响应结构。
type ClusterDetail struct {
	ID             uint       `json:"id"`              // 集群 ID
	Name           string     `json:"name"`            // 集群名称
	Description    string     `json:"description"`     // 集群描述
	Version        string     `json:"version"`         // 集群版本
	K8sVersion     string     `json:"k8s_version"`     // Kubernetes 版本
	Status         string     `json:"status"`          // 集群状态: active/inactive/error
	Source         string     `json:"source"`          // 来源: platform_managed/external_managed
	Type           string     `json:"type"`            // 类型: kubernetes/openshift
	NodeCount      int        `json:"node_count"`      // 节点数量
	Endpoint       string     `json:"endpoint"`        // API Server 地址
	PodCIDR        string     `json:"pod_cidr"`        // Pod 网段
	ServiceCIDR    string     `json:"service_cidr"`    // Service 网段
	ManagementMode string     `json:"management_mode"` // 管理模式
	CredentialID   *uint      `json:"credential_id"`   // 凭证 ID
	LastSyncAt     *time.Time `json:"last_sync_at"`    // 最后同步时间
	CreatedAt      time.Time  `json:"created_at"`      // 创建时间
	UpdatedAt      time.Time  `json:"updated_at"`      // 更新时间
}

// ClusterListItem 集群列表项响应结构。
type ClusterListItem struct {
	ID          uint       `json:"id"`          // 集群 ID
	Name        string     `json:"name"`        // 集群名称
	Version     string     `json:"version"`     // 集群版本
	K8sVersion  string     `json:"k8s_version"` // Kubernetes 版本
	Status      string     `json:"status"`      // 集群状态
	Source      string     `json:"source"`      // 来源
	NodeCount   int        `json:"node_count"`  // 节点数量
	Endpoint    string     `json:"endpoint"`    // API Server 地址
	Description string     `json:"description"` // 集群描述
	LastSyncAt  *time.Time `json:"last_sync_at"` // 最后同步时间
	CreatedAt   time.Time  `json:"created_at"`  // 创建时间
}

// ClusterCreateReq 集群创建请求结构。
type ClusterCreateReq struct {
	Name          string `json:"name" binding:"required"` // 集群名称 (必填)
	Description   string `json:"description"`             // 集群描述
	Kubeconfig    string `json:"kubeconfig"`              // Kubeconfig 内容
	Endpoint      string `json:"endpoint"`                // API Server 地址
	CACert        string `json:"ca_cert"`                 // CA 证书
	Cert          string `json:"cert"`                    // 客户端证书
	Key           string `json:"key"`                     // 客户端私钥
	Token         string `json:"token"`                   // 认证 Token
	SkipTLSVerify bool   `json:"skip_tls_verify"`         // 是否跳过 TLS 验证
	AuthMethod    string `json:"auth_method"`             // 认证方式: kubeconfig/certificate/token
}

// ClusterUpdateReq 集群更新请求结构。
type ClusterUpdateReq struct {
	Name        string `json:"name"`        // 集群名称
	Description string `json:"description"` // 集群描述
}

// ClusterTestResp 集群连通性测试响应结构。
type ClusterTestResp struct {
	ClusterID uint   `json:"cluster_id"`       // 集群 ID
	Connected bool   `json:"connected"`        // 是否连通
	Message   string `json:"message"`          // 连通消息
	Version   string `json:"version,omitempty"` // Kubernetes 版本
	LatencyMS int64  `json:"latency_ms,omitempty"` // 延迟 (毫秒)
	LastError string `json:"last_error,omitempty"` // 最后错误信息
}

// BootstrapStepStatus 引导步骤状态结构。
type BootstrapStepStatus struct {
	Name       string     `json:"name"`                  // 步骤名称
	Status     string     `json:"status"`                // 状态: pending/running/succeeded/failed
	Message    string     `json:"message,omitempty"`     // 步骤消息
	StartedAt  *time.Time `json:"started_at,omitempty"`  // 开始时间
	FinishedAt *time.Time `json:"finished_at,omitempty"` // 完成时间
	HostID     uint       `json:"host_id,omitempty"`     // 主机 ID
	Output     string     `json:"output,omitempty"`      // 输出内容
}

// BootstrapTaskDetail 引导任务详情结构。
type BootstrapTaskDetail struct {
	ID                   string                `json:"id"`                    // 任务 ID
	Name                 string                `json:"name"`                  // 任务名称
	ClusterID            *uint                 `json:"cluster_id"`            // 集群 ID
	K8sVersion           string                `json:"k8s_version"`           // Kubernetes 版本
	VersionChannel       string                `json:"version_channel"`       // 版本渠道
	RepoMode             string                `json:"repo_mode"`             // 仓库模式: online/offline
	RepoURL              string                `json:"repo_url"`              // 仓库地址
	ImageRepository      string                `json:"image_repository"`      // 镜像仓库
	EndpointMode         string                `json:"endpoint_mode"`         // 端点模式: nodeIP/vip
	ControlPlaneEndpoint string                `json:"control_plane_endpoint"` // 控制面端点
	VIPProvider          string                `json:"vip_provider"`          // VIP 提供者: kube-vip/keepalived
	EtcdMode             string                `json:"etcd_mode"`             // etcd 模式: stacked/external
	CNI                  string                `json:"cni"`                   // CNI 插件
	PodCIDR              string                `json:"pod_cidr"`              // Pod 网段
	ServiceCIDR          string                `json:"service_cidr"`          // Service 网段
	Status               string                `json:"status"`                // 任务状态
	Steps                []BootstrapStepStatus `json:"steps"`                 // 步骤列表
	CurrentStep          int                   `json:"current_step"`          // 当前步骤索引
	ErrorMessage         string                `json:"error_message,omitempty"` // 错误消息
	ResolvedConfigJSON   string                `json:"resolved_config_json,omitempty"` // 解析后配置
	DiagnosticsJSON      string                `json:"diagnostics_json,omitempty"` // 诊断信息
	CreatedAt            time.Time             `json:"created_at"`            // 创建时间
	UpdatedAt            time.Time             `json:"updated_at"`            // 更新时间
}

// BootstrapProfileExternalEtcd 外部 etcd 配置结构。
type BootstrapProfileExternalEtcd struct {
	Endpoints []string `json:"endpoints"` // etcd 端点列表
	CACert    string   `json:"ca_cert"`   // CA 证书
	Cert      string   `json:"cert"`      // 客户端证书
	Key       string   `json:"key"`       // 客户端私钥
}

// BootstrapProfileCreateReq 引导配置创建请求结构。
type BootstrapProfileCreateReq struct {
	Name                 string                        `json:"name" binding:"required"` // 配置名称 (必填)
	Description          string                        `json:"description"`             // 配置描述
	VersionChannel       string                        `json:"version_channel"`         // 版本渠道
	K8sVersion           string                        `json:"k8s_version"`             // Kubernetes 版本
	RepoMode             string                        `json:"repo_mode"`               // 仓库模式
	RepoURL              string                        `json:"repo_url"`                // 仓库地址
	ImageRepository      string                        `json:"image_repository"`        // 镜像仓库
	EndpointMode         string                        `json:"endpoint_mode"`           // 端点模式
	ControlPlaneEndpoint string                        `json:"control_plane_endpoint"`  // 控制面端点
	VIPProvider          string                        `json:"vip_provider"`            // VIP 提供者
	EtcdMode             string                        `json:"etcd_mode"`               // etcd 模式
	ExternalEtcd         *BootstrapProfileExternalEtcd `json:"external_etcd"`           // 外部 etcd 配置
}

// BootstrapProfileUpdateReq 引导配置更新请求结构。
type BootstrapProfileUpdateReq struct {
	Description          string                        `json:"description"`             // 配置描述
	VersionChannel       string                        `json:"version_channel"`         // 版本渠道
	K8sVersion           string                        `json:"k8s_version"`             // Kubernetes 版本
	RepoMode             string                        `json:"repo_mode"`               // 仓库模式
	RepoURL              string                        `json:"repo_url"`                // 仓库地址
	ImageRepository      string                        `json:"image_repository"`        // 镜像仓库
	EndpointMode         string                        `json:"endpoint_mode"`           // 端点模式
	ControlPlaneEndpoint string                        `json:"control_plane_endpoint"`  // 控制面端点
	VIPProvider          string                        `json:"vip_provider"`            // VIP 提供者
	EtcdMode             string                        `json:"etcd_mode"`               // etcd 模式
	ExternalEtcd         *BootstrapProfileExternalEtcd `json:"external_etcd"`           // 外部 etcd 配置
}

// BootstrapProfileItem 引导配置列表项结构。
type BootstrapProfileItem struct {
	ID                   uint        `json:"id"`                    // 配置 ID
	Name                 string      `json:"name"`                  // 配置名称
	Description          string      `json:"description"`           // 配置描述
	VersionChannel       string      `json:"version_channel"`       // 版本渠道
	K8sVersion           string      `json:"k8s_version"`           // Kubernetes 版本
	RepoMode             string      `json:"repo_mode"`             // 仓库模式
	RepoURL              string      `json:"repo_url"`              // 仓库地址
	ImageRepository      string      `json:"image_repository"`      // 镜像仓库
	EndpointMode         string      `json:"endpoint_mode"`         // 端点模式
	ControlPlaneEndpoint string      `json:"control_plane_endpoint"` // 控制面端点
	VIPProvider          string      `json:"vip_provider"`          // VIP 提供者
	EtcdMode             string      `json:"etcd_mode"`             // etcd 模式
	ExternalEtcd         interface{} `json:"external_etcd,omitempty"` // 外部 etcd 配置
	CreatedAt            time.Time   `json:"created_at"`            // 创建时间
	UpdatedAt            time.Time   `json:"updated_at"`            // 更新时间
}
