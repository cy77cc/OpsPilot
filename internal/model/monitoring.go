package model

import "time"

type AlertRule struct {
	ID             uint      `gorm:"primaryKey;column:id" json:"id"`
	Name           string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Metric         string    `gorm:"column:metric;type:varchar(64);not null;index" json:"metric"`
	Operator       string    `gorm:"column:operator;type:varchar(8);default:'gt'" json:"operator"`
	Threshold      float64   `gorm:"column:threshold;type:decimal(12,4);default:0" json:"threshold"`
	DurationSec    int       `gorm:"column:duration_sec;default:300" json:"duration_sec"`
	WindowSec      int       `gorm:"column:window_sec;default:3600" json:"window_sec"`
	GranularitySec int       `gorm:"column:granularity_sec;default:60" json:"granularity_sec"`
	DimensionsJSON string    `gorm:"column:dimensions_json;type:longtext" json:"dimensions_json"`
	Severity       string    `gorm:"column:severity;type:varchar(16);default:'warning'" json:"severity"`
	Source         string    `gorm:"column:source;type:varchar(32);default:'system'" json:"source"`
	Scope          string    `gorm:"column:scope;type:varchar(32);default:'global'" json:"scope"`
	State          string    `gorm:"column:state;type:varchar(16);default:'enabled'" json:"state"`
	Enabled        bool      `gorm:"column:enabled;default:true" json:"enabled"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AlertRule) TableName() string { return "alert_rules" }

type AlertEvent struct {
	ID          uint       `gorm:"primaryKey;column:id" json:"id"`
	RuleID      uint       `gorm:"column:rule_id;index" json:"rule_id"`
	Title       string     `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Message     string     `gorm:"column:message;type:text" json:"message"`
	Metric      string     `gorm:"column:metric;type:varchar(64);index" json:"metric"`
	Value       float64    `gorm:"column:value;type:decimal(14,4);default:0" json:"value"`
	Threshold   float64    `gorm:"column:threshold;type:decimal(14,4);default:0" json:"threshold"`
	Severity    string     `gorm:"column:severity;type:varchar(16);default:'warning'" json:"severity"`
	Source      string     `gorm:"column:source;type:varchar(128);index" json:"source"`
	Status      string     `gorm:"column:status;type:varchar(16);default:'firing';index" json:"status"`
	TriggeredAt time.Time  `gorm:"column:triggered_at;index" json:"triggered_at"`
	ResolvedAt  *time.Time `gorm:"column:resolved_at" json:"resolved_at,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AlertEvent) TableName() string { return "alerts" }

type MetricPoint struct {
	ID             uint      `gorm:"primaryKey;column:id" json:"id"`
	Metric         string    `gorm:"column:metric;type:varchar(64);index:idx_metric_time,priority:1" json:"metric"`
	Source         string    `gorm:"column:source;type:varchar(128);index" json:"source"`
	DimensionsJSON string    `gorm:"column:dimensions_json;type:longtext" json:"dimensions_json"`
	Value          float64   `gorm:"column:value;type:decimal(14,4);default:0" json:"value"`
	Collected      time.Time `gorm:"column:collected_at;index:idx_metric_time,priority:2" json:"collected_at"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (MetricPoint) TableName() string { return "metric_points" }

type AlertRuleEvaluation struct {
	ID          uint      `gorm:"primaryKey;column:id" json:"id"`
	RuleID      uint      `gorm:"column:rule_id;index" json:"rule_id"`
	Metric      string    `gorm:"column:metric;type:varchar(64);index" json:"metric"`
	Operator    string    `gorm:"column:operator;type:varchar(8);default:'gt'" json:"operator"`
	Value       float64   `gorm:"column:value;type:decimal(14,4);default:0" json:"value"`
	Threshold   float64   `gorm:"column:threshold;type:decimal(14,4);default:0" json:"threshold"`
	Triggered   bool      `gorm:"column:triggered;default:false;index" json:"triggered"`
	PrevState   string    `gorm:"column:prev_state;type:varchar(16);default:'normal'" json:"prev_state"`
	NextState   string    `gorm:"column:next_state;type:varchar(16);default:'normal'" json:"next_state"`
	EvaluatedAt time.Time `gorm:"column:evaluated_at;index" json:"evaluated_at"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
}

func (AlertRuleEvaluation) TableName() string { return "alert_rule_evaluations" }

type AlertNotificationChannel struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`
	Name       string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Type       string    `gorm:"column:type;type:varchar(32);not null;index" json:"type"` // log|webhook
	Provider   string    `gorm:"column:provider;type:varchar(64);default:''" json:"provider"`
	Target     string    `gorm:"column:target;type:varchar(512);default:''" json:"target"`
	ConfigJSON string    `gorm:"column:config_json;type:longtext" json:"config_json"`
	Enabled    bool      `gorm:"column:enabled;default:true;index" json:"enabled"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AlertNotificationChannel) TableName() string { return "alert_notification_channels" }

type AlertNotificationDelivery struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`
	AlertID      uint      `gorm:"column:alert_id;index" json:"alert_id"`
	RuleID       uint      `gorm:"column:rule_id;index" json:"rule_id"`
	ChannelID    uint      `gorm:"column:channel_id;index" json:"channel_id"`
	ChannelType  string    `gorm:"column:channel_type;type:varchar(32);index" json:"channel_type"`
	Target       string    `gorm:"column:target;type:varchar(512);default:''" json:"target"`
	Status       string    `gorm:"column:status;type:varchar(16);default:'sent';index" json:"status"`
	ErrorMessage string    `gorm:"column:error_message;type:text" json:"error_message"`
	DeliveredAt  time.Time `gorm:"column:delivered_at;index" json:"delivered_at"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
}

func (AlertNotificationDelivery) TableName() string { return "alert_notification_deliveries" }

type ClusterBootstrapTask struct {
	ID             string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	Name           string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	ClusterID      *uint     `gorm:"column:cluster_id;index" json:"cluster_id"`
	ControlPlaneID uint      `gorm:"column:control_plane_host_id;index" json:"control_plane_host_id"`
	WorkerIDsJSON  string    `gorm:"column:worker_ids_json;type:longtext" json:"worker_ids_json"`
	K8sVersion     string    `gorm:"column:k8s_version;type:varchar(32)" json:"k8s_version"`
	VersionChannel string    `gorm:"column:version_channel;type:varchar(32)" json:"version_channel"`
	RepoMode       string    `gorm:"column:repo_mode;type:varchar(16);default:'online'" json:"repo_mode"`
	RepoURL        string    `gorm:"column:repo_url;type:varchar(512)" json:"repo_url"`
	ImageRepository string   `gorm:"column:image_repository;type:varchar(256)" json:"image_repository"`
	EndpointMode   string    `gorm:"column:endpoint_mode;type:varchar(16);default:'nodeIP'" json:"endpoint_mode"`
	ControlPlaneEndpoint string `gorm:"column:control_plane_endpoint;type:varchar(256)" json:"control_plane_endpoint"`
	VIPProvider    string    `gorm:"column:vip_provider;type:varchar(32)" json:"vip_provider"`
	EtcdMode       string    `gorm:"column:etcd_mode;type:varchar(16);default:'stacked'" json:"etcd_mode"`
	ExternalEtcdJSON string  `gorm:"column:external_etcd_json;type:longtext" json:"external_etcd_json"`
	CNI            string    `gorm:"column:cni;type:varchar(32);default:'flannel'" json:"cni"`
	PodCIDR        string    `gorm:"column:pod_cidr;type:varchar(32)" json:"pod_cidr"`
	ServiceCIDR    string    `gorm:"column:service_cidr;type:varchar(32)" json:"service_cidr"`
	StepsJSON      string    `gorm:"column:steps_json;type:longtext" json:"steps_json"`
	ResolvedConfigJSON string `gorm:"column:resolved_config_json;type:longtext" json:"resolved_config_json"`
	DiagnosticsJSON string    `gorm:"column:diagnostics_json;type:longtext" json:"diagnostics_json"`
	Status         string    `gorm:"column:status;type:varchar(32);index" json:"status"`
	ResultJSON     string    `gorm:"column:result_json;type:longtext" json:"result_json"`
	ErrorMessage   string    `gorm:"column:error_message;type:text" json:"error_message"`
	CreatedBy      uint64    `gorm:"column:created_by;index" json:"created_by"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ClusterBootstrapTask) TableName() string { return "cluster_bootstrap_tasks" }
