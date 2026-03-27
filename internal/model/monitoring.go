// Package model 提供监控告警相关的数据模型定义。
//
// 本文件包含以下模型:
//   - AlertRule: 告警规则
//   - AlertEvent: 告警事件
//   - AlertNotificationChannel: 告警通知渠道
//   - AlertNotificationDelivery: 告警通知投递记录
//   - AlertSilence: 告警静默配置
//   - ClusterBootstrapTask: 集群引导任务
package model

import "time"

// AlertRule 是告警规则表模型。
//
// 存储告警规则的配置信息，包括指标、阈值、持续时间等参数。
// 支持自定义 PromQL 表达式和标签配置。
//
// 表名: alert_rules
type AlertRule struct {
	ID              uint      `gorm:"primaryKey;column:id" json:"id"`                                          // 规则 ID
	Name            string    `gorm:"column:name;type:varchar(128);not null" json:"name"`                     // 规则名称
	Metric          string    `gorm:"column:metric;type:varchar(64);not null;index" json:"metric"`            // 监控指标名称
	PromQLExpr      string    `gorm:"column:promql_expr;type:varchar(512);default:''" json:"promql_expr"`     // 自定义 PromQL 表达式
	Operator        string    `gorm:"column:operator;type:varchar(8);default:'gt'" json:"operator"`           // 比较运算符 (gt/gte/lt/lte/eq)
	Threshold       float64   `gorm:"column:threshold;type:decimal(12,4);default:0" json:"threshold"`         // 告警阈值
	DurationSec     int       `gorm:"column:duration_sec;default:300" json:"duration_sec"`                    // 持续时间 (秒)
	WindowSec       int       `gorm:"column:window_sec;default:3600" json:"window_sec"`                       // 时间窗口 (秒)
	GranularitySec  int       `gorm:"column:granularity_sec;default:60" json:"granularity_sec"`               // 采集粒度 (秒)
	DimensionsJSON  string    `gorm:"column:dimensions_json;type:longtext" json:"dimensions_json"`            // 维度标签 JSON
	LabelsJSON      string    `gorm:"column:labels_json;type:longtext" json:"labels_json"`                    // 自定义标签 JSON
	AnnotationsJSON string    `gorm:"column:annotations_json;type:longtext" json:"annotations_json"`          // 注解 JSON
	Severity        string    `gorm:"column:severity;type:varchar(16);default:'warning'" json:"severity"`     // 严重级别 (critical/warning/info)
	Source          string    `gorm:"column:source;type:varchar(32);default:'system'" json:"source"`          // 规则来源 (system/custom)
	Scope           string    `gorm:"column:scope;type:varchar(32);default:'global'" json:"scope"`            // 作用域 (global/cluster/host)
	State           string    `gorm:"column:state;type:varchar(16);default:'enabled'" json:"state"`           // 规则状态 (enabled/disabled)
	Enabled         bool      `gorm:"column:enabled;default:true" json:"enabled"`                             // 是否启用
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                     // 创建时间
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                     // 更新时间
}

// TableName 返回 AlertRule 的数据库表名。
func (AlertRule) TableName() string { return "alert_rules" }

