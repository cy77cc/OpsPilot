package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Policy 策略
type Policy struct {
	ID        uint                   `gorm:"primaryKey" json:"id"`
	Name      string                 `gorm:"type:varchar(255);not null" json:"name"`
	Type      string                 `gorm:"type:varchar(32);not null;index" json:"type"` // traffic, resilience, access, slo
	TargetID  uint                   `gorm:"index" json:"target_id"`
	Config    map[string]interface{} `gorm:"type:json" json:"config"`
	Enabled   bool                   `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

func (Policy) TableName() string {
	return "policies"
}

// Scan implements sql.Scanner for Config
func (p *Policy) Scan(value interface{}) error {
	if value == nil {
		p.Config = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &p.Config)
}

// Value implements driver.Valuer for Config
func (p Policy) Value() (driver.Value, error) {
	if p.Config == nil {
		return nil, nil
	}
	return json.Marshal(p.Config)
}

// Policy types
const (
	PolicyTypeTraffic    = "traffic"
	PolicyTypeResilience = "resilience"
	PolicyTypeAccess     = "access"
	PolicyTypeSLO        = "slo"
)
