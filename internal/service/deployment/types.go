// Package deployment 提供部署管理服务的请求和响应类型定义。
//
// 本文件包含所有 HTTP 请求参数和响应结构的类型定义。
package deployment

import "time"

// TargetNodeReq 是目标节点请求参数。
type TargetNodeReq struct {
	HostID uint   `json:"host_id"` // 主机 ID
	Role   string `json:"role"`    // 角色: manager/worker
	Weight int    `json:"weight"`  // 权重
}

// TargetUpsertReq 是创建或更新部署目标的请求参数。
type TargetUpsertReq struct {
	Name           string          `json:"name" binding:"required"`           // 目标名称
	TargetType     string          `json:"target_type" binding:"required"`    // 目标类型: k8s/compose
	RuntimeType    string          `json:"runtime_type"`                      // 运行时类型
	ClusterID      uint            `json:"cluster_id"`                        // 集群 ID
	ClusterSource  string          `json:"cluster_source"`                    // 集群来源
	CredentialID   uint            `json:"credential_id"`                     // 凭证 ID
	BootstrapJobID string          `json:"bootstrap_job_id"`                  // 引导任务 ID
	ProjectID      uint            `json:"project_id"`                        // 项目 ID
	TeamID         uint            `json:"team_id"`                           // 团队 ID
	Env            string          `json:"env"`                               // 环境
	Nodes          []TargetNodeReq `json:"nodes"`                             // 节点列表
}

// TargetNodeResp 是目标节点响应。
type TargetNodeResp struct {
	HostID uint   `json:"host_id"` // 主机 ID
	Name   string `json:"name"`    // 主机名称
	IP     string `json:"ip"`      // 主机 IP
	Status string `json:"status"`  // 主机状态
	Role   string `json:"role"`    // 角色
	Weight int    `json:"weight"`  // 权重
}

// TargetResp 是部署目标响应。
type TargetResp struct {
	ID              uint             `json:"id"`                         // 目标 ID
	Name            string           `json:"name"`                       // 目标名称
	TargetType      string           `json:"target_type"`                // 目标类型
	RuntimeType     string           `json:"runtime_type"`               // 运行时类型
	ClusterID       uint             `json:"cluster_id"`                 // 集群 ID
	ClusterSource   string           `json:"cluster_source"`             // 集群来源
	CredentialID    uint             `json:"credential_id"`              // 凭证 ID
	BootstrapJobID  string           `json:"bootstrap_job_id,omitempty"` // 引导任务 ID
	ProjectID       uint             `json:"project_id"`                 // 项目 ID
	TeamID          uint             `json:"team_id"`                    // 团队 ID
	Env             string           `json:"env"`                        // 环境
	Status          string           `json:"status"`                     // 状态
	ReadinessStatus string           `json:"readiness_status"`           // 就绪状态
	Nodes           []TargetNodeResp `json:"nodes,omitempty"`            // 节点列表
	CreatedAt       time.Time        `json:"created_at"`                 // 创建时间
	UpdatedAt       time.Time        `json:"updated_at"`                 // 更新时间
}

// ReleasePreviewReq 是发布预览请求参数。
type ReleasePreviewReq struct {
	ServiceID      uint              `json:"service_id" binding:"required"`  // 服务 ID
	TargetID       uint              `json:"target_id" binding:"required"`   // 目标 ID
	Env            string            `json:"env"`                            // 环境
	Strategy       string            `json:"strategy"`                       // 部署策略
	Variables      map[string]string `json:"variables"`                      // 模板变量
	TriggerSource  string            `json:"trigger_source,omitempty"`       // 触发来源: manual/ci
	TriggerContext map[string]any    `json:"trigger_context,omitempty"`      // 触发上下文
	CIRunID        uint              `json:"ci_run_id,omitempty"`            // CI 运行 ID
	ApprovalToken  string            `json:"approval_token"`                 // 审批令牌 (向后兼容)
	PreviewToken   string            `json:"preview_token"`                  // 预览令牌
}

// ReleasePreviewResp 是发布预览响应。
type ReleasePreviewResp struct {
	ResolvedManifest string              `json:"resolved_manifest"`        // 解析后的清单
	Checks           []map[string]string `json:"checks"`                   // 检查项
	Warnings         []map[string]string `json:"warnings"`                 // 警告项
	Runtime          string              `json:"runtime"`                  // 运行时类型
	PreviewToken     string              `json:"preview_token,omitempty"`  // 预览令牌
	PreviewExpiresAt *time.Time          `json:"preview_expires_at,omitempty"` // 预览过期时间
}

