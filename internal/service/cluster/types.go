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
	ControlPlaneEndpoint string                        `json:"control_plane_endpoint"` // 控制面端点
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

// PolicyReference 表示发布记录中的策略引用。
type PolicyReference struct {
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"` // API 版本
	Kind       string `json:"kind,omitempty" yaml:"kind,omitempty"`             // 资源类型
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`             // 策略名称
	Namespace  string `json:"namespace,omitempty" yaml:"namespace,omitempty"`   // 策略命名空间
}

// PolicyTargetCluster 表示目标集群信息。
type PolicyTargetCluster struct {
	ClusterID  uint   `json:"clusterId,omitempty" yaml:"clusterId,omitempty"`   // 集群 ID
	CNIType    string `json:"cniType,omitempty" yaml:"cniType,omitempty"`       // CNI 类型
	CNIVersion string `json:"cniVersion,omitempty" yaml:"cniVersion,omitempty"` // CNI 版本
}

// PolicyReleaseStatus 表示发布状态与风险信息。
type PolicyReleaseStatus struct {
	Phase     PolicyReleaseState `json:"phase,omitempty" yaml:"phase,omitempty"`         // 发布阶段
	RiskScore int                `json:"riskScore,omitempty" yaml:"riskScore,omitempty"` // 风险分
	RiskLevel PolicyRiskLevel    `json:"riskLevel,omitempty" yaml:"riskLevel,omitempty"` // 风险等级
}

// PolicyIssue 表示仿真发现的问题。
type PolicyIssue struct {
	Code       string              `json:"code,omitempty" yaml:"code,omitempty"`             // 问题码
	Message    string              `json:"message,omitempty" yaml:"message,omitempty"`       // 问题说明
	Severity   PolicyIssueSeverity `json:"severity,omitempty" yaml:"severity,omitempty"`     // 严重级别
	Suggestion string              `json:"suggestion,omitempty" yaml:"suggestion,omitempty"` // 修复建议
}

// PolicyWarning 表示非阻断告警。
type PolicyWarning struct {
	Code    string `json:"code,omitempty" yaml:"code,omitempty"`       // 告警码
	Message string `json:"message,omitempty" yaml:"message,omitempty"` // 告警说明
}

// PolicyImpactSummary 表示仿真的影响面摘要。
type PolicyImpactSummary struct {
	AffectedPods       int      `json:"affectedPods,omitempty" yaml:"affectedPods,omitempty"`             // 受影响 Pod 数
	AffectedNamespaces []string `json:"affectedNamespaces,omitempty" yaml:"affectedNamespaces,omitempty"` // 受影响命名空间
	NewDeniedFlows     []string `json:"newDeniedFlows,omitempty" yaml:"newDeniedFlows,omitempty"`         // 新增阻断流量
}

// PolicySimulationStatus 表示仿真结果摘要。
type PolicySimulationStatus struct {
	JobID          string              `json:"jobId,omitempty" yaml:"jobId,omitempty"`                   // 仿真任务 ID
	PassedAt       *time.Time          `json:"passedAt,omitempty" yaml:"passedAt,omitempty"`             // 通过时间
	BlockingIssues []PolicyIssue       `json:"blockingIssues,omitempty" yaml:"blockingIssues,omitempty"` // 阻断问题
	Warnings       []PolicyWarning     `json:"warnings,omitempty" yaml:"warnings,omitempty"`             // 告警列表
	ImpactSummary  PolicyImpactSummary `json:"impactSummary,omitempty" yaml:"impactSummary,omitempty"`   // 影响面摘要
}

// PolicyApprovalStatus 表示审批阶段摘要。
type PolicyApprovalStatus struct {
	Required      bool       `json:"required,omitempty" yaml:"required,omitempty"`           // 是否需要审批
	Approvers     []string   `json:"approvers,omitempty" yaml:"approvers,omitempty"`         // 审批人列表
	ApprovedAt    *time.Time `json:"approvedAt,omitempty" yaml:"approvedAt,omitempty"`       // 审批通过时间
	ApprovalToken string     `json:"approvalToken,omitempty" yaml:"approvalToken,omitempty"` // 审批令牌
}

// PolicyAuditStatus 表示发布审计时间线。
type PolicyAuditStatus struct {
	CreatedAt  *time.Time `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`   // 创建时间
	CreatedBy  uint       `json:"createdBy,omitempty" yaml:"createdBy,omitempty"`   // 创建人
	AppliedAt  *time.Time `json:"appliedAt,omitempty" yaml:"appliedAt,omitempty"`   // 应用时间
	RollbackAt *time.Time `json:"rollbackAt,omitempty" yaml:"rollbackAt,omitempty"` // 回滚时间
}
