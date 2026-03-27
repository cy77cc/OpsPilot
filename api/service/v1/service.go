package v1

import "time"

// ---------- shared primitive types ----------

type LabelKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type EnvKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PortConfig struct {
	Name          string `json:"name"`
	Protocol      string `json:"protocol"`
	ContainerPort int32  `json:"container_port"`
	ServicePort   int32  `json:"service_port"`
}

type HealthCheckConfig struct {
	Type             string `json:"type"` // http/tcp/cmd
	Path             string `json:"path"`
	Port             int32  `json:"port"`
	Command          string `json:"command"`
	InitialDelaySec  int32  `json:"initial_delay_sec"`
	PeriodSec        int32  `json:"period_sec"`
	FailureThreshold int32  `json:"failure_threshold"`
}

type VolumeConfig struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	HostPath  string `json:"host_path"`
}

type StandardServiceConfig struct {
	Image       string             `json:"image"`
	Replicas    int32              `json:"replicas"`
	Ports       []PortConfig       `json:"ports"`
	Envs        []EnvKV            `json:"envs"`
	Resources   map[string]string  `json:"resources"` // cpu, memory
	Volumes     []VolumeConfig     `json:"volumes"`
	HealthCheck *HealthCheckConfig `json:"health_check"`
}

type TemplateVar struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
	SourcePath  string `json:"source_path,omitempty"`
}

type RenderDiagnostic struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ---------- request types ----------

type RenderPreviewReq struct {
	Mode           string                 `json:"mode"`   // standard/custom
	Target         string                 `json:"target"` // k8s/compose
	StandardConfig *StandardServiceConfig `json:"standard_config"`
	CustomYAML     string                 `json:"custom_yaml"`
	ServiceName    string                 `json:"service_name"`
	ServiceType    string                 `json:"service_type"` // stateless/stateful
	Variables      map[string]string      `json:"variables"`
	ValidateOnly   bool                   `json:"validate_only"`
}

type TransformReq struct {
	StandardConfig *StandardServiceConfig `json:"standard_config"`
	Target         string                 `json:"target"`
	ServiceName    string                 `json:"service_name"`
	ServiceType    string                 `json:"service_type"`
}

type ServiceCreateReq struct {
	ProjectID       uint                   `json:"project_id"`
	TeamID          uint                   `json:"team_id"`
	Name            string                 `json:"name"`
	Env             string                 `json:"env"`
	Owner           string                 `json:"owner"`
	ServiceKind     string                 `json:"service_kind"`
	Visibility      string                 `json:"visibility"`
	GrantedTeams    []uint                 `json:"granted_teams"`
	Icon            string                 `json:"icon"`
	Tags            []string               `json:"tags"`
	ServiceType     string                 `json:"service_type"`
	RuntimeType     string                 `json:"runtime_type"`
	ConfigMode      string                 `json:"config_mode"`
	RenderTarget    string                 `json:"render_target"`
	Labels          []LabelKV              `json:"labels"`
	StandardConfig  *StandardServiceConfig `json:"standard_config"`
	CustomYAML      string                 `json:"custom_yaml"`
	SourceTemplateV string                 `json:"source_template_version"`
	Status          string                 `json:"status"`

	// legacy compat
	Image         string            `json:"image"`
	Replicas      int32             `json:"replicas"`
	ServicePort   int32             `json:"service_port"`
	ContainerPort int32             `json:"container_port"`
	NodePort      int32             `json:"node_port"`
	EnvVars       []EnvKV           `json:"env_vars"`
	Resources     map[string]string `json:"resources"`
	YamlContent   string            `json:"yaml_content"`
}

type HelmImportReq struct {
	ServiceID    uint   `json:"service_id"`
	ChartName    string `json:"chart_name"`
	ChartVersion string `json:"chart_version"`
	ChartRef     string `json:"chart_ref"`
	ValuesYAML   string `json:"values_yaml"`
	RenderedYAML string `json:"rendered_yaml"`
}

type HelmRenderReq struct {
	ReleaseID    uint   `json:"release_id"`
	ChartRef     string `json:"chart_ref"`
	ChartName    string `json:"chart_name"`
	ValuesYAML   string `json:"values_yaml"`
	RenderedYAML string `json:"rendered_yaml"`
}

type DeployReq struct {
	ClusterID     uint              `json:"cluster_id"`
	Namespace     string            `json:"namespace"`
	Env           string            `json:"env"`
	VariablesRef  string            `json:"variables_ref"`
	Variables     map[string]string `json:"variables"`
	DeployTarget  string            `json:"deploy_target"` // k8s/compose/helm
	ApprovalToken string            `json:"approval_token"`
}

type VisibilityUpdateReq struct {
	Visibility string `json:"visibility"`
}

type GrantTeamsReq struct {
	GrantedTeams []uint `json:"granted_teams"`
}

type VariableExtractReq struct {
	StandardConfig *StandardServiceConfig `json:"standard_config"`
	CustomYAML     string                 `json:"custom_yaml"`
	RenderTarget   string                 `json:"render_target"`
	ServiceName    string                 `json:"service_name"`
	ServiceType    string                 `json:"service_type"`
}

