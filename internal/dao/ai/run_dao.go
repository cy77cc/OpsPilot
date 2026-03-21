package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mattn/go-sqlite3"
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
	normalizeRunClientRequestID(run, "")
	return d.db.WithContext(ctx).Create(run).Error
}

func (d *AIRunDAO) FindByClientRequestID(ctx context.Context, sessionID, clientRequestID string) (*model.AIRun, error) {
	if strings.TrimSpace(clientRequestID) == "" {
		return nil, nil
	}

	var run model.AIRun
	err := d.db.WithContext(ctx).
		Where("session_id = ? AND client_request_id = ?", strings.TrimSpace(sessionID), strings.TrimSpace(clientRequestID)).
		First(&run).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (d *AIRunDAO) CreateOrReuseRunShell(ctx context.Context, userID uint64, sessionID, clientRequestID string, build func() (*model.AIRun, *model.AIChatMessage, *model.AIChatMessage)) (*model.AIRun, bool, error) {
	_ = userID

	normalizedSessionID := strings.TrimSpace(sessionID)
	normalizedClientRequestID := strings.TrimSpace(clientRequestID)
	if normalizedSessionID == "" {
		return nil, false, fmt.Errorf("session id is required")
	}
	if build == nil {
		return nil, false, fmt.Errorf("build callback is required")
	}

	if normalizedClientRequestID == "" {
		run, err := d.createRunShell(ctx, normalizedSessionID, normalizedClientRequestID, build)
		return run, true, err
	}

	run, err := d.createRunShell(ctx, normalizedSessionID, normalizedClientRequestID, build)
	if err == nil {
		return run, true, nil
	}
	if !isDuplicateKeyError(err) {
		return nil, false, err
	}

	existing, lookupErr := d.findByClientRequestIDWithRetry(ctx, normalizedSessionID, normalizedClientRequestID)
	if lookupErr != nil {
		return nil, false, lookupErr
	}
	if existing == nil {
		return nil, false, err
	}
	return existing, false, nil
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

func (d *AIRunDAO) createRunShell(ctx context.Context, sessionID, clientRequestID string, build func() (*model.AIRun, *model.AIChatMessage, *model.AIChatMessage)) (*model.AIRun, error) {
	var createdRun *model.AIRun
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		run, userMessage, assistantMessage := build()
		if run == nil || userMessage == nil || assistantMessage == nil {
			return fmt.Errorf("build callback must return run and message shells")
		}

		run.SessionID = sessionID
		if strings.TrimSpace(clientRequestID) == "" {
			run.ClientRequestID = ""
		}
		normalizeRunClientRequestID(run, clientRequestID)

		userMessage.SessionID = sessionID
		assistantMessage.SessionID = sessionID
		if strings.TrimSpace(run.UserMessageID) == "" || strings.TrimSpace(run.AssistantMessageID) == "" {
			return fmt.Errorf("run message ids are required")
		}
		if userMessage.ID != run.UserMessageID {
			userMessage.ID = run.UserMessageID
		}
		if assistantMessage.ID != run.AssistantMessageID {
			assistantMessage.ID = run.AssistantMessageID
		}

		if err := tx.WithContext(ctx).Create(run).Error; err != nil {
			return err
		}

		chatDAO := NewAIChatDAO(tx)
		if err := chatDAO.createMessage(ctx, tx, userMessage); err != nil {
			return err
		}
		if err := chatDAO.createMessage(ctx, tx, assistantMessage); err != nil {
			return err
		}

		createdRun = run
		return nil
	})
	if err != nil {
		return nil, err
	}
	return createdRun, nil
}

func (d *AIRunDAO) findByClientRequestIDWithRetry(ctx context.Context, sessionID, clientRequestID string) (*model.AIRun, error) {
	for attempt := 0; attempt < 5; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		run, err := d.FindByClientRequestID(ctx, sessionID, clientRequestID)
		if err != nil {
			return nil, err
		}
		if run != nil {
			return run, nil
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, nil
}

func normalizeRunClientRequestID(run *model.AIRun, clientRequestID string) {
	if run == nil {
		return
	}

	normalizedClientRequestID := strings.TrimSpace(clientRequestID)
	if normalizedClientRequestID == "" && strings.TrimSpace(run.ClientRequestID) != "" {
		normalizedClientRequestID = strings.TrimSpace(run.ClientRequestID)
	}
	if normalizedClientRequestID == "" {
		normalizedClientRequestID = strings.TrimSpace(run.ID)
	}
	run.ClientRequestID = normalizedClientRequestID
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}

	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code == sqlite3.ErrConstraint ||
			sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey ||
			sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
	}

	return false
}
