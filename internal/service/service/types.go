// Package service 提供服务目录管理的类型定义。
//
// 本文件定义服务管理模块的请求、响应和数据结构类型。
package service

import "time"

// LabelKV 表示标签键值对。
type LabelKV struct {
	Key   string `json:"key"`   // 标签键
	Value string `json:"value"` // 标签值
}

// EnvKV 表示环境变量键值对。
type EnvKV struct {
	Key   string `json:"key"`   // 环境变量名
	Value string `json:"value"` // 环境变量值
}

// PortConfig 表示端口配置。
type PortConfig struct {
	Name          string `json:"name"`           // 端口名称
	Protocol      string `json:"protocol"`       // 协议类型 (TCP/UDP)
	ContainerPort int32  `json:"container_port"` // 容器端口
	ServicePort   int32  `json:"service_port"`   // 服务端口
}

// HealthCheckConfig 表示健康检查配置。
type HealthCheckConfig struct {
	Type             string `json:"type"`              // 检查类型 (http/tcp/cmd)
	Path             string `json:"path"`              // HTTP 检查路径
	Port             int32  `json:"port"`              // 检查端口
	Command          string `json:"command"`           // 命令检查命令
	InitialDelaySec  int32  `json:"initial_delay_sec"` // 初始延迟秒数
	PeriodSec        int32  `json:"period_sec"`        // 检查周期秒数
	FailureThreshold int32  `json:"failure_threshold"` // 失败阈值
}

// VolumeConfig 表示存储卷配置。
type VolumeConfig struct {
	Name      string `json:"name"`       // 卷名称
	MountPath string `json:"mount_path"` // 挂载路径
	HostPath  string `json:"host_path"`  // 主机路径
}

// StandardServiceConfig 表示标准服务配置。
//
// 用于通过标准化的方式定义服务运行参数，可自动转换为 K8s 或 Docker Compose YAML。
type StandardServiceConfig struct {
	Image       string             `json:"image"`       // 镜像地址
	Replicas    int32              `json:"replicas"`    // 副本数
	Ports       []PortConfig       `json:"ports"`       // 端口配置列表
	Envs        []EnvKV            `json:"envs"`        // 环境变量列表
	Resources   map[string]string  `json:"resources"`   // 资源限制 (cpu,memory)
	Volumes     []VolumeConfig     `json:"volumes"`     // 存储卷配置列表
	HealthCheck *HealthCheckConfig `json:"health_check"` // 健康检查配置
}

// RenderPreviewReq 表示渲染预览请求。
type RenderPreviewReq struct {
	Mode           string                 `json:"mode"`           // 渲染模式 (standard/custom)
	Target         string                 `json:"target"`         // 目标平台 (k8s/compose)
	StandardConfig *StandardServiceConfig `json:"standard_config"` // 标准配置
	CustomYAML     string                 `json:"custom_yaml"`     // 自定义 YAML
	ServiceName    string                 `json:"service_name"`    // 服务名称
	ServiceType    string                 `json:"service_type"`    // 服务类型 (stateless/stateful)
	Variables      map[string]string      `json:"variables"`       // 变量值映射
	ValidateOnly   bool                   `json:"validate_only"`   // 仅校验不渲染
}

// RenderDiagnostic 表示渲染诊断信息。
type RenderDiagnostic struct {
	Level   string `json:"level"`   // 诊断级别 (error/warning/info)
	Code    string `json:"code"`    // 诊断代码
	Message string `json:"message"` // 诊断消息
}

// RenderPreviewResp 表示渲染预览响应。
type RenderPreviewResp struct {
	RenderedYAML     string             `json:"rendered_yaml"`             // 渲染后的 YAML
	ResolvedYAML     string             `json:"resolved_yaml,omitempty"`   // 变量替换后的 YAML
	Diagnostics      []RenderDiagnostic `json:"diagnostics"`               // 诊断信息列表
	UnresolvedVars   []string           `json:"unresolved_vars,omitempty"` // 未解析的变量列表
	DetectedVars     []TemplateVar      `json:"detected_vars,omitempty"`   // 检测到的变量列表
	ASTSummary       map[string]any     `json:"ast_summary,omitempty"`     // AST 摘要
	NormalizedConfig any                `json:"normalized_config,omitempty"` // 标准化配置
}

