// Package notification 提供通知管理服务。
//
// 本包实现用户通知的查询、状态更新等功能，支持告警、任务、系统等类型通知。
// 主要功能:
//   - 通知列表查询（分页、过滤）
//   - 未读数量统计（按类型、严重级别）
//   - 通知状态管理（已读、忽略、确认）
//   - WebSocket 实时推送
package notification

import (
	"strconv"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NotificationService 是通知管理服务。
//
// 提供用户通知的查询、状态更新等 HTTP 接口处理。
type NotificationService struct {
	svcCtx *svc.ServiceContext
}

// NewNotificationService 创建通知管理服务实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//
// 返回: 通知管理服务实例
func NewNotificationService(svcCtx *svc.ServiceContext) *NotificationService {
	return &NotificationService{svcCtx: svcCtx}
}

// ListNotifications 获取通知列表。
//
// @Summary 获取通知列表
// @Description 获取当前用户的通知列表，支持分页、类型和严重级别过滤
// @Tags 通知
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param unread_only query bool false "仅未读"
// @Param type query string false "通知类型: alert/task/system/approval"
// @Param severity query string false "严重级别: critical/warning/info"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /notifications [get]
func (s *NotificationService) ListNotifications(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		httpx.Fail(c, xcode.Unauthorized, "未授权")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	unreadOnly := c.Query("unread_only") == "true"
	notifType := c.Query("type")
	severity := c.Query("severity")

	offset := (page - 1) * pageSize

	var userNotifs []model.UserNotification
	var total int64

	query := s.svcCtx.DB.Model(&model.UserNotification{}).
		Preload("Notification").
		Where("user_id = ? AND dismissed_at IS NULL", userID)

	if unreadOnly {
		query = query.Where("read_at IS NULL")
	}
	if notifType != "" {
		query = query.Joins("JOIN notifications ON notifications.id = user_notifications.notification_id").
			Where("notifications.type = ?", notifType)
	}
	if severity != "" {
		query = query.Joins("JOIN notifications ON notifications.id = user_notifications.notification_id").
			Where("notifications.severity = ?", severity)
	}

	query.Count(&total)
	query.Order("user_notifications.id DESC").Offset(offset).Limit(pageSize).Find(&userNotifs)

	httpx.OK(c, gin.H{
		"list":  userNotifs,
		"total": total,
	})
}

// UnreadCount 获取未读数量。
//
// @Summary 获取未读通知数量
// @Description 获取当前用户的未读通知数量，按类型和严重级别分组统计
// @Tags 通知
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /notifications/unread-count [get]
func (s *NotificationService) UnreadCount(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		httpx.Fail(c, xcode.Unauthorized, "未授权")
		return
	}

	var total int64
	s.svcCtx.DB.Model(&model.UserNotification{}).
		Where("user_id = ? AND read_at IS NULL AND dismissed_at IS NULL", userID).
		Count(&total)

	// 按类型统计
	type TypeCount struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	var typeCounts []TypeCount
	s.svcCtx.DB.Model(&model.UserNotification{}).
		Select("notifications.type, COUNT(*) as count").
		Joins("JOIN notifications ON notifications.id = user_notifications.notification_id").
		Where("user_notifications.user_id = ? AND user_notifications.read_at IS NULL AND user_notifications.dismissed_at IS NULL", userID).
		Group("notifications.type").
		Scan(&typeCounts)

	byType := make(map[string]int64)
	for _, tc := range typeCounts {
		byType[tc.Type] = tc.Count
	}

	// 按严重级别统计
	type SeverityCount struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	var severityCounts []SeverityCount
	s.svcCtx.DB.Model(&model.UserNotification{}).
		Select("notifications.severity, COUNT(*) as count").
		Joins("JOIN notifications ON notifications.id = user_notifications.notification_id").
		Where("user_notifications.user_id = ? AND user_notifications.read_at IS NULL AND user_notifications.dismissed_at IS NULL", userID).
		Group("notifications.severity").
		Scan(&severityCounts)

	bySeverity := make(map[string]int64)
	for _, sc := range severityCounts {
		bySeverity[sc.Severity] = sc.Count
	}

	httpx.OK(c, gin.H{
		"total":       total,
		"by_type":     byType,
		"by_severity": bySeverity,
	})
}