// ReleaseApplyResp 是发布执行响应。
type ReleaseApplyResp struct {
	ReleaseID        uint   `json:"release_id"`                  // 发布 ID
	UnifiedReleaseID uint   `json:"unified_release_id,omitempty"` // 统一发布 ID
	Status           string `json:"status"`                      // 状态
	RuntimeType      string `json:"runtime_type"`                // 运行时类型
	TriggerSource    string `json:"trigger_source,omitempty"`    // 触发来源
	TriggerContext   any    `json:"trigger_context,omitempty"`   // 触发上下文
	CIRunID          uint   `json:"ci_run_id,omitempty"`         // CI 运行 ID
	ApprovalRequired bool   `json:"approval_required,omitempty"` // 是否需要审批
	ApprovalTicket   string `json:"approval_ticket,omitempty"`   // 审批单号
	LifecycleState   string `json:"lifecycle_state,omitempty"`   // 生命周期状态
	ReasonCode       string `json:"reason_code,omitempty"`       // 原因码
}

// ReleaseSummaryResp 是发布摘要响应。
type ReleaseSummaryResp struct {
	ID                 uint       `json:"id"`                           // 发布 ID
	UnifiedReleaseID   uint       `json:"unified_release_id,omitempty"` // 统一发布 ID
	ServiceID          uint       `json:"service_id"`                   // 服务 ID
	TargetID           uint       `json:"target_id"`                    // 目标 ID
	NamespaceOrProject string     `json:"namespace_or_project"`         // 命名空间/项目
	RuntimeType        string     `json:"runtime_type"`                 // 运行时类型
	Strategy           string     `json:"strategy"`                     // 部署策略
	TriggerSource      string     `json:"trigger_source,omitempty"`     // 触发来源
	TriggerContextJSON string     `json:"trigger_context_json,omitempty"` // 触发上下文 JSON
	CIRunID            uint       `json:"ci_run_id,omitempty"`          // CI 运行 ID
	RevisionID         uint       `json:"revision_id"`                  // 配置版本 ID
	SourceReleaseID    uint       `json:"source_release_id"`            // 源发布 ID
	TargetRevision     string     `json:"target_revision"`              // 目标版本
	Status             string     `json:"status"`                       // 状态
	LifecycleState     string     `json:"lifecycle_state"`              // 生命周期状态
	DiagnosticsJSON    string     `json:"diagnostics_json"`             // 诊断信息 JSON
	VerificationJSON   string     `json:"verification_json"`            // 验证结果 JSON
	CreatedAt          time.Time  `json:"created_at"`                   // 创建时间
	UpdatedAt          time.Time  `json:"updated_at"`                   // 更新时间
	PreviewExpiresAt   *time.Time `json:"preview_expires_at,omitempty"` // 预览过期时间
}

// ReleaseDecisionReq 是发布审批决定请求参数。
type ReleaseDecisionReq struct {
	Comment string `json:"comment"` // 审批意见
}

// ReleaseTimelineEventResp 是发布时间线事件响应。
type ReleaseTimelineEventResp struct {
	ID            uint      `json:"id"`                       // 事件 ID
	ReleaseID     uint      `json:"release_id"`               // 发布 ID
	CorrelationID string    `json:"correlation_id,omitempty"` // 关联 ID
	TraceID       string    `json:"trace_id,omitempty"`       // 链路追踪 ID
	Action        string    `json:"action"`                   // 操作类型
	Actor         uint      `json:"actor"`                    // 操作人 ID
	Detail        any       `json:"detail"`                   // 详情
	CreatedAt     time.Time `json:"created_at"`               // 创建时间
}

// GovernanceReq 是服务治理策略请求参数。
type GovernanceReq struct {
	Env              string         `json:"env"`              // 环境
	TrafficPolicy    map[string]any `json:"traffic_policy"`   // 流量策略
	ResiliencePolicy map[string]any `json:"resilience_policy"` // 弹性策略
	AccessPolicy     map[string]any `json:"access_policy"`    // 访问策略
	SLOPolicy        map[string]any `json:"slo_policy"`       // SLO 策略
}

// ClusterBootstrapPreviewReq 是集群引导预览请求参数。
type ClusterBootstrapPreviewReq struct {
	Name           string `json:"name" binding:"required"`          // 集群名称
	ControlPlaneID uint   `json:"control_plane_host_id" binding:"required"` // 控制平面主机 ID
	WorkerIDs      []uint `json:"worker_host_ids"`                  // 工作节点 ID 列表
	CNI            string `json:"cni"`                              // CNI 插件
}

