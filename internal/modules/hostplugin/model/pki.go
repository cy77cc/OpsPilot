package model

import "time"

// OpsAgentCA stores the platform's OpsAgent root CA (singleton).
type OpsAgentCA struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CACertPEM string    `gorm:"column:ca_cert_pem;type:text;not null" json:"ca_cert_pem"`
	CAKeyPEM  string    `gorm:"column:ca_key_pem;type:text;not null" json:"ca_key_pem"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (OpsAgentCA) TableName() string { return "opsagent_ca" }

// OpsAgentHostCert stores per-host client certificates issued by the CA.
type OpsAgentHostCert struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	HostID       uint64    `gorm:"column:host_id;not null;index" json:"host_id"`
	InstanceID   uint64    `gorm:"column:instance_id;not null;index" json:"instance_id"`
	SerialNumber string    `gorm:"column:serial_number;type:varchar(64);not null" json:"serial_number"`
	CertPEM      string    `gorm:"column:cert_pem;type:text;not null" json:"cert_pem"`
	KeyPEM       string    `gorm:"column:key_pem;type:text;not null" json:"key_pem"`
	NotAfter     time.Time `gorm:"column:not_after;not null" json:"not_after"`
	Revoked      bool      `gorm:"column:revoked;not null;default:false" json:"revoked"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (OpsAgentHostCert) TableName() string { return "opsagent_host_certificates" }

// HostPluginPackage stores uploaded agent packages.
type HostPluginPackage struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PluginKey   string    `gorm:"column:plugin_key;type:varchar(64);not null;uniqueIndex:uk_host_plugin_package,priority:1" json:"plugin_key"`
	Version     string    `gorm:"column:version;type:varchar(64);not null;uniqueIndex:uk_host_plugin_package,priority:2" json:"version"`
	Arch        string    `gorm:"column:arch;type:varchar(32);not null;uniqueIndex:uk_host_plugin_package,priority:3" json:"arch"`
	Filename    string    `gorm:"column:filename;type:varchar(255);not null" json:"filename"`
	StoragePath string    `gorm:"column:storage_path;type:varchar(512);not null" json:"storage_path"`
	Checksum    string    `gorm:"column:checksum;type:varchar(128);not null" json:"checksum"`
	SizeBytes   int64     `gorm:"column:size_bytes;not null;default:0" json:"size_bytes"`
	UploadedBy  uint64    `gorm:"column:uploaded_by;not null;default:0" json:"uploaded_by"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (HostPluginPackage) TableName() string { return "host_plugin_packages" }
