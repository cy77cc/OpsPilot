package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// AIRunDAO handles AIRun persistence.
type AIRunDAO struct {
	db *gorm.DB
}

// RunStatusUpdate describes fields modified when updating a run's status.
type RunStatusUpdate struct {
	Status       string
	ErrorMessage *string
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

// NewAIRunDAO creates an AIRunDAO.
func NewAIRunDAO(db *gorm.DB) *AIRunDAO {
	return &AIRunDAO{db: db}
}

// CreateRun persists a run record.
func (d *AIRunDAO) CreateRun(ctx context.Context, run *model.AIRun) error {
	return d.db.WithContext(ctx).Create(run).Error
}

// UpdateRunStatus updates the status-related fields of a run.
func (d *AIRunDAO) UpdateRunStatus(ctx context.Context, runID string, update RunStatusUpdate) error {
	updates := map[string]interface{}{
		"status": update.Status,
	}
	if update.ErrorMessage != nil {
		updates["error_message"] = *update.ErrorMessage
	}
	if update.StartedAt != nil {
		updates["started_at"] = *update.StartedAt
	}
	if update.FinishedAt != nil {
		updates["finished_at"] = *update.FinishedAt
	}

	return d.db.WithContext(ctx).
		Model(&model.AIRun{}).
		Where("id = ?", runID).
		Updates(updates).
		Error
}

// GetRun retrieves a run by ID.
func (d *AIRunDAO) GetRun(ctx context.Context, runID string) (*model.AIRun, error) {
	row := d.db.WithContext(ctx).Raw(`
SELECT
  id,
  session_id,
  user_message_id,
  assistant_message_id,
  intent_type,
  assistant_type,
  risk_level,
  status,
  trace_id,
  error_message,
  started_at,
  finished_at,
  created_at,
  updated_at
FROM ai_runs
WHERE id = ?
`, runID).Row()

	var record struct {
		ID                 string
		SessionID          string
		UserMessageID      string
		AssistantMessageID sql.NullString
		IntentType         string
		AssistantType      string
		RiskLevel          string
		Status             string
		TraceID            string
		ErrorMessage       string
		StartedAt          sql.NullString
		FinishedAt         sql.NullString
		CreatedAt          time.Time
		UpdatedAt          time.Time
	}
	if err := row.Scan(
		&record.ID,
		&record.SessionID,
		&record.UserMessageID,
		&record.AssistantMessageID,
		&record.IntentType,
		&record.AssistantType,
		&record.RiskLevel,
		&record.Status,
		&record.TraceID,
		&record.ErrorMessage,
		&record.StartedAt,
		&record.FinishedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return nil, err
	}

	run := &model.AIRun{
		ID:            record.ID,
		SessionID:     record.SessionID,
		UserMessageID: record.UserMessageID,
		IntentType:    record.IntentType,
		AssistantType: record.AssistantType,
		RiskLevel:     record.RiskLevel,
		Status:        record.Status,
		TraceID:       record.TraceID,
		ErrorMessage:  record.ErrorMessage,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
	if record.AssistantMessageID.Valid {
		run.AssistantMessageID = &record.AssistantMessageID.String
	}
	if record.StartedAt.Valid {
		parsed, err := parseDBTime(record.StartedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse started_at: %w", err)
		}
		run.StartedAt = &parsed
	}
	if record.FinishedAt.Valid {
		parsed, err := parseDBTime(record.FinishedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse finished_at: %w", err)
		}
		run.FinishedAt = &parsed
	}
	return run, nil
}

func parseDBTime(raw string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", raw)
}
