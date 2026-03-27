// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现 AI 聊天会话和消息的数据访问对象。
package ai

import (
	"context"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// AIChatDAO 提供 AI 会话和消息的数据访问功能。
type AIChatDAO struct {
	db *gorm.DB
}

// AIChatSessionSummaryRow 会话摘要查询结果行。
//
// 包含会话基本信息和最后一条消息的预览。
type AIChatSessionSummaryRow struct {
	ID                      string     `gorm:"column:id"`
	UserID                  uint64     `gorm:"column:user_id"`
	Scene                   string     `gorm:"column:scene"`
	Title                   string     `gorm:"column:title"`
	CreatedAt               time.Time  `gorm:"column:created_at"`
	UpdatedAt               time.Time  `gorm:"column:updated_at"`
	LastMessageID           *string    `gorm:"column:last_message_id"`
	LastMessageSessionID    *string    `gorm:"column:last_message_session_id"`
	LastMessageSessionIDNum *int       `gorm:"column:last_message_session_id_num"`
	LastMessageRole         *string    `gorm:"column:last_message_role"`
	LastMessageContent      *string    `gorm:"column:last_message_content"`
	LastMessageStatus       *string    `gorm:"column:last_message_status"`
	LastMessageCreatedAt    *time.Time `gorm:"column:last_message_created_at"`
	LastMessageUpdatedAt    *time.Time `gorm:"column:last_message_updated_at"`
}

// Session 从摘要行中提取会话模型。
func (r AIChatSessionSummaryRow) Session() model.AIChatSession {
	return model.AIChatSession{
		ID:        r.ID,
		UserID:    r.UserID,
		Scene:     r.Scene,
		Title:     r.Title,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// LastMessage 从摘要行中提取最后一条消息。
//
// 如果最后消息字段不完整，返回 nil。
func (r AIChatSessionSummaryRow) LastMessage() *model.AIChatMessage {
	if r.LastMessageID == nil || r.LastMessageSessionID == nil || r.LastMessageSessionIDNum == nil || r.LastMessageRole == nil || r.LastMessageContent == nil || r.LastMessageStatus == nil || r.LastMessageCreatedAt == nil {
		return nil
	}

	message := &model.AIChatMessage{
		ID:           *r.LastMessageID,
		SessionID:    *r.LastMessageSessionID,
		SessionIDNum: *r.LastMessageSessionIDNum,
		Role:         *r.LastMessageRole,
		Content:      *r.LastMessageContent,
		Status:       *r.LastMessageStatus,
		CreatedAt:    *r.LastMessageCreatedAt,
	}
	if r.LastMessageUpdatedAt != nil {
		message.UpdatedAt = *r.LastMessageUpdatedAt
	}
	return message
}

// NewAIChatDAO 创建 AI 聊天 DAO 实例。
func NewAIChatDAO(db *gorm.DB) *AIChatDAO {
	return &AIChatDAO{db: db}
}

// CreateSession 创建新会话。
func (d *AIChatDAO) CreateSession(ctx context.Context, session *model.AIChatSession) error {
	return d.db.WithContext(ctx).Create(session).Error
}

// ListSessions 列出用户的会话列表。
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户 ID
//   - scene: 场景过滤（可选）
//
// 返回: 会话列表（按更新时间降序）
func (d *AIChatDAO) ListSessions(ctx context.Context, userID uint64, scene string) ([]model.AIChatSession, error) {
	var sessions []model.AIChatSession
	q := d.db.WithContext(ctx).Where("user_id = ?", userID)
	if strings.TrimSpace(scene) != "" {
		q = q.Where("scene = ?", strings.TrimSpace(scene))
	}
	err := q.Order("updated_at DESC, created_at DESC").Find(&sessions).Error
	return sessions, err
}

// ListSessionSummaries 列出用户的会话摘要列表。
//
// 摘要包含会话基本信息和最后一条消息预览。
// 使用子查询优化最后消息查询性能。
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户 ID
//   - scene: 场景过滤（可选）
//
// 返回: 会话摘要列表
func (d *AIChatDAO) ListSessionSummaries(ctx context.Context, userID uint64, scene string) ([]AIChatSessionSummaryRow, error) {
	var rows []AIChatSessionSummaryRow

	latestMessageID := d.db.WithContext(ctx).
		Table("ai_chat_messages AS m2").
		Select("m2.id").
		Where("m2.session_id = s.id AND m2.deleted_at IS NULL").
		Order("m2.session_id_num DESC, m2.created_at DESC, m2.id DESC").
		Limit(1)

	q := d.db.WithContext(ctx).
		Table("ai_chat_sessions AS s").
		Select(`
			s.id,
			s.user_id,
			s.scene,
			s.title,
			s.created_at,
			s.updated_at,
			m.id AS last_message_id,
			m.session_id AS last_message_session_id,
			m.session_id_num AS last_message_session_id_num,
			m.role AS last_message_role,
			m.content AS last_message_content,
			m.status AS last_message_status,
			m.created_at AS last_message_created_at,
			m.updated_at AS last_message_updated_at
		`).
		Joins("LEFT JOIN ai_chat_messages AS m ON m.id = (?)", latestMessageID).
		Where("s.user_id = ? AND s.deleted_at IS NULL", userID)
	if strings.TrimSpace(scene) != "" {
		q = q.Where("s.scene = ?", strings.TrimSpace(scene))
	}

	err := q.Order("s.updated_at DESC, s.created_at DESC").Scan(&rows).Error
	return rows, err
}

// GetSession 获取单个会话。
//
// 参数:
//   - ctx: 上下文
//   - sessionID: 会话 ID
//   - userID: 用户 ID（权限校验）
//   - scene: 场景过滤（可选）
//
// 返回: 会话或 nil（不存在时）
func (d *AIChatDAO) GetSession(ctx context.Context, sessionID string, userID uint64, scene string) (*model.AIChatSession, error) {
	var session model.AIChatSession
	q := d.db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID)
	if strings.TrimSpace(scene) != "" {
		q = q.Where("scene = ?", strings.TrimSpace(scene))
	}
	err := q.First(&session).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteSession 删除会话。
//
// 参数:
//   - ctx: 上下文
//   - sessionID: 会话 ID
//   - userID: 用户 ID（权限校验）
//
// 返回: 错误信息
func (d *AIChatDAO) DeleteSession(ctx context.Context, sessionID string, userID uint64) error {
	return d.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", sessionID, userID).
		Delete(&model.AIChatSession{}).Error
}

// CreateMessage 创建新消息。
//
// 自动分配 session_id_num 并更新会话的 updated_at。
func (d *AIChatDAO) CreateMessage(ctx context.Context, message *model.AIChatMessage) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return d.createMessage(ctx, tx, message)
	})
}