// AlertEvent 是告警事件表模型。
//
// 记录告警触发和恢复的事件信息，包括告警标题、消息、状态等。
// 每次告警触发都会创建新记录，状态变更会更新现有记录。
//
// 表名: alerts
type AlertEvent struct {
	ID          uint       `gorm:"primaryKey;column:id" json:"id"`                                      // 事件 ID
	RuleID      uint       `gorm:"column:rule_id;index" json:"rule_id"`                                 // 关联规则 ID
	Title       string     `gorm:"column:title;type:varchar(255);not null" json:"title"`                // 告警标题
	Message     string     `gorm:"column:message;type:text" json:"message"`                             // 告警消息
	Metric      string     `gorm:"column:metric;type:varchar(64);index" json:"metric"`                  // 监控指标名称
	Value       float64    `gorm:"column:value;type:decimal(14,4);default:0" json:"value"`              // 当前值
	Threshold   float64    `gorm:"column:threshold;type:decimal(14,4);default:0" json:"threshold"`      // 阈值
	Severity    string     `gorm:"column:severity;type:varchar(16);default:'warning'" json:"severity"`  // 严重级别 (critical/warning/info)
	Source      string     `gorm:"column:source;type:varchar(128);index" json:"source"`                 // 告警来源标识
	Status      string     `gorm:"column:status;type:varchar(16);default:'firing';index" json:"status"` // 状态 (firing/resolved)
	TriggeredAt time.Time  `gorm:"column:triggered_at;index" json:"triggered_at"`                       // 触发时间
	ResolvedAt  *time.Time `gorm:"column:resolved_at" json:"resolved_at,omitempty"`                     // 恢复时间
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`            // 创建时间
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                  // 更新时间
}

// TableName 返回 AlertEvent 的数据库表名。
func (AlertEvent) TableName() string { return "alerts" }

// AlertNotificationChannel 是告警通知渠道表模型。
//
// 配置告警通知的发送渠道，支持多种类型如日志、Webhook 等。
// 每个渠道可以配置独立的目标地址和参数。
//
// 表名: alert_notification_channels
type AlertNotificationChannel struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`                             // 渠道 ID
	Name       string    `gorm:"column:name;type:varchar(128);not null" json:"name"`         // 渠道名称
	Type       string    `gorm:"column:type;type:varchar(32);not null;index" json:"type"`    // 渠道类型 (log/webhook)
	Provider   string    `gorm:"column:provider;type:varchar(64);default:''" json:"provider"` // 通知提供者
	Target     string    `gorm:"column:target;type:varchar(512);default:''" json:"target"`   // 目标地址 (URL/邮箱/手机号等)
	ConfigJSON string    `gorm:"column:config_json;type:longtext" json:"config_json"`        // 配置参数 JSON
	Enabled    bool      `gorm:"column:enabled;default:true;index" json:"enabled"`           // 是否启用
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`         // 创建时间
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`         // 更新时间
}

// TableName 返回 AlertNotificationChannel 的数据库表名。
func (AlertNotificationChannel) TableName() string { return "alert_notification_channels" }

// AlertNotificationDelivery 是告警通知投递记录表模型。
//
// 记录每次告警通知的发送结果，包括成功/失败状态和错误信息。
// 用于追踪告警通知的投递情况和故障排查。
//
// 表名: alert_notification_deliveries
type AlertNotificationDelivery struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`                                 // 投递记录 ID
	AlertID      uint      `gorm:"column:alert_id;index" json:"alert_id"`                          // 关联告警 ID
	RuleID       uint      `gorm:"column:rule_id;index" json:"rule_id"`                            // 关联规则 ID
	ChannelID    uint      `gorm:"column:channel_id;index" json:"channel_id"`                      // 关联渠道 ID
	ChannelType  string    `gorm:"column:channel_type;type:varchar(32);index" json:"channel_type"` // 渠道类型
	Target       string    `gorm:"column:target;type:varchar(512);default:''" json:"target"`       // 目标地址
	Status       string    `gorm:"column:status;type:varchar(16);default:'sent';index" json:"status"` // 投递状态 (sent/failed)
	ErrorMessage string    `gorm:"column:error_message;type:text" json:"error_message"`            // 错误信息
	DeliveredAt  time.Time `gorm:"column:delivered_at;index" json:"delivered_at"`                  // 投递时间
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`       // 创建时间
}

// TableName 返回 AlertNotificationDelivery 的数据库表名。
func (AlertNotificationDelivery) TableName() string { return "alert_notification_deliveries" }

// AlertSilence 是告警静默配置表模型。
//
// 配置告警静默规则，在指定时间范围内抑制特定告警。
// 支持基于标签匹配器的灵活静默规则配置。
//
// 表名: alert_silences
type AlertSilence struct {
	ID           uint64    `gorm:"primaryKey;column:id" json:"id"`                                                // 静默配置 ID
	SilenceID    string    `gorm:"column:silence_id;type:varchar(64);not null;index" json:"silence_id"`           // 静默规则标识
	MatchersJSON string    `gorm:"column:matchers_json;type:longtext;not null" json:"matchers_json"`              // 匹配器配置 JSON
	StartsAt     time.Time `gorm:"column:starts_at;index:idx_alert_silences_time,priority:1" json:"starts_at"`    // 静默开始时间
	EndsAt       time.Time `gorm:"column:ends_at;index:idx_alert_silences_time,priority:2" json:"ends_at"`        // 静默结束时间
	CreatedBy    uint64    `gorm:"column:created_by;not null" json:"created_by"`                                  // 创建者 ID
	Comment      string    `gorm:"column:comment;type:varchar(512);default:''" json:"comment"`                    // 备注说明
	Status       string    `gorm:"column:status;type:varchar(16);default:'active';index" json:"status"`           // 状态 (active/expired)
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                            // 创建时间
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                            // 更新时间
}