// ClusterBootstrapPreviewResp 是集群引导预览响应。
type ClusterBootstrapPreviewResp struct {
	Name             string   `json:"name"`                 // 集群名称
	ControlPlaneID   uint     `json:"control_plane_host_id"` // 控制平面主机 ID
	WorkerHostIDs    []uint   `json:"worker_host_ids"`      // 工作节点 ID 列表
	CNI              string   `json:"cni"`                  // CNI 插件
	Steps            []string `json:"steps"`                // 执行步骤
	ExpectedEndpoint string   `json:"expected_endpoint"`    // 预期端点
}

// ClusterBootstrapApplyResp 是集群引导执行响应。
type ClusterBootstrapApplyResp struct {
	TaskID    string `json:"task_id"`        // 任务 ID
	Status    string `json:"status"`         // 状态
	ClusterID uint   `json:"cluster_id,omitempty"` // 集群 ID
	TargetID  uint   `json:"target_id,omitempty"`  // 目标 ID
}

// EnvironmentBootstrapReq 是环境引导请求参数。
type EnvironmentBootstrapReq struct {
	Name           string `json:"name" binding:"required"`        // 环境名称
	RuntimeType    string `json:"runtime_type" binding:"required"` // 运行时类型: k8s/compose
	PackageVersion string `json:"package_version" binding:"required"` // 安装包版本
	Env            string `json:"env"`                            // 环境
	TargetID       uint   `json:"target_id"`                      // 目标 ID
	ClusterID      uint   `json:"cluster_id"`                     // 集群 ID
	ControlPlaneID uint   `json:"control_plane_host_id"`          // 控制平面主机 ID
	WorkerIDs      []uint `json:"worker_host_ids"`                // 工作节点 ID 列表
	NodeIDs        []uint `json:"node_ids"`                       // 节点 ID 列表
}

// EnvironmentBootstrapResp 是环境引导响应。
type EnvironmentBootstrapResp struct {
	JobID          string `json:"job_id"`          // 任务 ID
	Status         string `json:"status"`          // 状态
	RuntimeType    string `json:"runtime_type"`    // 运行时类型
	PackageVersion string `json:"package_version"` // 安装包版本
	TargetID       uint   `json:"target_id,omitempty"` // 目标 ID
}

// ClusterCredentialImportReq 是导入外部集群凭证请求参数。
type ClusterCredentialImportReq struct {
	Name        string `json:"name" binding:"required"` // 凭证名称
	RuntimeType string `json:"runtime_type"`            // 运行时类型
	Source      string `json:"source"`                  // 来源: external_managed
	AuthMethod  string `json:"auth_method"`             // 认证方式
	Endpoint    string `json:"endpoint"`                // 集群端点
	Kubeconfig  string `json:"kubeconfig"`              // kubeconfig 内容
	CACert      string `json:"ca_cert"`                 // CA 证书
	Cert        string `json:"cert"`                    // 客户端证书
	Key         string `json:"key"`                     // 客户端私钥
	Token       string `json:"token"`                   // Bearer Token
}

// PlatformCredentialRegisterReq 是注册平台托管凭证请求参数。
type PlatformCredentialRegisterReq struct {
	Name        string `json:"name"`               // 凭证名称
	RuntimeType string `json:"runtime_type"`       // 运行时类型
	ClusterID   uint   `json:"cluster_id" binding:"required"` // 集群 ID
}

// ClusterCredentialResp 是集群凭证响应。
type ClusterCredentialResp struct {
	ID              uint       `json:"id"`                       // 凭证 ID
	Name            string     `json:"name"`                     // 凭证名称
	RuntimeType     string     `json:"runtime_type"`             // 运行时类型
	Source          string     `json:"source"`                   // 来源
	ClusterID       uint       `json:"cluster_id"`               // 集群 ID
	Endpoint        string     `json:"endpoint"`                 // 集群端点
	AuthMethod      string     `json:"auth_method"`              // 认证方式
	Status          string     `json:"status"`                   // 状态
	LastTestAt      *time.Time `json:"last_test_at,omitempty"`   // 最后测试时间
	LastTestStatus  string     `json:"last_test_status,omitempty"` // 最后测试状态
	LastTestMessage string     `json:"last_test_message,omitempty"` // 最后测试消息
	CreatedAt       time.Time  `json:"created_at"`               // 创建时间
	UpdatedAt       time.Time  `json:"updated_at"`               // 更新时间
}

// ClusterCredentialTestResp 是凭证连通性测试响应。
type ClusterCredentialTestResp struct {
	CredentialID uint   `json:"credential_id"`    // 凭证 ID
	Connected    bool   `json:"connected"`        // 是否连通
	Message      string `json:"message"`          // 消息
	LatencyMS    int64  `json:"latency_ms,omitempty"` // 延迟 (毫秒)
}