// createMessage 内部方法：创建消息并更新会话时间。
func (d *AIChatDAO) createMessage(ctx context.Context, tx *gorm.DB, message *model.AIChatMessage) error {
	query := tx.WithContext(ctx)
	if message.SessionIDNum <= 0 {
		var last model.AIChatMessage
		err := query.
			Where("session_id = ?", message.SessionID).
			Order("session_id_num DESC").
			First(&last).Error
		switch err {
		case nil:
			message.SessionIDNum = last.SessionIDNum + 1
		case gorm.ErrRecordNotFound:
			message.SessionIDNum = 1
		default:
			return err
		}
	}

	if err := query.Create(message).Error; err != nil {
		return err
	}

	return query.Model(&model.AIChatSession{}).
		Where("id = ?", message.SessionID).
		Update("updated_at", time.Now()).
		Error
}

// UpdateMessage 更新消息。
//
// 参数:
//   - ctx: 上下文
//   - messageID: 消息 ID
//   - updates: 更新字段映射
//
// 返回: 错误信息
func (d *AIChatDAO) UpdateMessage(ctx context.Context, messageID string, updates map[string]any) error {
	return d.db.WithContext(ctx).
		Model(&model.AIChatMessage{}).
		Where("id = ?", messageID).
		Updates(updates).Error
}

// ListMessagesBySession 列出会话的所有消息。
//
// 按 session_id_num 升序排列，确保消息顺序正确。
//
// 参数:
//   - ctx: 上下文
//   - sessionID: 会话 ID
//
// 返回: 消息列表
func (d *AIChatDAO) ListMessagesBySession(ctx context.Context, sessionID string) ([]model.AIChatMessage, error) {
	var messages []model.AIChatMessage
	err := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("session_id_num ASC, created_at ASC, id ASC").
		Find(&messages).Error
	return messages, err
}

// GetMessage 根据 ID 获取单条消息。
//
// 返回消息或 nil（不存在时）。
func (d *AIChatDAO) GetMessage(ctx context.Context, messageID string) (*model.AIChatMessage, error) {
	var message model.AIChatMessage
	if err := d.db.WithContext(ctx).Where("id = ?", messageID).First(&message).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}