// TableName 返回 AlertSilence 的数据库表名。
func (AlertSilence) TableName() string { return "alert_silences" }

// ClusterBootstrapTask 是集群引导任务表模型。
//
// 记录 Kubernetes 集群的初始化引导任务配置和执行状态。
// 包含集群版本、网络配置、CNI 插件等完整部署参数。
//
// 表名: cluster_bootstrap_tasks
type ClusterBootstrapTask struct {
	ID                   string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`                            // 任务 ID
	Name                 string    `gorm:"column:name;type:varchar(128);not null" json:"name"`                         // 任务名称
	ClusterID            *uint     `gorm:"column:cluster_id;index" json:"cluster_id"`                                  // 关联集群 ID
	ControlPlaneID       uint      `gorm:"column:control_plane_host_id;index" json:"control_plane_host_id"`            // 控制平面主机 ID
	WorkerIDsJSON        string    `gorm:"column:worker_ids_json;type:longtext" json:"worker_ids_json"`                // Worker 节点 ID 列表 JSON
	K8sVersion           string    `gorm:"column:k8s_version;type:varchar(32)" json:"k8s_version"`                     // Kubernetes 版本
	VersionChannel       string    `gorm:"column:version_channel;type:varchar(32)" json:"version_channel"`             // 版本通道 (stable/edge)
	RepoMode             string    `gorm:"column:repo_mode;type:varchar(16);default:'online'" json:"repo_mode"`        // 仓库模式 (online/offline)
	RepoURL              string    `gorm:"column:repo_url;type:varchar(512)" json:"repo_url"`                          // 自定义仓库地址
	ImageRepository      string    `gorm:"column:image_repository;type:varchar(256)" json:"image_repository"`         // 镜像仓库地址
	EndpointMode         string    `gorm:"column:endpoint_mode;type:varchar(16);default:'nodeIP'" json:"endpoint_mode"` // 端点模式 (nodeIP/vip)
	ControlPlaneEndpoint string    `gorm:"column:control_plane_endpoint;type:varchar(256)" json:"control_plane_endpoint"` // 控制平面端点
	VIPProvider          string    `gorm:"column:vip_provider;type:varchar(32)" json:"vip_provider"`                   // VIP 提供者 (keepalived/haproxy)
	EtcdMode             string    `gorm:"column:etcd_mode;type:varchar(16);default:'stacked'" json:"etcd_mode"`       // etcd 模式 (stacked/external)
	ExternalEtcdJSON     string    `gorm:"column:external_etcd_json;type:longtext" json:"external_etcd_json"`          // 外部 etcd 配置 JSON
	CNI                  string    `gorm:"column:cni;type:varchar(32);default:'flannel'" json:"cni"`                   // CNI 插件 (flannel/calico/cilium)
	PodCIDR              string    `gorm:"column:pod_cidr;type:varchar(32)" json:"pod_cidr"`                           // Pod CIDR
	ServiceCIDR          string    `gorm:"column:service_cidr;type:varchar(32)" json:"service_cidr"`                   // Service CIDR
	StepsJSON            string    `gorm:"column:steps_json;type:longtext" json:"steps_json"`                          // 执行步骤 JSON
	ResolvedConfigJSON   string    `gorm:"column:resolved_config_json;type:longtext" json:"resolved_config_json"`      // 解析后的配置 JSON
	DiagnosticsJSON      string    `gorm:"column:diagnostics_json;type:longtext" json:"diagnostics_json"`              // 诊断信息 JSON
	Status               string    `gorm:"column:status;type:varchar(32);index" json:"status"`                         // 任务状态 (pending/running/success/failed)
	ResultJSON           string    `gorm:"column:result_json;type:longtext" json:"result_json"`                        // 执行结果 JSON
	ErrorMessage         string    `gorm:"column:error_message;type:text" json:"error_message"`                        // 错误信息
	CreatedBy            uint64    `gorm:"column:created_by;index" json:"created_by"`                                  // 创建者 ID
	CreatedAt            time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`                   // 创建时间
	UpdatedAt            time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                         // 更新时间
}

// TableName 返回 ClusterBootstrapTask 的数据库表名。
func (ClusterBootstrapTask) TableName() string { return "cluster_bootstrap_tasks" }
