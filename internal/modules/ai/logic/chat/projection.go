// Package chat 实现 AI Chat/Session/Projection 相关的查询和操作。
package chat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	projectionruntime "github.com/cy77cc/OpsPilot/internal/modules/ai/runtime/projection"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrInvalidProjectionCursor 是无效游标错误。
var ErrInvalidProjectionCursor = errors.New("invalid projection cursor")

// projectionBlockPage 是投影分页结果。
type projectionBlockPage struct {
	airuntime.RunProjection
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type projectionBlockMeta struct {
	Block     airuntime.ProjectionBlock
	CreatedAt time.Time
}

type projectionCursor struct {
	createdAtUnixNano int64
	blockID           string
}

// GetRunProjection 获取运行投影。
func GetRunProjection(ctx context.Context, l *Logic, userID uint64, runID string) (*ai.AIRunProjection, error) {
	if l.RunProjectionDAO == nil {
		return nil, nil
	}
	run, err := l.GetRun(ctx, userID, runID)
	if err != nil || run == nil {
		return nil, err
	}
	projection, err := l.RunProjectionDAO.GetByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if projection != nil && strings.TrimSpace(projection.ProjectionJSON) != "" {
		var decoded airuntime.RunProjection
		if err := json.Unmarshal([]byte(projection.ProjectionJSON), &decoded); err == nil && strings.TrimSpace(decoded.RunID) != "" {
			return projection, nil
		}
	}
	if l.RunEventDAO == nil {
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
	rebuilt := &ai.AIRunProjection{ID: uuid.NewString(), RunID: runID, SessionID: run.SessionID, Version: built.Version, Status: built.Status, ProjectionJSON: string(data)}
	if l.SvcCtx == nil || l.SvcCtx.DB == nil {
		return rebuilt, nil
	}
	if err := l.SvcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
}

// GetRunProjectionPayload 获取投影的分页负载。
func GetRunProjectionPayload(ctx context.Context, l *Logic, userID uint64, runID string, query RunProjectionQuery) (any, error) {
	projection, err := GetRunProjection(ctx, l, userID, runID)
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

func loadIncrementalProjectionState(ctx context.Context, l *Logic, runID string) (projectionruntime.State, *ai.AIRunProjection, error) {
	if l == nil || l.RunProjectionDAO == nil {
		return projectionruntime.State{}, nil, nil
	}
	current, err := l.RunProjectionDAO.GetByRunID(ctx, runID)
	if err != nil {
		return projectionruntime.State{}, nil, err
	}
	if current == nil || strings.TrimSpace(current.ProjectionJSON) == "" {
		return projectionruntime.State{}, current, nil
	}
	var decoded airuntime.RunProjection
	if err := json.Unmarshal([]byte(current.ProjectionJSON), &decoded); err != nil {
		return projectionruntime.State{}, current, nil
	}
	state := projectionruntime.FromProjection(&decoded)
	state.Version = current.Version
	if l.RunContentDAO != nil {
		if contentID := state.CurrentContentID(); strings.TrimSpace(contentID) != "" {
			content, err := l.RunContentDAO.Get(ctx, contentID)
			if err != nil {
				return projectionruntime.State{}, current, err
			}
			if content != nil {
				state.Contents = append(state.Contents, content)
			}
		}
	}
	return state, current, nil
}

func persistIncrementalProjection(ctx context.Context, l *Logic, sessionID string, state projectionruntime.State, current *ai.AIRunProjection) error {
	if l == nil || l.RunProjectionDAO == nil {
		return nil
	}
	projection := state.Projection()
	if strings.TrimSpace(projection.SessionID) == "" {
		projection.SessionID = sessionID
	}
	if strings.TrimSpace(projection.Status) == "" && current != nil {
		projection.Status = current.Status
	}
	projectionJSON, err := json.Marshal(projection)
	if err != nil {
		return err
	}
	if l.RunContentDAO != nil {
		for _, content := range state.Contents {
			if err := l.RunContentDAO.Upsert(ctx, content); err != nil {
				return err
			}
		}
	}
	projectionRow := &ai.AIRunProjection{
		ID:             uuid.NewString(),
		RunID:          projection.RunID,
		SessionID:      projection.SessionID,
		Version:        projection.Version,
		Status:         projection.Status,
		ProjectionJSON: string(projectionJSON),
	}
	if current != nil {
		projectionRow.ID = current.ID
	}
	return l.RunProjectionDAO.Upsert(ctx, projectionRow)
}

// GetRunContent 获取运行内容。
func GetRunContent(ctx context.Context, l *Logic, userID uint64, contentID string) (*ai.AIRunContent, error) {
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

func buildProjectionBlockPage(projection airuntime.RunProjection, fallbackCreatedAt time.Time, events []ai.AIRunEvent, cursor string, limit int) (*projectionBlockPage, error) {
	eventTimes := make(map[string]time.Time, len(events))
	for _, event := range events {
		eventTimes[event.ID] = event.CreatedAt
	}
	blocks := make([]projectionBlockMeta, 0, len(projection.Blocks))
	for _, block := range projection.Blocks {
		blocks = append(blocks, projectionBlockMeta{Block: block, CreatedAt: projectionBlockCreatedAt(block, eventTimes, fallbackCreatedAt)})
	}
	sort.Slice(blocks, func(i, j int) bool {
		l, r := blocks[i], blocks[j]
		if l.CreatedAt.Equal(r.CreatedAt) {
			return l.Block.ID < r.Block.ID
		}
		return l.CreatedAt.Before(r.CreatedAt)
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
	page := &projectionBlockPage{RunProjection: airuntime.RunProjection{Version: projection.Version, RunID: projection.RunID, SessionID: projection.SessionID, Status: projection.Status, Summary: projection.Summary, Blocks: pageBlocks}}
	if end < len(blocks) && end > start {
		page.HasMore = true
		page.NextCursor = encodeProjectionCursor(blocks[end-1].CreatedAt, blocks[end-1].Block.ID)
	}
	return page, nil
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
	return &projectionCursor{createdAtUnixNano: createdAtUnixNano, blockID: parts[1]}, nil
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
