package dao

import (
	"context"

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

func (d *AIChatDAO) ListSessions(ctx context.Context, userID uint64) ([]model.AIChatSession, error) {
	var sessions []model.AIChatSession
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC, created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (d *AIChatDAO) GetSession(ctx context.Context, sessionID string, userID uint64) (*model.AIChatSession, error) {
	var session model.AIChatSession
	err := d.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", sessionID, userID).
		First(&session).Error
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
	return d.db.WithContext(ctx).Create(message).Error
}

func (d *AIChatDAO) ListMessagesBySession(ctx context.Context, sessionID string) ([]model.AIChatMessage, error) {
	var messages []model.AIChatMessage
	err := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC, id ASC").
		Find(&messages).Error
	return messages, err
}
