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