// MarkAsRead 标记单条通知为已读。
//
// @Summary 标记已读
// @Description 将指定通知标记为已读状态
// @Tags 通知
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "用户通知 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /notifications/{id}/read [post]
func (s *NotificationService) MarkAsRead(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		httpx.Fail(c, xcode.Unauthorized, "未授权")
		return
	}

	id := c.Param("id")
	now := time.Now()

	result := s.svcCtx.DB.Model(&model.UserNotification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read_at", now)

	if result.Error != nil {
		httpx.Fail(c, xcode.ServerError, result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		httpx.Fail(c, xcode.NotFound, "通知不存在")
		return
	}

	httpx.OK(c, nil)
}

// Dismiss 忽略通知。
//
// @Summary 忽略通知
// @Description 将指定通知标记为忽略状态，不再显示在列表中
// @Tags 通知
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "用户通知 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /notifications/{id}/dismiss [post]
func (s *NotificationService) Dismiss(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		httpx.Fail(c, xcode.Unauthorized, "未授权")
		return
	}

	id := c.Param("id")
	now := time.Now()

	result := s.svcCtx.DB.Model(&model.UserNotification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("dismissed_at", now)

	if result.Error != nil {
		httpx.Fail(c, xcode.ServerError, result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		httpx.Fail(c, xcode.NotFound, "通知不存在")
		return
	}

	httpx.OK(c, nil)
}

// Confirm 确认告警通知。
//
// @Summary 确认告警
// @Description 确认告警类型通知，同时更新告警事件状态为 confirmed
// @Tags 通知
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "用户通知 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /notifications/{id}/confirm [post]
func (s *NotificationService) Confirm(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		httpx.Fail(c, xcode.Unauthorized, "未授权")
		return
	}

	id := c.Param("id")
	now := time.Now()

	// 更新用户通知状态
	result := s.svcCtx.DB.Model(&model.UserNotification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"read_at":      now,
			"confirmed_at": now,
		})

	if result.Error != nil {
		httpx.Fail(c, xcode.ServerError, result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		httpx.Fail(c, xcode.NotFound, "通知不存在")
		return
	}

	// 如果是告警类型，更新告警状态
	var userNotif model.UserNotification
	s.svcCtx.DB.Preload("Notification").First(&userNotif, id)
	if userNotif.Notification.Type == "alert" && userNotif.Notification.SourceID != "" {
		alertID := userNotif.Notification.SourceID
		s.svcCtx.DB.Model(&model.AlertEvent{}).
			Where("id = ?", alertID).
			Update("status", "confirmed")
	}

	httpx.OK(c, nil)
}

// MarkAllAsRead 全部标记已读。
//
// @Summary 全部已读
// @Description 将当前用户所有未读通知标记为已读
// @Tags 通知
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /notifications/read-all [post]
func (s *NotificationService) MarkAllAsRead(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		httpx.Fail(c, xcode.Unauthorized, "未授权")
		return
	}

	now := time.Now()

	s.svcCtx.DB.Model(&model.UserNotification{}).
		Where("user_id = ? AND read_at IS NULL AND dismissed_at IS NULL", userID).
		Update("read_at", now)

	httpx.OK(c, nil)
}

// CreateNotification 创建通知（内部使用）。
//
// 在事务中创建通知主体和用户通知关联，供其他服务调用。
//
// 参数:
//   - notif: 通知主体模型
//   - userIDs: 接收通知的用户 ID 列表
//
// 返回: 成功返回 nil，失败返回错误
func (s *NotificationService) CreateNotification(notif *model.Notification, userIDs []uint64) error {
	return s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		// 创建通知
		if err := tx.Create(notif).Error; err != nil {
			return err
		}

		// 创建用户通知关联
		for _, userID := range userIDs {
			userNotif := model.UserNotification{
				UserID:         userID,
				NotificationID: notif.ID,
			}
			if err := tx.Create(&userNotif).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// getUserID 从上下文获取用户 ID。
//
// 尝试从 Gin 上下文中读取 "uid" 或 "user_id" 键，支持多种数值类型转换。
//
// 参数:
//   - c: Gin 上下文
//
// 返回: 用户 ID，未找到返回 0
func getUserID(c *gin.Context) uint64 {
	read := func(key string) (uint64, bool) {
		userID, exists := c.Get(key)
		if !exists {
			return 0, false
		}
		switch v := userID.(type) {
		case uint64:
			return v, true
		case uint:
			return uint64(v), true
		case int64:
			if v > 0 {
				return uint64(v), true
			}
		case int:
			if v > 0 {
				return uint64(v), true
			}
		case float64:
			if v > 0 {
				return uint64(v), true
			}
		}
		return 0, false
	}

	if uid, ok := read("uid"); ok {
		return uid
	}
	if uid, ok := read("user_id"); ok {
		return uid
	}
	return 0
}

// RegisterNotificationHandlers 注册通知相关路由。
//
// 路由列表:
//   - GET  /notifications           - 获取通知列表
//   - GET  /notifications/unread-count - 获取未读数量
//   - POST /notifications/:id/read   - 标记已读
//   - POST /notifications/:id/dismiss - 忽略通知
//   - POST /notifications/:id/confirm - 确认告警
//   - POST /notifications/read-all   - 全部已读
//
// 参数:
//   - r: Gin 路由组
//   - svcCtx: 服务上下文
func RegisterNotificationHandlers(r *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	svc := NewNotificationService(svcCtx)

	notifications := r.Group("/notifications")
	{
		notifications.GET("", svc.ListNotifications)
		notifications.GET("/unread-count", svc.UnreadCount)
		notifications.POST("/:id/read", svc.MarkAsRead)
		notifications.POST("/:id/dismiss", svc.Dismiss)
		notifications.POST("/:id/confirm", svc.Confirm)
		notifications.POST("/read-all", svc.MarkAllAsRead)
	}
}
