// Package notification 提供通知管理服务。
//
// integration.go 实现通知集成服务，负责创建各类通知并推送。
// 支持的通知类型:
//   - alert: 告警通知
//   - task: 任务通知
//   - system: 系统通知
package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/websocket"
	"gorm.io/gorm"
)

// NotificationIntegrator 是通知集成服务。
//
// 负责创建各类通知并通过 WebSocket 实时推送给用户。
type NotificationIntegrator struct {
	db  *gorm.DB
	hub *websocket.Hub
}

// NewNotificationIntegrator 创建通知集成服务实例。
//
// 参数:
//   - db: 数据库连接
//
// 返回: 通知集成服务实例
func NewNotificationIntegrator(db *gorm.DB) *NotificationIntegrator {
	return &NotificationIntegrator{
		db:  db,
		hub: websocket.GetHub(),
	}
}

// CreateAlertNotification 创建告警通知。
//
// 根据告警事件创建通知并推送给相关用户。
// 通知接收用户为具有 admin 或 super_admin 角色的用户。
//
// 参数:
//   - ctx: 上下文
//   - alert: 告警事件模型
//
// 返回: 成功返回 nil，失败返回错误
func (n *NotificationIntegrator) CreateAlertNotification(ctx context.Context, alert *model.AlertEvent) error {
	// 创建通知主体
	notif := model.Notification{
		Type:       "alert",
		Title:      alert.Title,
		Content:    alert.Message,
		Severity:   alert.Severity,
		Source:     alert.Source,
		SourceID:   fmt.Sprintf("%d", alert.ID),
		ActionURL:  fmt.Sprintf("/monitor?alert_id=%d", alert.ID),
		ActionType: "confirm",
	}

	// 获取所有应该通知的用户
	userIDs, err := n.getAlertNotificationUsers(ctx, alert)
	if err != nil {
		return err
	}

	if len(userIDs) == 0 {
		return nil
	}

	// 使用事务创建通知
	return n.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建通知
		if err := tx.Create(&notif).Error; err != nil {
			return err
		}

		// 创建用户通知关联并推送
		for _, userID := range userIDs {
			userNotif := model.UserNotification{
				UserID:         userID,
				NotificationID: notif.ID,
			}
			if err := tx.Create(&userNotif).Error; err != nil {
				return err
			}

			// 加载关联的通知用于推送
			userNotif.Notification = notif

			// 通过 WebSocket 推送
			go n.hub.PushNotification(userID, &userNotif)
		}

		return nil
	})
}

// CreateTaskNotification 创建任务通知。
//
// 根据任务状态创建通知并推送给指定用户。
// 严重级别根据任务状态自动判定: failed/error -> critical, completed/success -> info, 其他 -> warning。
//
// 参数:
//   - ctx: 上下文
//   - taskID: 任务 ID
//   - userID: 接收用户 ID
//   - title: 通知标题
//   - content: 通知内容
//   - status: 任务状态
//
// 返回: 成功返回 nil，失败返回错误
func (n *NotificationIntegrator) CreateTaskNotification(ctx context.Context, taskID, userID uint64, title, content, status string) error {
	notif := model.Notification{
		Type:      "task",
		Title:     title,
		Content:   content,
		Severity:  n.getTaskSeverity(status),
		Source:    "任务系统",
		SourceID:  fmt.Sprintf("%d", taskID),
		ActionURL: fmt.Sprintf("/tasks/%d", taskID),
	}

	return n.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&notif).Error; err != nil {
			return err
		}

		userNotif := model.UserNotification{
			UserID:         userID,
			NotificationID: notif.ID,
		}
		if err := tx.Create(&userNotif).Error; err != nil {
			return err
		}

		userNotif.Notification = notif
		go n.hub.PushNotification(userID, &userNotif)

		return nil
	})
}

// CreateSystemNotification 创建系统通知。
//
// 创建系统级别通知并推送给指定用户列表。
// 系统通知默认严重级别为 info。
//
// 参数:
//   - ctx: 上下文
//   - title: 通知标题
//   - content: 通知内容
//   - userIDs: 接收用户 ID 列表
//
// 返回: 成功返回 nil，失败返回错误
func (n *NotificationIntegrator) CreateSystemNotification(ctx context.Context, title, content string, userIDs []uint64) error {
	notif := model.Notification{
		Type:     "system",
		Title:    title,
		Content:  content,
		Severity: "info",
		Source:   "系统",
	}

	return n.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&notif).Error; err != nil {
			return err
		}

		for _, userID := range userIDs {
			userNotif := model.UserNotification{
				UserID:         userID,
				NotificationID: notif.ID,
			}
			if err := tx.Create(&userNotif).Error; err != nil {
				return err
			}

			userNotif.Notification = notif
			go n.hub.PushNotification(userID, &userNotif)
		}

		return nil
	})
}

// getAlertNotificationUsers 获取应该接收告警通知的用户。
//
// 当前实现: 获取所有 admin 和 super_admin 角色的用户。
// 后续可根据告警规则的 channels 配置确定通知用户。
//
// 参数:
//   - ctx: 上下文
//   - alert: 告警事件（预留扩展）
//
// 返回: 用户 ID 列表，失败返回错误
func (n *NotificationIntegrator) getAlertNotificationUsers(ctx context.Context, alert *model.AlertEvent) ([]uint64, error) {
	// 目前简单实现：获取所有管理员用户
	// 后续可以根据告警规则的 channels 配置确定通知用户
	var users []struct {
		ID uint64 `gorm:"column:id"`
	}

	// 查询有监控查看权限的用户
	err := n.db.WithContext(ctx).
		Table("users").
		Where("role = ? OR role = ?", "admin", "super_admin").
		Pluck("id", &users).Error

	if err != nil {
		return nil, err
	}

	userIDs := make([]uint64, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	return userIDs, nil
}

// getTaskSeverity 根据任务状态获取通知严重级别。
//
// 状态映射:
//   - failed/error -> critical
//   - completed/success -> info
//   - 其他 -> warning
//
// 参数:
//   - status: 任务状态
//
// 返回: 严重级别字符串
func (n *NotificationIntegrator) getTaskSeverity(status string) string {
	switch status {
	case "failed", "error":
		return "critical"
	case "completed", "success":
		return "info"
	default:
		return "warning"
	}
}

// PushNotificationUpdate 推送通知状态更新。
//
// 通过 WebSocket 向指定用户推送通知状态变更（已读/忽略/确认）。
//
// 参数:
//   - userID: 用户 ID
//   - notifID: 通知 ID
//   - readAt: 阅读时间
//   - dismissedAt: 忽略时间
//   - confirmedAt: 确认时间
func (n *NotificationIntegrator) PushNotificationUpdate(userID uint64, notifID uint, readAt, dismissedAt, confirmedAt *time.Time) {
	n.hub.PushUpdate(userID, notifID, readAt, dismissedAt, confirmedAt)
}
