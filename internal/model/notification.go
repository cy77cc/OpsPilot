// Package model 提供数据库模型定义。
//
// 本文件定义通知相关的数据模型，包括通知主体和用户通知关联。
package model

import "time"

// Notification 是通知主体表模型，存储系统通知内容。
//
// 表名: notifications
// 关联: UserNotification (一对多，非外键)
//
// 类型说明:
//   - Type: alert (告警) / task (任务) / system (系统) / approval (审批)
//   - Severity: critical / warning / info
//   - ActionType: confirm (确认) / approve (审批) / view (查看)
type Notification struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`                                      // 通知 ID
	Type       string    `gorm:"column:type;type:varchar(32);not null;index" json:"type"`             // 类型: alert/task/system/approval
	Title      string    `gorm:"column:title;type:varchar(255);not null" json:"title"`               // 标题
	Content    string    `gorm:"column:content;type:text" json:"content"`                            // 内容
	Severity   string    `gorm:"column:severity;type:varchar(16);default:'info';index" json:"severity"` // 严重程度: critical/warning/info
	Source     string    `gorm:"column:source;type:varchar(128);index" json:"source"`                // 来源模块
	SourceID   string    `gorm:"column:source_id;type:varchar(128);index" json:"source_id"`          // 来源 ID
	ActionURL  string    `gorm:"column:action_url;type:varchar(512)" json:"action_url"`              // 操作链接
	ActionType string    `gorm:"column:action_type;type:varchar(32)" json:"action_type"`             // 操作类型: confirm/approve/view
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`           // 创建时间
}

// TableName 返回通知主体表名。
func (Notification) TableName() string { return "notifications" }

// UserNotification 是用户通知关联表模型，记录用户的通知状态。
//
// 表名: user_notifications
// 关联:
//   - User (多对一，通过 user_id)
//   - Notification (多对一，通过 notification_id)
//
// 状态字段:
//   - ReadAt: 阅读时间 (已读)
//   - DismissedAt: 忽略时间 (已忽略)
//   - ConfirmedAt: 确认时间 (已确认操作)
type UserNotification struct {
	ID             uint         `gorm:"primaryKey;column:id" json:"id"`                                         // 关联 ID
	UserID         uint64       `gorm:"column:user_id;not null;index:idx_user_notification" json:"user_id"`     // 用户 ID
	NotificationID uint         `gorm:"column:notification_id;not null;index:idx_user_notification" json:"notification_id"` // 通知 ID
	ReadAt         *time.Time   `gorm:"column:read_at" json:"read_at"`                                          // 阅读时间
	DismissedAt    *time.Time   `gorm:"column:dismissed_at" json:"dismissed_at"`                                // 忽略时间
	ConfirmedAt    *time.Time   `gorm:"column:confirmed_at" json:"confirmed_at"`                                // 确认时间
	Notification   Notification `gorm:"foreignKey:NotificationID" json:"notification"`                          // 关联通知
	CreatedAt      time.Time    `gorm:"column:created_at;autoCreateTime" json:"created_at"`                     // 创建时间
}

// TableName 返回用户通知关联表名。
func (UserNotification) TableName() string { return "user_notifications" }
