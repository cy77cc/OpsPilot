package notification

import (
	"net/http"
	"strconv"
	"time"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NotificationService struct {
	svcCtx *svc.ServiceContext
}

func NewNotificationService(svcCtx *svc.ServiceContext) *NotificationService {
	return &NotificationService{svcCtx: svcCtx}
}

// ListNotifications 获取通知列表
func (s *NotificationService) ListNotifications(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4001, "msg": "未授权"})
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

	c.JSON(http.StatusOK, gin.H{
		"code": 1000,
		"msg":  "ok",
		"data": gin.H{
			"list":  userNotifs,
			"total": total,
		},
	})
}

// UnreadCount 获取未读数量
func (s *NotificationService) UnreadCount(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4001, "msg": "未授权"})
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

	c.JSON(http.StatusOK, gin.H{
		"code": 1000,
		"msg":  "ok",
		"data": gin.H{
			"total":       total,
			"by_type":     byType,
			"by_severity": bySeverity,
		},
	})
}

// MarkAsRead 标记已读
func (s *NotificationService) MarkAsRead(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4001, "msg": "未授权"})
		return
	}

	id := c.Param("id")
	now := time.Now()

	result := s.svcCtx.DB.Model(&model.UserNotification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read_at", now)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3000, "msg": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "msg": "通知不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 1000, "msg": "ok"})
}

// Dismiss 忽略通知
func (s *NotificationService) Dismiss(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4001, "msg": "未授权"})
		return
	}

	id := c.Param("id")
	now := time.Now()

	result := s.svcCtx.DB.Model(&model.UserNotification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("dismissed_at", now)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3000, "msg": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "msg": "通知不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 1000, "msg": "ok"})
}

// Confirm 确认告警
func (s *NotificationService) Confirm(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4001, "msg": "未授权"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3000, "msg": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "msg": "通知不存在"})
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

	c.JSON(http.StatusOK, gin.H{"code": 1000, "msg": "ok"})
}

// MarkAllAsRead 全部已读
func (s *NotificationService) MarkAllAsRead(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 4001, "msg": "未授权"})
		return
	}

	now := time.Now()

	s.svcCtx.DB.Model(&model.UserNotification{}).
		Where("user_id = ? AND read_at IS NULL AND dismissed_at IS NULL", userID).
		Update("read_at", now)

	c.JSON(http.StatusOK, gin.H{"code": 1000, "msg": "ok"})
}

// CreateNotification 创建通知（内部使用）
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

// getUserID 从上下文获取用户ID
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
