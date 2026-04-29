package model

import "time"

type HostPlugin struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PluginKey      string    `gorm:"column:plugin_key;type:varchar(64);not null;uniqueIndex" json:"plugin_key"`
	Name           string    `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Category       string    `gorm:"column:category;type:varchar(64);not null" json:"category"`
	Description    string    `gorm:"column:description;type:text;not null" json:"description"`
	DefaultVersion string    `gorm:"column:default_version;type:varchar(64);not null" json:"default_version"`
	Status         string    `gorm:"column:status;type:varchar(32);not null;default:active" json:"status"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (HostPlugin) TableName() string { return "host_plugins" }

type HostPluginVersion struct {
	ID               uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PluginID         uint64    `gorm:"column:plugin_id;not null;uniqueIndex:uk_host_plugin_version_arch,priority:1" json:"plugin_id"`
	Version          string    `gorm:"column:version;type:varchar(64);not null;uniqueIndex:uk_host_plugin_version_arch,priority:2" json:"version"`
	Arch             string    `gorm:"column:arch;type:varchar(32);not null;uniqueIndex:uk_host_plugin_version_arch,priority:3" json:"arch"`
	PackagePath      string    `gorm:"column:package_path;type:varchar(255);not null" json:"package_path"`
	InstallEntry     string    `gorm:"column:install_entry;type:varchar(128);not null" json:"install_entry"`
	UpgradeEntry     string    `gorm:"column:upgrade_entry;type:varchar(128);not null" json:"upgrade_entry"`
	UninstallEntry   string    `gorm:"column:uninstall_entry;type:varchar(128);not null" json:"uninstall_entry"`
	Checksum         string    `gorm:"column:checksum;type:varchar(128);not null" json:"checksum"`
	CapabilitiesJSON string    `gorm:"column:capabilities_json;type:json;not null" json:"capabilities_json"`
	ConfigSchemaJSON string    `gorm:"column:config_schema_json;type:json;not null" json:"config_schema_json"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (HostPluginVersion) TableName() string { return "host_plugin_versions" }

type HostPluginInstance struct {
	ID               uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	HostID           uint64     `gorm:"column:host_id;not null;uniqueIndex:uk_host_plugin_instance,priority:1" json:"host_id"`
	PluginID         uint64     `gorm:"column:plugin_id;not null;uniqueIndex:uk_host_plugin_instance,priority:2" json:"plugin_id"`
	DesiredVersion   string     `gorm:"column:desired_version;type:varchar(64);not null" json:"desired_version"`
	InstalledVersion string     `gorm:"column:installed_version;type:varchar(64);not null;default:''" json:"installed_version"`
	InstallStatus    string     `gorm:"column:install_status;type:varchar(32);not null;default:pending" json:"install_status"`
	RuntimeStatus    string     `gorm:"column:runtime_status;type:varchar(32);not null;default:pending_online" json:"runtime_status"`
	HealthStatus     string     `gorm:"column:health_status;type:varchar(32);not null;default:unknown" json:"health_status"`
	AgentID          string     `gorm:"column:agent_id;type:varchar(128);not null;default:''" json:"agent_id"`
	LastSeenAt       *time.Time `gorm:"column:last_seen_at" json:"last_seen_at"`
	CapabilitiesJSON string     `gorm:"column:capabilities_json;type:json;not null" json:"capabilities_json"`
	LastError        string     `gorm:"column:last_error;type:text;not null" json:"last_error"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (HostPluginInstance) TableName() string { return "host_plugin_instances" }

type HostPluginConfigRevision struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	InstanceID     uint64    `gorm:"column:instance_id;not null;index" json:"instance_id"`
	Version        string    `gorm:"column:version;type:varchar(64);not null" json:"version"`
	ConfigYAML     string    `gorm:"column:config_yaml;type:text;not null" json:"config_yaml"`
	Checksum       string    `gorm:"column:checksum;type:varchar(128);not null" json:"checksum"`
	DeliveryStatus string    `gorm:"column:delivery_status;type:varchar(32);not null;default:pending" json:"delivery_status"`
	CreatedBy      uint64    `gorm:"column:created_by;not null;default:0" json:"created_by"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (HostPluginConfigRevision) TableName() string { return "host_plugin_config_revisions" }

type HostPluginTask struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	InstanceID   uint64     `gorm:"column:instance_id;not null;index" json:"instance_id"`
	Operation    string     `gorm:"column:operation;type:varchar(32);not null" json:"operation"`
	Status       string     `gorm:"column:status;type:varchar(32);not null;index" json:"status"`
	RequestedBy  uint64     `gorm:"column:requested_by;not null;default:0" json:"requested_by"`
	StartedAt    *time.Time `gorm:"column:started_at" json:"started_at"`
	FinishedAt   *time.Time `gorm:"column:finished_at" json:"finished_at"`
	ErrorMessage string     `gorm:"column:error_message;type:text;not null" json:"error_message"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (HostPluginTask) TableName() string { return "host_plugin_tasks" }

type HostPluginTaskLog struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TaskID    uint64    `gorm:"column:task_id;not null;index" json:"task_id"`
	Stream    string    `gorm:"column:stream;type:varchar(16);not null" json:"stream"`
	Content   string    `gorm:"column:content;type:text;not null" json:"content"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (HostPluginTaskLog) TableName() string { return "host_plugin_task_logs" }
