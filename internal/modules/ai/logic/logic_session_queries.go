package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (l *Logic) CreateSession(ctx context.Context, userID uint64, title, scene string) (*ai.AIChatSession, error) {
	if l.ChatDAO == nil {
		return nil, nil
	}

	s := &ai.AIChatSession{
		ID:     uuid.NewString(),
		UserID: userID,
		Title:  title,
		Scene:  normalizeScene(scene),
	}
	if err := l.ChatDAO.CreateSession(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

type SessionSummary struct {
	Session     ai.AIChatSession
	LastMessage *ai.AIChatMessage
}

// ListSessions 列出用户的所有会话。
func (l *Logic) ListSessions(ctx context.Context, userID uint64, scene string) ([]SessionSummary, error) {
	if l.ChatDAO == nil {
		return []SessionSummary{}, nil
	}

	rows, err := l.ChatDAO.ListSessionSummaries(ctx, userID, scene)
	if err != nil {
		return nil, err
	}

	summaries := make([]SessionSummary, 0, len(rows))
	for _, row := range rows {
		summaries = append(summaries, SessionSummary{
			Session:     row.Session(),
			LastMessage: row.LastMessage(),
		})
	}

	return summaries, nil
}

// GetSession 获取会话详情。
func (l *Logic) GetSession(ctx context.Context, userID uint64, scene, sessionID string) (*ai.AIChatSession, []ai.AIChatMessage, error) {
	if l.ChatDAO == nil {
		return nil, nil, nil
	}

	session, err := l.ChatDAO.GetSession(ctx, sessionID, userID, scene)
	if err != nil || session == nil {
		return session, nil, err
	}

	messages, err := l.ChatDAO.ListMessagesBySession(ctx, session.ID)
	if err != nil {
		return nil, nil, err
	}

	return session, messages, nil
}

// DeleteSession 删除会话。
func (l *Logic) DeleteSession(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	if l.ChatDAO == nil {
		return false, nil
	}

	session, err := l.ChatDAO.GetSession(ctx, sessionID, userID, "")
	if err != nil {
		return false, err
	}
	if session == nil {
		return false, nil
	}

	if err := l.ChatDAO.DeleteSession(ctx, session.ID, userID); err != nil {
		return false, err
	}
	return true, nil
}

// GetMessageWithOwnership 获取消息并验证所有权。
//
// 验证消息所属会话是否属于当前用户。
// 返回消息或 nil（不存在或无权限时）。
func (l *Logic) GetMessageWithOwnership(ctx context.Context, userID uint64, messageID string) (*ai.AIChatMessage, error) {
	if l.ChatDAO == nil {
		return nil, nil
	}

	// 获取消息
	message, err := l.ChatDAO.GetMessage(ctx, messageID)
	if err != nil || message == nil {
		return nil, err
	}

	// 验证会话所有权
	session, err := l.ChatDAO.GetSession(ctx, message.SessionID, userID, "")
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil // 无权限
	}

	return message, nil
}

// GetRun 获取 Run 状态。
func (l *Logic) GetRun(ctx context.Context, userID uint64, runID string) (*ai.AIRun, *ai.AIDiagnosisReport, error) {
	if l.RunDAO == nil {
		return nil, nil, nil
	}

	run, err := l.RunDAO.GetRun(ctx, runID)
	if err != nil || run == nil {
		return run, nil, err
	}

	// 验证用户权限
	if l.ChatDAO != nil {
		session, err := l.ChatDAO.GetSession(ctx, run.SessionID, userID, "")
		if err != nil {
			return nil, nil, err
		}
		if session == nil {
			return nil, nil, nil
		}
	}

	// 获取关联的诊断报告
	var report *ai.AIDiagnosisReport
	if l.DiagnosisReportDAO != nil {
		report, err = l.DiagnosisReportDAO.GetReportByRunID(ctx, run.ID)
		if err != nil {
			return nil, nil, err
		}
	}

	return run, report, nil
}

func (l *Logic) GetRunProjection(ctx context.Context, userID uint64, runID string) (*ai.AIRunProjection, error) {
	if l.RunProjectionDAO == nil || l.RunEventDAO == nil {
		return nil, nil
	}

	run, _, err := l.GetRun(ctx, userID, runID)
	if err != nil || run == nil {
		return nil, err
	}

	projection, err := l.RunProjectionDAO.GetByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if projection != nil && isSteadyProjectionStatus(projection.Status) && strings.TrimSpace(projection.ProjectionJSON) != "" {
		return projection, nil
	}

	events, err := l.RunEventDAO.ListByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	built, contents, err := airuntime.BuildProjection(events)
	if err != nil {
		return nil, err
	}
	built.Status = run.Status
	data, err := json.Marshal(built)
	if err != nil {
		return nil, err
	}
	rebuilt := &ai.AIRunProjection{
		ID:             uuid.NewString(),
		RunID:          runID,
		SessionID:      run.SessionID,
		Version:        built.Version,
		Status:         built.Status,
		ProjectionJSON: string(data),
	}
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return rebuilt, nil
	}
	value, err, _ := l.projectionGroup.Do(runID, func() (any, error) {
		if err := l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			projectionDAO := aidao.NewAIRunProjectionDAO(tx)
			contentDAO := aidao.NewAIRunContentDAO(tx)
			for _, content := range contents {
				if existing, err := contentDAO.Get(ctx, content.ID); err != nil {
					return err
				} else if existing == nil {
					if err := contentDAO.Create(ctx, content); err != nil {
						return err
					}
				}
			}
			return projectionDAO.Upsert(ctx, rebuilt)
		}); err != nil {
			return nil, err
		}
		return rebuilt, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*ai.AIRunProjection), nil
}