// TransformReq 表示标准配置转自定义 YAML 请求。
type TransformReq struct {
	StandardConfig *StandardServiceConfig `json:"standard_config"` // 标准配置
	Target         string                 `json:"target"`         // 目标平台 (k8s/compose)
	ServiceName    string                 `json:"service_name"`   // 服务名称
	ServiceType    string                 `json:"service_type"`   // 服务类型 (stateless/stateful)
}

// TransformResp 表示标准配置转自定义 YAML 响应。
type TransformResp struct {
	CustomYAML   string        `json:"custom_yaml"`            // 转换后的自定义 YAML
	SourceHash   string        `json:"source_hash"`            // 源配置哈希
	DetectedVars []TemplateVar `json:"detected_vars,omitempty"` // 检测到的变量列表
}

// ServiceCreateReq 表示服务创建请求。
type ServiceCreateReq struct {
	ProjectID       uint                   `json:"project_id"`            // 项目 ID
	TeamID          uint                   `json:"team_id"`               // 团队 ID
	Name            string                 `json:"name"`                  // 服务名称
	Env             string                 `json:"env"`                   // 环境 (development/staging/production)
	Owner           string                 `json:"owner"`                 // 负责人
	ServiceKind     string                 `json:"service_kind"`          // 服务种类 (business/middleware)
	Visibility      string                 `json:"visibility"`            // 可见性 (private/team/team-granted/public)
	GrantedTeams    []uint                 `json:"granted_teams"`         // 授权团队 ID 列表
	Icon            string                 `json:"icon"`                  // 图标
	Tags            []string               `json:"tags"`                  // 标签列表
	ServiceType     string                 `json:"service_type"`          // 服务类型 (stateless/stateful)
	RuntimeType     string                 `json:"runtime_type"`          // 运行时类型 (k8s/compose)
	ConfigMode      string                 `json:"config_mode"`           // 配置模式 (standard/custom)
	RenderTarget    string                 `json:"render_target"`         // 渲染目标 (k8s/compose)
	Labels          []LabelKV              `json:"labels"`                // 标签列表
	StandardConfig  *StandardServiceConfig `json:"standard_config"`       // 标准配置
	CustomYAML      string                 `json:"custom_yaml"`           // 自定义 YAML
	SourceTemplateV string                 `json:"source_template_version"` // 源模板版本
	Status          string                 `json:"status"`                // 状态 (draft/active)

	// legacy compat - 兼容旧版本字段
	Image         string            `json:"image"`          // 镜像地址
	Replicas      int32             `json:"replicas"`       // 副本数
	ServicePort   int32             `json:"service_port"`   // 服务端口
	ContainerPort int32             `json:"container_port"` // 容器端口
	NodePort      int32             `json:"node_port"`      // NodePort
	EnvVars       []EnvKV           `json:"env_vars"`       // 环境变量列表
	Resources     map[string]string `json:"resources"`      // 资源限制
	YamlContent   string            `json:"yaml_content"`   // YAML 内容
}

// ServiceListItem 表示服务列表项。
type ServiceListItem struct {
	ID                    uint                   `json:"id"`                       // 服务 ID
	ProjectID             uint                   `json:"project_id"`               // 项目 ID
	TeamID                uint                   `json:"team_id"`                  // 团队 ID
	Name                  string                 `json:"name"`                     // 服务名称
	Env                   string                 `json:"env"`                      // 环境
	Owner                 string                 `json:"owner"`                    // 负责人
	RuntimeType           string                 `json:"runtime_type"`             // 运行时类型
	ConfigMode            string                 `json:"config_mode"`              // 配置模式
	ServiceKind           string                 `json:"service_kind"`             // 服务种类
	Visibility            string                 `json:"visibility"`               // 可见性
	GrantedTeams          []uint                 `json:"granted_teams,omitempty"`  // 授权团队 ID 列表
	Icon                  string                 `json:"icon,omitempty"`           // 图标
	Tags                  []string               `json:"tags,omitempty"`           // 标签列表
	DeployCount           int                    `json:"deploy_count"`             // 部署次数
	Status                string                 `json:"status"`                   // 状态
	Labels                []LabelKV              `json:"labels"`                   // 标签列表
	StandardConfig        *StandardServiceConfig `json:"standard_config,omitempty"` // 标准配置
	CustomYAML            string                 `json:"custom_yaml,omitempty"`    // 自定义 YAML
	RenderedYAML          string                 `json:"rendered_yaml,omitempty"`  // 渲染后的 YAML
	LastRevisionID        uint                   `json:"last_revision_id"`         // 最新版本 ID
	DefaultTargetID       uint                   `json:"default_target_id"`        // 默认部署目标 ID
	TemplateEngineVersion string                 `json:"template_engine_version,omitempty"` // 模板引擎版本
	CreatedAt             time.Time              `json:"created_at"`               // 创建时间
	UpdatedAt             time.Time              `json:"updated_at"`               // 更新时间
}

