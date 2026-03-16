package dao

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// AIChatDAO handles AI chat session and message persistence.
type AIChatDAO struct {
	db *gorm.DB
}

// NewAIChatDAO creates an AIChatDAO.
func NewAIChatDAO(db *gorm.DB) *AIChatDAO {
	return &AIChatDAO{db: db}
}

// CreateSession persists a chat session.
func (d *AIChatDAO) CreateSession(ctx context.Context, session *model.AIChatSession) error {
	return d.db.WithContext(ctx).Create(session).Error
}

// ListSessions returns sessions for a user, optionally scoped to a scene.
func (d *AIChatDAO) ListSessions(ctx context.Context, userID uint64, scene string) ([]model.AIChatSession, error) {
	var sessions []model.AIChatSession
	query := d.db.WithContext(ctx).Model(&model.AIChatSession{}).
		Where("user_id = ?", userID)
	if scene != "" {
		query = query.Where("scene = ?", scene)
	}

	err := query.Order("updated_at DESC").Find(&sessions).Error
	return sessions, err
}

// GetSession retrieves a session by ID.
func (d *AIChatDAO) GetSession(ctx context.Context, sessionID string) (*model.AIChatSession, error) {
	var session model.AIChatSession
	if err := d.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// DeleteSession removes a session by ID.
func (d *AIChatDAO) DeleteSession(ctx context.Context, sessionID string) error {
	return d.db.WithContext(ctx).
		Where("id = ?", sessionID).
		Delete(&model.AIChatSession{}).
		Error
}

// CreateMessage persists a chat message.
func (d *AIChatDAO) CreateMessage(ctx context.Context, message *model.AIChatMessage) error {
	return d.db.WithContext(ctx).Create(message).Error
}

// ListMessagesBySession returns all messages for a session.
func (d *AIChatDAO) ListMessagesBySession(ctx context.Context, sessionID string) ([]model.AIChatMessage, error) {
	var messages []model.AIChatMessage
	err := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}
