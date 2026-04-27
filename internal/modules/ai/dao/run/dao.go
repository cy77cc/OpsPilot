// Package run 提供 AI 运行记录的数据访问对象，包括：
//   - Run 创建和查询
//   - 状态更新
//   - 幂等性保证（客户端请求 ID 去重）
package run

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mattn/go-sqlite3"
	"gorm.io/gorm"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/dao/sequenceguard"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

// AIRunDAO 提供 AI 运行记录的数据访问功能。
type AIRunDAO struct {
	db *gorm.DB
}

// AIRunStatusUpdate 运行状态更新参数。
type AIRunStatusUpdate struct {
	Status             string
	AssistantMessageID string
	ProgressSummary    string
	ErrorMessage       string
	IntentType         string
	AssistantType      string
}

// NewAIRunDAO 创建 AI 运行记录 DAO 实例。
func NewAIRunDAO(db *gorm.DB) *AIRunDAO {
	return &AIRunDAO{db: db}
}

// CreateRun 创建新的运行记录。
func (d *AIRunDAO) CreateRun(ctx context.Context, run *model.AIRun) error {
	normalizeRunClientRequestID(run, "")
	return d.db.WithContext(ctx).Create(run).Error
}

// FindByClientRequestID 根据客户端请求 ID 查找运行记录。
//
// 用于实现幂等性：相同 client_request_id 返回已存在的 Run。
//
// 参数:
//   - ctx: 上下文
//   - sessionID: 会话 ID
//   - clientRequestID: 客户端请求 ID
//
// 返回: 运行记录或 nil（不存在时）
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

// CreateOrReuseRunShell 创建或复用运行外壳。
//
// 实现幂等性：如果相同 client_request_id 的 Run 已存在，返回已有记录。
// 否则创建新的 Run、用户消息和助手消息。
//
// 注意：调用方必须在调用此方法前验证 session 的所有权（即 session 属于指定用户）。
// 本方法不在 DAO 层重复验证，以避免额外的数据库查询。
//
// 参数:
//   - ctx: 上下文
//   - sessionID: 会话 ID
//   - clientRequestID: 客户端请求 ID
//   - build: 构建 Run 和消息的回调函数
//
// 返回:
//   - *model.AIRun: 运行记录
//   - bool: 是否为新创建（true=新创建，false=复用）
//   - error: 错误信息
func (d *AIRunDAO) CreateOrReuseRunShell(ctx context.Context, sessionID, clientRequestID string, build func() (*model.AIRun, *model.AIChatMessage, *model.AIChatMessage)) (*model.AIRun, bool, error) {

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

// UpdateRunStatus 更新运行状态。
//
// 参数:
//   - ctx: 上下文
//   - runID: 运行 ID
//   - update: 状态更新参数
//
// 返回: 错误信息
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

// isTerminalRunStatus 判断是否为终态。
func isTerminalRunStatus(status string) bool {
	switch status {
	case "completed", "completed_with_tool_errors", "failed", "failed_runtime", "cancelled":
		return true
	default:
		return false
	}
}

// GetRun 根据 ID 获取运行记录。
//
// 返回运行记录或 nil（不存在时）。
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

// ListBySession 列出会话的所有运行记录。
//
// 参数:
//   - ctx: 上下文
//   - sessionID: 会话 ID
//
// 返回: 运行记录列表（按创建时间升序）
func (d *AIRunDAO) ListBySession(ctx context.Context, sessionID string) ([]model.AIRun, error) {
	var runs []model.AIRun
	err := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC, id ASC").
		Find(&runs).Error
	return runs, err
}

const maxSessionIDsInQuery = 100

// ListBySessionIDs 批量列出多个会话的运行记录。
//
// 参数:
//   - ctx: 上下文
//   - sessionIDs: 会话 ID 列表
//
// 返回: 运行记录列表
func (d *AIRunDAO) ListBySessionIDs(ctx context.Context, sessionIDs []string) ([]model.AIRun, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}
	if len(sessionIDs) > maxSessionIDsInQuery {
		return nil, fmt.Errorf("too many session IDs: max %d", maxSessionIDsInQuery)
	}
	var runs []model.AIRun
	err := d.db.WithContext(ctx).
		Where("session_id IN ?", sessionIDs).
		Order("created_at ASC, id ASC").
		Find(&runs).Error
	return runs, err
}

// createRunShell 内部方法：创建运行外壳（Run + 用户消息 + 助手消息）。
func (d *AIRunDAO) createRunShell(ctx context.Context, sessionID, clientRequestID string, build func() (*model.AIRun, *model.AIChatMessage, *model.AIChatMessage)) (*model.AIRun, error) {
	unlock := sequenceguard.LockKey(sessionID)
	defer unlock()

	var createdRun *model.AIRun
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
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

			// 创建用户消息和助手消息
			// 注意：这里需要手动设置 session_id_num，因为无法调用 chat 包的私有方法
			for _, msg := range []*model.AIChatMessage{userMessage, assistantMessage} {
				if msg.SessionIDNum <= 0 {
					var last model.AIChatMessage
					err := tx.WithContext(ctx).
						Where("session_id = ?", msg.SessionID).
						Order("session_id_num DESC").
						First(&last).Error
					switch err {
					case nil:
						msg.SessionIDNum = last.SessionIDNum + 1
					case gorm.ErrRecordNotFound:
						msg.SessionIDNum = 1
					default:
						return err
					}
				}
				if err := tx.WithContext(ctx).Create(msg).Error; err != nil {
					return err
				}
			}

			createdRun = run

			// 更新会话的 updated_at
			return tx.WithContext(ctx).Model(&model.AIChatSession{}).
				Where("id = ?", sessionID).
				Update("updated_at", time.Now()).
				Error
		})
		if err == nil {
			return createdRun, nil
		}
		lastErr = err
		if !isRetryableRunShellWriteError(err) || attempt == 19 {
			return nil, err
		}
		if waitErr := waitForRunShellRetry(ctx, attempt); waitErr != nil {
			return nil, waitErr
		}
	}
	return nil, lastErr
}

// findByClientRequestIDWithRetry 带重试的查找。
//
// 在并发场景下，创建后立即查询可能因数据库延迟而查不到，
// 此方法提供短暂重试机制。
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

// normalizeRunClientRequestID 规范化客户端请求 ID。
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

func isRetryableRunShellWriteError(err error) bool {
	if err == nil {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(normalized, "database table is locked"),
		strings.Contains(normalized, "database is locked"),
		strings.Contains(normalized, "uk_ai_chat_messages_session_seq"),
		strings.Contains(normalized, "ai_chat_messages.session_id, ai_chat_messages.session_id_num"),
		strings.Contains(normalized, "duplicate entry") && strings.Contains(normalized, "uk_ai_chat_messages_session_seq"),
		strings.Contains(normalized, "duplicate key value") && strings.Contains(normalized, "uk_ai_chat_messages_session_seq"):
		return true
	default:
		return false
	}
}

func waitForRunShellRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(attempt+1) * 20 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// isDuplicateKeyError 判断是否为重复键错误。
//
// 支持 MySQL、PostgreSQL、SQLite 三种数据库。
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