// HelmImportReq 表示 Helm Chart 导入请求。
type HelmImportReq struct {
	ServiceID    uint   `json:"service_id"`    // 服务 ID
	ChartName    string `json:"chart_name"`    // Chart 名称
	ChartVersion string `json:"chart_version"` // Chart 版本
	ChartRef     string `json:"chart_ref"`     // Chart 引用路径
	ValuesYAML   string `json:"values_yaml"`   // Values YAML
	RenderedYAML string `json:"rendered_yaml"` // 渲染后的 YAML
}

// HelmRenderReq 表示 Helm 渲染请求。
type HelmRenderReq struct {
	ReleaseID    uint   `json:"release_id"`    // 发布 ID
	ChartRef     string `json:"chart_ref"`     // Chart 引用路径
	ChartName    string `json:"chart_name"`    // Chart 名称
	ValuesYAML   string `json:"values_yaml"`   // Values YAML
	RenderedYAML string `json:"rendered_yaml"` // 渲染后的 YAML
}

// DeployReq 表示部署请求。
type DeployReq struct {
	ClusterID     uint              `json:"cluster_id"`     // 集群 ID
	Namespace     string            `json:"namespace"`      // 命名空间
	Env           string            `json:"env"`            // 环境
	VariablesRef  string            `json:"variables_ref"`  // 变量引用
	Variables     map[string]string `json:"variables"`      // 变量值映射
	DeployTarget  string            `json:"deploy_target"`  // 部署目标 (k8s/compose/helm)
	ApprovalToken string            `json:"approval_token"` // 审批令牌
}

// VisibilityUpdateReq 表示可见性更新请求。
type VisibilityUpdateReq struct {
	Visibility string `json:"visibility"` // 可见性 (private/team/team-granted/public)
}

// GrantTeamsReq 表示授权团队更新请求。
type GrantTeamsReq struct {
	GrantedTeams []uint `json:"granted_teams"` // 授权团队 ID 列表
}

// TemplateVar 表示模板变量定义。
type TemplateVar struct {
	Name        string `json:"name"`                  // 变量名
	Required    bool   `json:"required"`              // 是否必填
	Default     string `json:"default,omitempty"`     // 默认值
	Description string `json:"description,omitempty"` // 描述
	SourcePath  string `json:"source_path,omitempty"` // 来源路径
}

// VariableExtractReq 表示变量提取请求。
type VariableExtractReq struct {
	StandardConfig *StandardServiceConfig `json:"standard_config"` // 标准配置
	CustomYAML     string                 `json:"custom_yaml"`     // 自定义 YAML
	RenderTarget   string                 `json:"render_target"`   // 渲染目标
	ServiceName    string                 `json:"service_name"`    // 服务名称
	ServiceType    string                 `json:"service_type"`    // 服务类型
}

// VariableExtractResp 表示变量提取响应。
type VariableExtractResp struct {
	Vars []TemplateVar `json:"vars"` // 变量列表
}

// VariableValuesUpsertReq 表示变量值更新请求。
type VariableValuesUpsertReq struct {
	Env        string            `json:"env"`        // 环境名称
	Values     map[string]string `json:"values"`     // 变量值映射
	SecretKeys []string          `json:"secret_keys"` // 敏感变量键列表
}

// VariableValuesResp 表示变量值响应。
type VariableValuesResp struct {
	ServiceID  uint              `json:"service_id"`            // 服务 ID
	Env        string            `json:"env"`                   // 环境名称
	Values     map[string]string `json:"values"`                // 变量值映射
	SecretKeys []string          `json:"secret_keys,omitempty"` // 敏感变量键列表
	UpdatedAt  time.Time         `json:"updated_at"`            // 更新时间
}

