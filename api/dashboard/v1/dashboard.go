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

// OverviewResponse is the response of dashboard overview API.
type OverviewResponse struct {
	Hosts    HealthStats   `json:"hosts"`
	Clusters HealthStats   `json:"clusters"`
	Services HealthStats   `json:"services"`
	Alerts   AlertSummary  `json:"alerts"`
	Events   []EventItem   `json:"events"`
	Metrics  MetricsSeries `json:"metrics"`
}
