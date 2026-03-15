package v1

import "time"

// OverviewRequest holds query params for dashboard overview API.
type OverviewRequest struct {
	TimeRange string `form:"time_range" json:"time_range"`
}

// HealthStats contains health summary numbers.
type HealthStats struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Degraded  int `json:"degraded,omitempty"`
	Unhealthy int `json:"unhealthy,omitempty"`
	Offline   int `json:"offline,omitempty"`
}

// AlertItem is a compact alert payload for dashboard panel.
type AlertItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Severity  string    `json:"severity"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"createdAt"`
}

// EventItem is a compact event payload for dashboard stream.
type EventItem struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

// MetricPoint is a point in metric time-series.
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricSeries represents a single host's metric series.
type MetricSeries struct {
	HostID   uint64        `json:"hostId"`
	HostName string        `json:"hostName"`
	Data     []MetricPoint `json:"data"`
}

// AlertSummary includes firing count and recent alert list.
type AlertSummary struct {
	Firing int         `json:"firing"`
	Recent []AlertItem `json:"recent"`
}

// MetricsSeries includes cpu/memory trend lines per host.
type MetricsSeries struct {
	CPUUsage    []MetricSeries `json:"cpu_usage"`
	MemoryUsage []MetricSeries `json:"memory_usage"`
}

// AIStatsSummary AI 助手统计摘要。
type AIStatsSummary struct {
	SessionCount   int64   `json:"sessionCount"`   // 会话总数
	TokenCount     int64   `json:"tokenCount"`     // Token 消耗总数
	AvgDurationMs  int64   `json:"avgDurationMs"`  // 平均响应时间（毫秒）
	SuccessRate    float64 `json:"successRate"`    // 成功率 (0-100)
	PreviousChange string  `json:"previousChange"` // 与上一周期对比变化趋势
}

// AISessionItem 最近 AI 会话条目。
type AISessionItem struct {
	ID        string    `json:"id"`
	Scene     string    `json:"scene"`     // 场景 (host/cluster/service/k8s)
	Title     string    `json:"title"`     // 会话标题
	Status    string    `json:"status"`    // 状态 (success/error)
	CreatedAt time.Time `json:"createdAt"` // 创建时间
}

// AIActivity AI 活动面板数据。
type AIActivity struct {
	Stats    AIStatsSummary  `json:"stats"`    // 统计摘要
	Sessions []AISessionItem `json:"sessions"` // 最近会话
	ByScene  map[string]int  `json:"byScene"`  // 按场景分组统计
}

// OverviewResponse is the response of dashboard overview API.
type OverviewResponse struct {
	Hosts    HealthStats   `json:"hosts"`
	Clusters HealthStats   `json:"clusters"`
	Services HealthStats   `json:"services"`
	Alerts   AlertSummary  `json:"alerts"`
	Events   []EventItem   `json:"events"`
	Metrics  MetricsSeries `json:"metrics"`
	AI       AIActivity    `json:"ai"` // AI 助手活动数据
}

// WorkloadHealth 单类工作负载健康统计。
type WorkloadHealth struct {
	Total   int `json:"total"`
	Healthy int `json:"healthy"`
}

// WorkloadStats K8s 工作负载健康统计。
type WorkloadStats struct {
	Deployments  WorkloadHealth `json:"deployments"`
	StatefulSets WorkloadHealth `json:"statefulsets"`
	DaemonSets   WorkloadHealth `json:"daemonsets"`
	Services     int            `json:"services"`
	Ingresses    int            `json:"ingresses"`
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

// ClusterResource 集群资源概览。
type ClusterResource struct {
	ClusterID   uint           `json:"clusterId"`
	ClusterName string         `json:"clusterName"`
	CPU         ResourceMetric `json:"cpu"`
	Memory      ResourceMetric `json:"memory"`
	Pods        PodStats       `json:"pods"`
}

// DeploymentStats 部署状态统计。
type DeploymentStats struct {
	Running         int `json:"running"`
	PendingApproval int `json:"pendingApproval"`
	TodayTotal      int `json:"todayTotal"`
	TodaySuccess    int `json:"todaySuccess"`
	TodayFailed     int `json:"todayFailed"`
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