func (l *Logic) GetRunProjectionPayload(ctx context.Context, userID uint64, runID string, query RunProjectionQuery) (any, error) {
	projection, err := l.GetRunProjection(ctx, userID, runID)
	if err != nil || projection == nil {
		return projection, err
	}

	var decoded airuntime.RunProjection
	if err := json.Unmarshal([]byte(projection.ProjectionJSON), &decoded); err != nil {
		return nil, err
	}

	if !query.Paginate {
		return decoded, nil
	}

	events, err := l.RunEventDAO.ListByRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	paged, err := buildProjectionBlockPage(decoded, projection.CreatedAt, events, query.Cursor, query.Limit)
	if err != nil {
		return nil, err
	}
	return paged, nil
}

func (l *Logic) GetRunContent(ctx context.Context, userID uint64, contentID string) (*ai.AIRunContent, error) {
	if l.RunContentDAO == nil {
		return nil, nil
	}
	content, err := l.RunContentDAO.Get(ctx, contentID)
	if err != nil || content == nil {
		return content, err
	}
	if l.ChatDAO == nil {
		return content, nil
	}
	session, err := l.ChatDAO.GetSession(ctx, content.SessionID, userID, "")
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}
	return content, nil
}

func isSteadyProjectionStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "completed", "completed_with_tool_errors", "failed_runtime", "interrupted":
		return true
	default:
		return false
	}
}