type VariableValuesUpsertReq struct {
	Env        string            `json:"env"`
	Values     map[string]string `json:"values"`
	SecretKeys []string          `json:"secret_keys"`
}

type RevisionCreateReq struct {
	ServiceID      uint                   `json:"service_id"`
	ConfigMode     string                 `json:"config_mode"`
	RenderTarget   string                 `json:"render_target"`
	StandardConfig *StandardServiceConfig `json:"standard_config"`
	CustomYAML     string                 `json:"custom_yaml"`
	VariableSchema []TemplateVar          `json:"variable_schema"`
}

type DeployTargetUpsertReq struct {
	ClusterID    uint           `json:"cluster_id"`
	Namespace    string         `json:"namespace"`
	DeployTarget string         `json:"deploy_target"`
	Policy       map[string]any `json:"policy"`
}

// ---------- response types ----------

type RenderPreviewResp struct {
	RenderedYAML     string             `json:"rendered_yaml"`
	ResolvedYAML     string             `json:"resolved_yaml,omitempty"`
	Diagnostics      []RenderDiagnostic `json:"diagnostics"`
	UnresolvedVars   []string           `json:"unresolved_vars,omitempty"`
	DetectedVars     []TemplateVar      `json:"detected_vars,omitempty"`
	ASTSummary       map[string]any     `json:"ast_summary,omitempty"`
	NormalizedConfig any                `json:"normalized_config,omitempty"`
}

type TransformResp struct {
	CustomYAML   string        `json:"custom_yaml"`
	SourceHash   string        `json:"source_hash"`
	DetectedVars []TemplateVar `json:"detected_vars,omitempty"`
}

type ServiceListItem struct {
	ID                    uint                   `json:"id"`
	ProjectID             uint                   `json:"project_id"`
	TeamID                uint                   `json:"team_id"`
	Name                  string                 `json:"name"`
	Env                   string                 `json:"env"`
	Owner                 string                 `json:"owner"`
	RuntimeType           string                 `json:"runtime_type"`
	ConfigMode            string                 `json:"config_mode"`
	ServiceKind           string                 `json:"service_kind"`
	Visibility            string                 `json:"visibility"`
	GrantedTeams          []uint                 `json:"granted_teams,omitempty"`
	Icon                  string                 `json:"icon,omitempty"`
	Tags                  []string               `json:"tags,omitempty"`
	DeployCount           int                    `json:"deploy_count"`
	Status                string                 `json:"status"`
	Labels                []LabelKV              `json:"labels"`
	StandardConfig        *StandardServiceConfig `json:"standard_config,omitempty"`
	CustomYAML            string                 `json:"custom_yaml,omitempty"`
	RenderedYAML          string                 `json:"rendered_yaml,omitempty"`
	LastRevisionID        uint                   `json:"last_revision_id"`
	DefaultTargetID       uint                   `json:"default_target_id"`
	TemplateEngineVersion string                 `json:"template_engine_version,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

type VariableExtractResp struct {
	Vars []TemplateVar `json:"vars"`
}

type VariableValuesResp struct {
	ServiceID  uint              `json:"service_id"`
	Env        string            `json:"env"`
	Values     map[string]string `json:"values"`
	SecretKeys []string          `json:"secret_keys,omitempty"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type ServiceRevisionItem struct {
	ID             uint          `json:"id"`
	ServiceID      uint          `json:"service_id"`
	RevisionNo     uint          `json:"revision_no"`
	ConfigMode     string        `json:"config_mode"`
	RenderTarget   string        `json:"render_target"`
	VariableSchema []TemplateVar `json:"variable_schema,omitempty"`
	CreatedBy      uint          `json:"created_by"`
	CreatedAt      time.Time     `json:"created_at"`
}

type DeployTargetResp struct {
	ID           uint           `json:"id"`
	ServiceID    uint           `json:"service_id"`
	ClusterID    uint           `json:"cluster_id"`
	Namespace    string         `json:"namespace"`
	DeployTarget string         `json:"deploy_target"`
	Policy       map[string]any `json:"policy,omitempty"`
	IsDefault    bool           `json:"is_default"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type DeployPreviewResp struct {
	ResolvedYAML string             `json:"resolved_yaml"`
	Checks       []RenderDiagnostic `json:"checks"`
	Warnings     []RenderDiagnostic `json:"warnings"`
	Target       DeployTargetResp   `json:"target"`
}

type DeployResp struct {
	ReleaseRecordID uint `json:"release_record_id"`
}

type ReleaseRecordItem struct {
	ID           uint      `json:"id"`
	ServiceID    uint      `json:"service_id"`
	RevisionID   uint      `json:"revision_id"`
	ClusterID    uint      `json:"cluster_id"`
	Namespace    string    `json:"namespace"`
	Env          string    `json:"env"`
	DeployTarget string    `json:"deploy_target"`
	Status       string    `json:"status"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
