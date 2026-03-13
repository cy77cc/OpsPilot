package state

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatMessageRecord struct {
	ID              string           `json:"id"`
	Role            string           `json:"role"`
	Content         string           `json:"content"`
	Thinking        string           `json:"thinking,omitempty"`
	Status          string           `json:"status,omitempty"`
	TraceID         string           `json:"trace_id,omitempty"`
	ThoughtChain    []map[string]any `json:"thought_chain,omitempty"`
	Recommendations []map[string]any `json:"recommendations,omitempty"`
	RawEvidence     []string         `json:"raw_evidence,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type ChatBlockRecord struct {
	ID          string         `json:"id"`
	BlockType   string         `json:"block_type"`
	Position    int            `json:"position"`
	Status      string         `json:"status,omitempty"`
	Title       string         `json:"title,omitempty"`
	ContentText string         `json:"content_text,omitempty"`
	ContentJSON map[string]any `json:"content_json,omitempty"`
	Streaming   bool           `json:"streaming,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ChatTurnRecord struct {
	ID           string            `json:"id"`
	Role         string            `json:"role"`
	Status       string            `json:"status,omitempty"`
	Phase        string            `json:"phase,omitempty"`
	TraceID      string            `json:"trace_id,omitempty"`
	ParentTurnID string            `json:"parent_turn_id,omitempty"`
	Blocks       []ChatBlockRecord `json:"blocks,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

type ChatSessionRecord struct {
	ID        string              `json:"id"`
	Scene     string              `json:"scene"`
	Title     string              `json:"title"`
	Messages  []ChatMessageRecord `json:"messages,omitempty"`
	Turns     []ChatTurnRecord    `json:"turns,omitempty"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type ChatStore struct {
	db *gorm.DB
}

func NewChatStore(db *gorm.DB) *ChatStore {
	return &ChatStore{db: db}
}

func (s *ChatStore) EnsureSession(ctx context.Context, sessionID string, userID uint64, scene, title string) error {
	if s == nil || s.db == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	var row model.AIChatSession
	err := s.db.WithContext(ctx).Where("id = ?", sessionID).First(&row).Error
	if err == nil {
		return nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	return s.db.WithContext(ctx).Create(&model.AIChatSession{
		ID:     strings.TrimSpace(sessionID),
		UserID: userID,
		Scene:  strings.TrimSpace(scene),
		Title:  strings.TrimSpace(title),
	}).Error
}

func (s *ChatStore) AppendUserMessage(ctx context.Context, sessionID string, userID uint64, scene, title, content string) error {
	if err := s.EnsureSession(ctx, sessionID, userID, scene, title); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(&model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Role:      "user",
		Content:   content,
		Status:    "completed",
	}).Error
}

func (s *ChatStore) CreateAssistantMessage(ctx context.Context, sessionID string, userID uint64, scene, title, turnID string) (string, error) {
	if err := s.EnsureSession(ctx, sessionID, userID, scene, title); err != nil {
		return "", err
	}
	msgID := uuid.NewString()
	meta, _ := json.Marshal(map[string]any{"turn_id": strings.TrimSpace(turnID)})
	return msgID, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model.AIChatMessage{
			ID:           msgID,
			SessionID:    sessionID,
			Role:         "assistant",
			Status:       "streaming",
			MetadataJSON: string(meta),
		}).Error; err != nil {
			return err
		}
		if strings.TrimSpace(turnID) == "" {
			return nil
		}
		return tx.FirstOrCreate(&model.AIChatTurn{}, model.AIChatTurn{
			ID:        turnID,
			SessionID: sessionID,
			Role:      "assistant",
			Status:    "streaming",
		}).Error
	})
}

func (s *ChatStore) UpdateAssistantMessage(ctx context.Context, sessionID, messageID, turnID string, record ChatMessageRecord) error {
	if s == nil || s.db == nil {
		return nil
	}
	meta, _ := json.Marshal(map[string]any{
		"trace_id":        record.TraceID,
		"turn_id":         strings.TrimSpace(turnID),
		"thought_chain":   record.ThoughtChain,
		"recommendations": record.Recommendations,
		"raw_evidence":    record.RawEvidence,
	})
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AIChatMessage{}).
			Where("id = ? AND session_id = ?", messageID, sessionID).
			Updates(map[string]any{
				"content":       record.Content,
				"thinking":      record.Thinking,
				"status":        firstNonEmpty(record.Status, "completed"),
				"metadata_json": string(meta),
			}).Error; err != nil {
			return err
		}
		if strings.TrimSpace(turnID) == "" {
			return nil
		}
		return syncTurn(tx, sessionID, turnID, record)
	})
}

