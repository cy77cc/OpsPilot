package v1

import "time"

// CreateRuleRequest is the request body for creating an alert rule.
type CreateRuleRequest struct {
	Name           string  `json:"name" binding:"required"`
	Metric         string  `json:"metric" binding:"required"`
	Operator       string  `json:"operator"`
	Threshold      float64 `json:"threshold"`
	Severity       string  `json:"severity"`
	Enabled        *bool   `json:"enabled"`
	WindowSec      int     `json:"window_sec"`
	GranularitySec int     `json:"granularity_sec"`
	DimensionsJSON string  `json:"dimensions_json"`
}

// UpdateRuleRequest is the request body for updating an alert rule.
type UpdateRuleRequest struct {
	Name           string  `json:"name"`
	Operator       string  `json:"operator"`
	Threshold      float64 `json:"threshold"`
	Severity       string  `json:"severity"`
	Enabled        *bool   `json:"enabled"`
	WindowSec      int     `json:"window_sec"`
	GranularitySec int     `json:"granularity_sec"`
	DimensionsJSON *string `json:"dimensions_json"`
}

// CreateChannelRequest is the request body for creating a notification channel.
type CreateChannelRequest struct {
	Name       string `json:"name" binding:"required"`
	Type       string `json:"type"`
	Provider   string `json:"provider"`
	Target     string `json:"target"`
	ConfigJSON string `json:"config_json"`
	ProjectID  *uint  `json:"project_id"`
	Enabled    *bool  `json:"enabled"`
}

// UpdateChannelRequest is the request body for updating a notification channel.
type UpdateChannelRequest struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Provider   string  `json:"provider"`
	Target     string  `json:"target"`
	ConfigJSON *string `json:"config_json"`
	ProjectID  *uint   `json:"project_id"`
	Enabled    *bool   `json:"enabled"`
}

// TestChannelRequest is the request body for testing a notification channel.
type TestChannelRequest struct {
	Provider   string `json:"provider" binding:"required"`
	Target     string `json:"target"`
	ConfigJSON string `json:"config_json"`
}

// SeverityRouteUpsertRequest is the request body for creating or updating a severity route.
type SeverityRouteUpsertRequest struct {
	ProjectID  *uint  `json:"project_id"`
	Scope      string `json:"scope"`
	Severity   string `json:"severity" binding:"required"`
	ChannelIDs []uint `json:"channel_ids"`
	Enabled    *bool  `json:"enabled"`
}

// RuleChannelBindingCreateRequest is the request body for creating a rule-channel binding.
type RuleChannelBindingCreateRequest struct {
	ProjectID *uint `json:"project_id"`
	ChannelID uint  `json:"channel_id" binding:"required"`
	Priority  int   `json:"priority"`
	Enabled   *bool `json:"enabled"`
}

// RuleChannelBindingUpdateRequest is the request body for updating a rule-channel binding.
type RuleChannelBindingUpdateRequest struct {
	ProjectID *uint `json:"project_id"`
	Priority  int   `json:"priority"`
	Enabled   *bool `json:"enabled"`
}

// MetricQuery holds parameters for querying metric time-series data.
type MetricQuery struct {
	Metric         string
	Start          time.Time
	End            time.Time
	GranularitySec int
	Source         string
}

// MetricQueryResult is the response returned by the GetMetrics endpoint.
type MetricQueryResult struct {
	Window struct {
		Start          time.Time `json:"start"`
		End            time.Time `json:"end"`
		GranularitySec int       `json:"granularity_sec"`
	} `json:"window"`
	Dimensions map[string]any   `json:"dimensions"`
	Series     []map[string]any `json:"series"`
}

// NotificationPayload carries alert information sent to notification channels.
type NotificationPayload struct {
	AlertID   uint
	RuleID    uint
	Title     string
	Message   string
	Severity  string
	Metric    string
	Value     float64
	Threshold float64
}

// DeliveryResult represents the outcome of a notification delivery attempt.
type DeliveryResult struct {
	Status string
	Error  string
}
