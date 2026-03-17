package ai

import (
	"context"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// AIChatDAO provides focused persistence for phase 1 sessions and messages.
type AIChatDAO struct {
	db *gorm.DB
}

func NewAIChatDAO(db *gorm.DB) *AIChatDAO {
	return &AIChatDAO{db: db}
}

func (d *AIChatDAO) CreateSession(ctx context.Context, session *model.AIChatSession) error {
	return d.db.WithContext(ctx).Create(session).Error
}

func (d *AIChatDAO) ListSessions(ctx context.Context, userID uint64, scene string) ([]model.AIChatSession, error) {
	var sessions []model.AIChatSession
	q := d.db.WithContext(ctx).Where("user_id = ?", userID)
	if strings.TrimSpace(scene) != "" {
		q = q.Where("scene = ?", strings.TrimSpace(scene))
	}
	err := q.Order("updated_at DESC, created_at DESC").Find(&sessions).Error
	return sessions, err
}

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

func (d *AIChatDAO) DeleteSession(ctx context.Context, sessionID string, userID uint64) error {
	return d.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", sessionID, userID).
		Delete(&model.AIChatSession{}).Error
}

func (d *AIChatDAO) CreateMessage(ctx context.Context, message *model.AIChatMessage) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if message.SessionIDNum <= 0 {
			var last model.AIChatMessage
			err := tx.
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

		if err := tx.Create(message).Error; err != nil {
			return err
		}

		return tx.Model(&model.AIChatSession{}).
			Where("id = ?", message.SessionID).
			Update("updated_at", time.Now()).
			Error
	})
}

func (d *AIChatDAO) UpdateMessage(ctx context.Context, messageID string, updates map[string]any) error {
	return d.db.WithContext(ctx).
		Model(&model.AIChatMessage{}).
		Where("id = ?", messageID).
		Updates(updates).Error
}

func (d *AIChatDAO) ListMessagesBySession(ctx context.Context, sessionID string) ([]model.AIChatMessage, error) {
	var messages []model.AIChatMessage
	err := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("session_id_num ASC, created_at ASC, id ASC").
		Find(&messages).Error
	return messages, err
}
