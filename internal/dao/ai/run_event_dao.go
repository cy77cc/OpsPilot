// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现 AI 运行事件的数据访问对象。
package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// ErrRunEventCursorExpired 表示事件游标已过期。
//
// 当客户端尝试从旧的事件 ID 恢复时，如果该事件已不存在，返回此错误。
var ErrRunEventCursorExpired = errors.New("run event cursor expired")

// AIRunEventDAO 提供 AI 运行事件的数据访问功能。
type AIRunEventDAO struct {
	db *gorm.DB
}

// NewAIRunEventDAO 创建 AI 运行事件 DAO 实例。
func NewAIRunEventDAO(db *gorm.DB) *AIRunEventDAO {
	return &AIRunEventDAO{db: db}
}

// Create 创建新的事件记录。
func (d *AIRunEventDAO) Create(ctx context.Context, event *model.AIRunEvent) error {
	return d.db.WithContext(ctx).Create(event).Error
}

// ListByRun 列出运行的所有事件。
//
// 按序号升序排列，确保事件顺序正确。
//
// 参数:
//   - ctx: 上下文
//   - runID: 运行 ID
//
// 返回: 事件列表
func (d *AIRunEventDAO) ListByRun(ctx context.Context, runID string) ([]model.AIRunEvent, error) {
	var events []model.AIRunEvent
	err := d.db.WithContext(ctx).
		Where("run_id = ?", runID).
		Order("seq ASC, created_at ASC, id ASC").
		Find(&events).Error
	return events, err
}

// FindByEventID 根据事件 ID 查找事件。
//
// 参数:
//   - ctx: 上下文
//   - runID: 运行 ID
//   - eventID: 事件 ID
//
// 返回: 事件或 nil（不存在时）
func (d *AIRunEventDAO) FindByEventID(ctx context.Context, runID, eventID string) (*model.AIRunEvent, error) {
	normalizedRunID := strings.TrimSpace(runID)
	normalizedEventID := strings.TrimSpace(eventID)
	if normalizedRunID == "" || normalizedEventID == "" {
		return nil, nil
	}

	var event model.AIRunEvent
	err := d.db.WithContext(ctx).
		Where("run_id = ? AND id = ?", normalizedRunID, normalizedEventID).
		First(&event).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// ListAfterEventID 列出指定事件之后的所有事件。
//
// 用于实现 SSE 流式恢复：客户端断线重连后，从上次收到的事件 ID 继续。
//
// 参数:
//   - ctx: 上下文
//   - runID: 运行 ID
//   - lastEventID: 上次收到的事件 ID（空则返回全部）
//
// 返回: 事件列表或 ErrRunEventCursorExpired（游标过期时）
func (d *AIRunEventDAO) ListAfterEventID(ctx context.Context, runID, lastEventID string) ([]model.AIRunEvent, error) {
	normalizedRunID := strings.TrimSpace(runID)
	normalizedEventID := strings.TrimSpace(lastEventID)
	if normalizedRunID == "" {
		return nil, fmt.Errorf("run id is required")
	}
	if normalizedEventID == "" {
		return d.ListByRun(ctx, normalizedRunID)
	}

	cursor, err := d.FindByEventID(ctx, normalizedRunID, normalizedEventID)
	if err != nil {
		return nil, err
	}
	if cursor == nil {
		return nil, ErrRunEventCursorExpired
	}

	var events []model.AIRunEvent
	err = d.db.WithContext(ctx).
		Where("run_id = ? AND seq > ?", normalizedRunID, cursor.Seq).
		Order("seq ASC, created_at ASC, id ASC").
		Find(&events).Error
	return events, err
}