func (s *ChatStore) ListSessions(ctx context.Context, userID uint64, scene string) ([]ChatSessionRecord, error) {
	rows := make([]model.AIChatSession, 0)
	q := s.db.WithContext(ctx).Order("updated_at desc").Where("user_id = ?", userID)
	if strings.TrimSpace(scene) != "" {
		q = q.Where("scene = ?", strings.TrimSpace(scene))
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ChatSessionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, ChatSessionRecord{
			ID:        row.ID,
			Scene:     row.Scene,
			Title:     row.Title,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (s *ChatStore) CurrentSession(ctx context.Context, userID uint64, scene string, includeMessages bool) (*ChatSessionRecord, error) {
	rows, err := s.ListSessions(ctx, userID, scene)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return s.GetSession(ctx, userID, scene, rows[0].ID, includeMessages)
}

func (s *ChatStore) GetSession(ctx context.Context, userID uint64, scene, sessionID string, includeMessages bool) (*ChatSessionRecord, error) {
	var row model.AIChatSession
	q := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID)
	if strings.TrimSpace(scene) != "" {
		q = q.Where("scene = ?", strings.TrimSpace(scene))
	}
	if err := q.First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	record := &ChatSessionRecord{
		ID:        row.ID,
		Scene:     row.Scene,
		Title:     row.Title,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	if !includeMessages {
		return record, nil
	}
	msgRows := make([]model.AIChatMessage, 0)
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&msgRows).Error; err != nil {
		return nil, err
	}
	for _, msg := range msgRows {
		record.Messages = append(record.Messages, decodeMessage(msg))
	}
	turnRows := make([]model.AIChatTurn, 0)
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&turnRows).Error; err != nil {
		return nil, err
	}
	for _, turn := range turnRows {
		blockRows := make([]model.AIChatBlock, 0)
		if err := s.db.WithContext(ctx).Where("turn_id = ?", turn.ID).Order("position asc").Find(&blockRows).Error; err != nil {
			return nil, err
		}
		blocks := make([]ChatBlockRecord, 0, len(blockRows))
		for _, block := range blockRows {
			blocks = append(blocks, decodeBlock(block))
		}
		record.Turns = append(record.Turns, ChatTurnRecord{
			ID:           turn.ID,
			Role:         turn.Role,
			Status:       turn.Status,
			Phase:        turn.Phase,
			TraceID:      turn.TraceID,
			ParentTurnID: turn.ParentTurnID,
			Blocks:       blocks,
			CreatedAt:    turn.CreatedAt,
			UpdatedAt:    turn.UpdatedAt,
			CompletedAt:  turn.CompletedAt,
		})
	}
	return record, nil
}

func (s *ChatStore) Clone(ctx context.Context, userID uint64, sourceID, newID, title string) (*ChatSessionRecord, error) {
	src, err := s.GetSession(ctx, userID, "", sourceID, true)
	if err != nil || src == nil {
		return nil, err
	}
	if err := s.EnsureSession(ctx, newID, userID, src.Scene, firstNonEmpty(strings.TrimSpace(title), src.Title)); err != nil {
		return nil, err
	}
	for _, msg := range src.Messages {
		meta, _ := json.Marshal(map[string]any{
			"trace_id":        msg.TraceID,
			"thought_chain":   msg.ThoughtChain,
			"recommendations": msg.Recommendations,
			"raw_evidence":    msg.RawEvidence,
		})
		if err := s.db.WithContext(ctx).Create(&model.AIChatMessage{
			ID:           uuid.NewString(),
			SessionID:    newID,
			Role:         msg.Role,
			Content:      msg.Content,
			Thinking:     msg.Thinking,
			Status:       msg.Status,
			MetadataJSON: string(meta),
		}).Error; err != nil {
			return nil, err
		}
	}
	for _, turn := range src.Turns {
		newTurnID := uuid.NewString()
		if err := s.db.WithContext(ctx).Create(&model.AIChatTurn{
			ID:          newTurnID,
			SessionID:   newID,
			Role:        turn.Role,
			Status:      turn.Status,
			Phase:       turn.Phase,
			TraceID:     turn.TraceID,
			CompletedAt: turn.CompletedAt,
		}).Error; err != nil {
			return nil, err
		}
		for _, block := range turn.Blocks {
			raw, _ := json.Marshal(block.ContentJSON)
			if err := s.db.WithContext(ctx).Create(&model.AIChatBlock{
				ID:          uuid.NewString(),
				TurnID:      newTurnID,
				BlockType:   block.BlockType,
				Position:    block.Position,
				Status:      block.Status,
				Title:       block.Title,
				ContentText: block.ContentText,
				ContentJSON: string(raw),
				Streaming:   block.Streaming,
			}).Error; err != nil {
				return nil, err
			}
		}
	}
	return s.GetSession(ctx, userID, src.Scene, newID, true)
}

func (s *ChatStore) UpdateTitle(ctx context.Context, userID uint64, sessionID, title string) error {
	return s.db.WithContext(ctx).Model(&model.AIChatSession{}).
		Where("id = ? AND user_id = ?", sessionID, userID).
		Update("title", strings.TrimSpace(title)).Error
}

func (s *ChatStore) Delete(ctx context.Context, userID uint64, sessionID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("turn_id IN (?)", tx.Model(&model.AIChatTurn{}).Select("id").Where("session_id = ?", sessionID)).
			Delete(&model.AIChatBlock{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&model.AIChatTurn{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&model.AIChatMessage{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", sessionID, userID).Delete(&model.AIChatSession{}).Error
	})
}

func syncTurn(tx *gorm.DB, sessionID, turnID string, record ChatMessageRecord) error {
	status := firstNonEmpty(record.Status, "completed")
	now := time.Now().UTC()
	updates := map[string]any{
		"role":       "assistant",
		"status":     status,
		"phase":      status,
		"trace_id":   record.TraceID,
		"updated_at": now,
	}
	if status == "completed" {
		updates["completed_at"] = now
	}
	if err := tx.Where("id = ?", turnID).Assign(model.AIChatTurn{
		SessionID: sessionID,
		Role:      "assistant",
		Status:    status,
		Phase:     status,
		TraceID:   record.TraceID,
	}).FirstOrCreate(&model.AIChatTurn{}).Error; err != nil {
		return err
	}
	if err := tx.Model(&model.AIChatTurn{}).Where("id = ?", turnID).Updates(updates).Error; err != nil {
		return err
	}
	if err := tx.Where("turn_id = ?", turnID).Delete(&model.AIChatBlock{}).Error; err != nil {
		return err
	}
	blocks := buildBlocks(record)
	for i, block := range blocks {
		raw, _ := json.Marshal(block.ContentJSON)
		if err := tx.Create(&model.AIChatBlock{
			ID:          uuid.NewString(),
			TurnID:      turnID,
			BlockType:   block.BlockType,
			Position:    i,
			Status:      block.Status,
			Title:       block.Title,
			ContentText: block.ContentText,
			ContentJSON: string(raw),
			Streaming:   block.Streaming,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func buildBlocks(record ChatMessageRecord) []ChatBlockRecord {
	blocks := make([]ChatBlockRecord, 0, len(record.ThoughtChain)+2)
	if strings.TrimSpace(record.Thinking) != "" {
		blocks = append(blocks, ChatBlockRecord{
			BlockType:   "thinking",
			Status:      firstNonEmpty(record.Status, "completed"),
			ContentText: record.Thinking,
		})
	}
	for _, stage := range record.ThoughtChain {
		blocks = append(blocks, ChatBlockRecord{
			BlockType:   "stage",
			Status:      stringValue(stage["status"]),
			Title:       stringValue(stage["title"]),
			ContentText: stringValue(stage["content"]),
			ContentJSON: stage,
		})
	}
	if strings.TrimSpace(record.Content) != "" {
		blocks = append(blocks, ChatBlockRecord{
			BlockType:   "text",
			Status:      firstNonEmpty(record.Status, "completed"),
			ContentText: record.Content,
		})
	}
	return blocks
}

func decodeMessage(row model.AIChatMessage) ChatMessageRecord {
	meta := map[string]any{}
	if strings.TrimSpace(row.MetadataJSON) != "" {
		_ = json.Unmarshal([]byte(row.MetadataJSON), &meta)
	}
	return ChatMessageRecord{
		ID:              row.ID,
		Role:            row.Role,
		Content:         row.Content,
		Thinking:        row.Thinking,
		Status:          row.Status,
		TraceID:         stringValue(meta["trace_id"]),
		ThoughtChain:    mapSlice(meta["thought_chain"]),
		Recommendations: mapSlice(meta["recommendations"]),
		RawEvidence:     stringSlice(meta["raw_evidence"]),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func decodeBlock(row model.AIChatBlock) ChatBlockRecord {
	content := map[string]any{}
	if strings.TrimSpace(row.ContentJSON) != "" {
		_ = json.Unmarshal([]byte(row.ContentJSON), &content)
	}
	return ChatBlockRecord{
		ID:          row.ID,
		BlockType:   row.BlockType,
		Position:    row.Position,
		Status:      row.Status,
		Title:       row.Title,
		ContentText: row.ContentText,
		ContentJSON: content,
		Streaming:   row.Streaming,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapSlice(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		out, ok := value.([]map[string]any)
		if ok {
			return out
		}
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if row, ok := item.(map[string]any); ok {
			out = append(out, row)
		}
	}
	return out
}

func stringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		out, ok := value.([]string)
		if ok {
			return out
		}
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
