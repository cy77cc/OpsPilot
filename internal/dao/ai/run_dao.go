package ai

import (
	"context"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

type AIRunDAO struct {
	db *gorm.DB
}

type AIRunStatusUpdate struct {
	Status             string
	AssistantMessageID string
	ProgressSummary    string
	ErrorMessage       string
	IntentType         string
	AssistantType      string
}

func NewAIRunDAO(db *gorm.DB) *AIRunDAO {
	return &AIRunDAO{db: db}
}

func (d *AIRunDAO) CreateRun(ctx context.Context, run *model.AIRun) error {
	return d.db.WithContext(ctx).Create(run).Error
}

func (d *AIRunDAO) UpdateRunStatus(ctx context.Context, runID string, update AIRunStatusUpdate) error {
	updates := map[string]any{}
	if update.Status != "" {
		updates["status"] = update.Status
	}
	if update.AssistantMessageID != "" {
		updates["assistant_message_id"] = update.AssistantMessageID
	}
	if update.ProgressSummary != "" {
		updates["progress_summary"] = update.ProgressSummary
	}
	if update.ErrorMessage != "" {
		updates["error_message"] = update.ErrorMessage
	}
	if isTerminalRunStatus(update.Status) {
		updates["finished_at"] = time.Now()
	}
	if update.IntentType != "" {
		updates["intent_type"] = update.IntentType
	}
	if update.AssistantType != "" {
		updates["assistant_type"] = update.AssistantType
	}
	if len(updates) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).
		Model(&model.AIRun{}).
		Where("id = ?", runID).
		Updates(updates).Error
}

func isTerminalRunStatus(status string) bool {
	switch status {
	case "completed", "completed_with_tool_errors", "failed", "failed_runtime", "cancelled":
		return true
	default:
		return false
	}
}

func (d *AIRunDAO) GetRun(ctx context.Context, runID string) (*model.AIRun, error) {
	var run model.AIRun
	err := d.db.WithContext(ctx).Where("id = ?", runID).First(&run).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (d *AIRunDAO) ListBySession(ctx context.Context, sessionID string) ([]model.AIRun, error) {
	var runs []model.AIRun
	err := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC, id ASC").
		Find(&runs).Error
	return runs, err
}

func (d *AIRunDAO) ListBySessionIDs(ctx context.Context, sessionIDs []string) ([]model.AIRun, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	var runs []model.AIRun
	err := d.db.WithContext(ctx).
		Where("session_id IN ?", sessionIDs).
		Order("created_at ASC, id ASC").
		Find(&runs).Error
	return runs, err
}