// ServiceRevisionItem 表示服务版本项。
type ServiceRevisionItem struct {
	ID             uint          `json:"id"`                       // 版本 ID
	ServiceID      uint          `json:"service_id"`               // 服务 ID
	RevisionNo     uint          `json:"revision_no"`              // 版本号
	ConfigMode     string        `json:"config_mode"`              // 配置模式
	RenderTarget   string        `json:"render_target"`            // 渲染目标
	VariableSchema []TemplateVar `json:"variable_schema,omitempty"` // 变量 Schema
	CreatedBy      uint          `json:"created_by"`               // 创建者 ID
	CreatedAt      time.Time     `json:"created_at"`               // 创建时间
}

// RevisionCreateReq 表示版本创建请求。
type RevisionCreateReq struct {
	ServiceID      uint                   `json:"service_id"`      // 服务 ID
	ConfigMode     string                 `json:"config_mode"`     // 配置模式
	RenderTarget   string                 `json:"render_target"`   // 渲染目标
	StandardConfig *StandardServiceConfig `json:"standard_config"` // 标准配置
	CustomYAML     string                 `json:"custom_yaml"`     // 自定义 YAML
	VariableSchema []TemplateVar          `json:"variable_schema"` // 变量 Schema
}

// DeployTargetUpsertReq 表示部署目标更新请求。
type DeployTargetUpsertReq struct {
	ClusterID    uint           `json:"cluster_id"`    // 集群 ID
	Namespace    string         `json:"namespace"`     // 命名空间
	DeployTarget string         `json:"deploy_target"` // 部署目标 (k8s/compose)
	Policy       map[string]any `json:"policy"`        // 部署策略
}

// DeployTargetResp 表示部署目标响应。
type DeployTargetResp struct {
	ID           uint           `json:"id"`                // 目标 ID
	ServiceID    uint           `json:"service_id"`        // 服务 ID
	ClusterID    uint           `json:"cluster_id"`        // 集群 ID
	Namespace    string         `json:"namespace"`         // 命名空间
	DeployTarget string         `json:"deploy_target"`     // 部署目标
	Policy       map[string]any `json:"policy,omitempty"`  // 部署策略
	IsDefault    bool           `json:"is_default"`        // 是否默认
	UpdatedAt    time.Time      `json:"updated_at"`        // 更新时间
}

// DeployPreviewResp 表示部署预览响应。
type DeployPreviewResp struct {
	ResolvedYAML     string             `json:"resolved_yaml"`             // 变量替换后的 YAML
	Checks           []RenderDiagnostic `json:"checks"`                    // 检查项列表
	Warnings         []RenderDiagnostic `json:"warnings"`                  // 警告列表
	Target           DeployTargetResp   `json:"target"`                    // 部署目标
	TargetID         uint               `json:"target_id,omitempty"`       // 目标 ID
	PreviewToken     string             `json:"preview_token,omitempty"`   // 预览令牌
	PreviewExpiresAt *time.Time         `json:"preview_expires_at,omitempty"` // 预览过期时间
}

// DeployResp 表示部署响应。
type DeployResp struct {
	ReleaseRecordID  uint   `json:"release_record_id"`       // 发布记录 ID
	UnifiedReleaseID uint   `json:"unified_release_id,omitempty"` // 统一发布 ID
	TriggerSource    string `json:"trigger_source,omitempty"`    // 触发来源
}

// ReleaseRecordItem 表示发布记录项。
type ReleaseRecordItem struct {
	ID               uint      `json:"id"`                          // 记录 ID
	UnifiedReleaseID uint      `json:"unified_release_id,omitempty"` // 统一发布 ID
	ServiceID        uint      `json:"service_id"`                  // 服务 ID
	RevisionID       uint      `json:"revision_id"`                 // 版本 ID
	ClusterID        uint      `json:"cluster_id"`                  // 集群 ID
	Namespace        string    `json:"namespace"`                   // 命名空间
	Env              string    `json:"env"`                         // 环境
	DeployTarget     string    `json:"deploy_target"`               // 部署目标
	Status           string    `json:"status"`                      // 状态
	TriggerSource    string    `json:"trigger_source,omitempty"`    // 触发来源
	CIRunID          uint      `json:"ci_run_id,omitempty"`         // CI 运行 ID
	Error            string    `json:"error,omitempty"`             // 错误信息
	CreatedAt        time.Time `json:"created_at"`                  // 创建时间
}