func buildProjectionBlockPage(
	projection airuntime.RunProjection,
	fallbackCreatedAt time.Time,
	events []ai.AIRunEvent,
	cursor string,
	limit int,
) (*projectionBlockPage, error) {
	eventTimes := make(map[string]time.Time, len(events))
	for _, event := range events {
		eventTimes[event.ID] = event.CreatedAt
	}

	blocks := make([]projectionBlockMeta, 0, len(projection.Blocks))
	for _, block := range projection.Blocks {
		blocks = append(blocks, projectionBlockMeta{
			Block:     block,
			CreatedAt: projectionBlockCreatedAt(block, eventTimes, fallbackCreatedAt),
		})
	}
	sort.Slice(blocks, func(i, j int) bool {
		left := blocks[i]
		right := blocks[j]
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.Block.ID < right.Block.ID
		}
		return left.CreatedAt.Before(right.CreatedAt)
	})

	marker, err := decodeProjectionCursor(cursor)
	if err != nil {
		return nil, err
	}

	start := 0
	if marker != nil {
		start = len(blocks)
		for idx, block := range blocks {
			if block.CreatedAt.UnixNano() > marker.createdAtUnixNano {
				start = idx
				break
			}
			if block.CreatedAt.UnixNano() == marker.createdAtUnixNano && block.Block.ID > marker.blockID {
				start = idx
				break
			}
		}
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	end := start + limit
	if end > len(blocks) {
		end = len(blocks)
	}

	pageSize := 0
	if end > start {
		pageSize = end - start
	}
	pageBlocks := make([]airuntime.ProjectionBlock, 0, pageSize)
	for _, block := range blocks[start:end] {
		pageBlocks = append(pageBlocks, block.Block)
	}

	page := &projectionBlockPage{
		RunProjection: airuntime.RunProjection{
			Version:   projection.Version,
			RunID:     projection.RunID,
			SessionID: projection.SessionID,
			Status:    projection.Status,
			Summary:   projection.Summary,
			Blocks:    pageBlocks,
		},
	}
	if end < len(blocks) && end > start {
		page.HasMore = true
		page.NextCursor = encodeProjectionCursor(blocks[end-1].CreatedAt, blocks[end-1].Block.ID)
	}
	return page, nil
}

type projectionCursor struct {
	createdAtUnixNano int64
	blockID           string
}

func decodeProjectionCursor(cursor string) (*projectionCursor, error) {
	if strings.TrimSpace(cursor) == "" {
		return nil, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("%w: decode: %v", ErrInvalidProjectionCursor, err)
	}
	parts := strings.SplitN(string(decoded), "_", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return nil, ErrInvalidProjectionCursor
	}
	createdAtUnixNano, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: parse created_at: %v", ErrInvalidProjectionCursor, err)
	}
	return &projectionCursor{
		createdAtUnixNano: createdAtUnixNano,
		blockID:           parts[1],
	}, nil
}

func encodeProjectionCursor(createdAt time.Time, blockID string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d_%s", createdAt.UnixNano(), blockID)))
}

func projectionBlockCreatedAt(block airuntime.ProjectionBlock, eventTimes map[string]time.Time, fallback time.Time) time.Time {
	blockEventIDs := make([]string, 0, len(block.EventIDs)+2+len(block.Items)*4)
	blockEventIDs = append(blockEventIDs, block.EventIDs...)
	if block.StartEventID != "" {
		blockEventIDs = append(blockEventIDs, block.StartEventID)
	}
	if block.EndEventID != "" {
		blockEventIDs = append(blockEventIDs, block.EndEventID)
	}
	for _, item := range block.Items {
		if item.StartEventID != "" {
			blockEventIDs = append(blockEventIDs, item.StartEventID)
		}
		if item.EndEventID != "" {
			blockEventIDs = append(blockEventIDs, item.EndEventID)
		}
		if item.EventID != "" {
			blockEventIDs = append(blockEventIDs, item.EventID)
		}
		if item.Result != nil && item.Result.EventID != "" {
			blockEventIDs = append(blockEventIDs, item.Result.EventID)
		}
	}

	createdAt := fallback
	found := false
	for _, eventID := range blockEventIDs {
		eventTime, ok := eventTimes[eventID]
		if !ok {
			continue
		}
		if !found || eventTime.Before(createdAt) {
			createdAt = eventTime
			found = true
		}
	}
	return createdAt
}

// GetDiagnosisReport 获取诊断报告。
func (l *Logic) GetDiagnosisReport(ctx context.Context, userID uint64, reportID string) (*ai.AIDiagnosisReport, error) {
	if l.DiagnosisReportDAO == nil {
		return nil, nil
	}

	report, err := l.DiagnosisReportDAO.GetReport(ctx, reportID)
	if err != nil || report == nil {
		return report, err
	}

	// 验证用户权限
	if l.ChatDAO != nil {
		session, err := l.ChatDAO.GetSession(ctx, report.SessionID, userID, "")
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, nil
		}
	}

	return report, nil
}
