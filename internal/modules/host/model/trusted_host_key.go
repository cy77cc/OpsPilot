package model

import "time"

type TrustedHostKeyStatus string

const (
	TrustedHostKeyStatusTrusted TrustedHostKeyStatus = "trusted"
	TrustedHostKeyStatusRotated TrustedHostKeyStatus = "rotated"
	TrustedHostKeyStatusRevoked TrustedHostKeyStatus = "revoked"
)

type TrustedHostKey struct {
	ID                uint64               `gorm:"column:id;primaryKey;autoIncrement"`
	HostID            uint64               `gorm:"column:host_id;index;not null"`
	Host              string               `gorm:"column:host;type:varchar(255);not null;index"`
	Port              int                  `gorm:"column:port;not null;index"`
	Algorithm         string               `gorm:"column:algorithm;type:varchar(64);not null"`
	FingerprintSHA256 string               `gorm:"column:fingerprint_sha256;type:varchar(128);not null;index"`
	PublicKey         string               `gorm:"column:public_key;type:text;not null"`
	Status            TrustedHostKeyStatus `gorm:"column:status;type:varchar(32);not null;index"`
	CreatedBy         uint64               `gorm:"column:created_by;not null;index"`
	ConfirmedAt       time.Time            `gorm:"column:confirmed_at;not null"`
	LastSeenAt        time.Time            `gorm:"column:last_seen_at;not null"`
	CreatedAt         time.Time            `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time            `gorm:"column:updated_at;autoUpdateTime"`
}

func (TrustedHostKey) TableName() string { return "host_trusted_keys" }
