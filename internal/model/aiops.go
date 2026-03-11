// Package model 提供数据库模型定义。
//
// 本文件定义 AIOps 相关的数据模型，包括风险发现、异常检测和优化建议。
package model

import "time"

// RiskFinding 是风险发现表模型，存储系统自动识别的潜在风险。
//
// 表名: risk_findings
// 关联: Service (通过 service_id，非外键)
//
// 风险类型示例:
//   - resource_exhaustion: 资源耗尽风险
//   - configuration_drift: 配置漂移
//   - security_vulnerability: 安全漏洞
//   - performance_degradation: 性能退化
type RiskFinding struct {
	ID          uint      `gorm:"primaryKey" json:"id"`                                // 风险 ID
	Type        string    `gorm:"type:varchar(64);not null;index" json:"type"`         // 风险类型
	Severity    string    `gorm:"type:varchar(16);not null;index" json:"severity"`     // 严重程度: critical/high/medium/low
	Title       string    `gorm:"type:varchar(255);not null" json:"title"`             // 风险标题
	Description string    `gorm:"type:text" json:"description"`                        // 风险描述
	ServiceID   uint      `gorm:"index" json:"service_id"`                             // 关联服务 ID
	ServiceName string    `gorm:"type:varchar(255)" json:"service_name"`               // 服务名称 (冗余字段)
	Metadata    string    `gorm:"type:text" json:"metadata"`                           // 扩展元数据 (JSON 格式)
	CreatedAt   time.Time `json:"created_at"`                                          // 发现时间
	ResolvedAt  *time.Time `json:"resolved_at"`                                        // 解决时间
}

// TableName 返回风险发现表名。
func (RiskFinding) TableName() string {
	return "risk_findings"
}

// Anomaly 是异常检测表模型，存储系统监控指标异常记录。
//
// 表名: anomalies
// 关联: Service (通过 service_id，非外键)
//
// 异常类型示例:
//   - cpu_spike: CPU 飙升
//   - memory_leak: 内存泄漏
//   - latency_increase: 延迟增加
//   - error_rate_spike: 错误率飙升
type Anomaly struct {
	ID          uint       `gorm:"primaryKey" json:"id"`                            // 异常 ID
	Type        string     `gorm:"type:varchar(64);not null;index" json:"type"`     // 异常类型
	Metric      string     `gorm:"type:varchar(64);not null" json:"metric"`         // 指标名称
	Value       float64    `json:"value"`                                           // 实际值
	Threshold   float64    `json:"threshold"`                                       // 阈值
	ServiceID   uint       `gorm:"index" json:"service_id"`                         // 关联服务 ID
	ServiceName string     `gorm:"type:varchar(255)" json:"service_name"`           // 服务名称 (冗余字段)
	DetectedAt  time.Time  `json:"detected_at"`                                     // 检测时间
	ResolvedAt  *time.Time `json:"resolved_at"`                                     // 解决时间
}

// TableName 返回异常检测表名。
func (Anomaly) TableName() string {
	return "anomalies"
}

// Suggestion 是优化建议表模型，存储系统生成的优化建议。
//
// 表名: suggestions
// 关联: Service (通过 service_id，非外键)
//
// 建议类型示例:
//   - resource_scaling: 资源扩缩容
//   - cost_optimization: 成本优化
//   - performance_tuning: 性能调优
//   - security_hardening: 安全加固
type Suggestion struct {
	ID          uint       `gorm:"primaryKey" json:"id"`                        // 建议 ID
	Type        string     `gorm:"type:varchar(64);not null;index" json:"type"` // 建议类型
	Title       string     `gorm:"type:varchar(255);not null" json:"title"`     // 建议标题
	Description string     `gorm:"type:text" json:"description"`                // 建议描述
	Impact      string     `gorm:"type:varchar(16);not null" json:"impact"`     // 影响程度: high/medium/low
	ServiceID   uint       `gorm:"index" json:"service_id"`                     // 关联服务 ID
	ServiceName string     `gorm:"type:varchar(255)" json:"service_name"`       // 服务名称 (冗余字段)
	CreatedAt   time.Time  `json:"created_at"`                                  // 创建时间
	AppliedAt   *time.Time `json:"applied_at"`                                  // 应用时间
}

// TableName 返回优化建议表名。
func (Suggestion) TableName() string {
	return "suggestions"
}
