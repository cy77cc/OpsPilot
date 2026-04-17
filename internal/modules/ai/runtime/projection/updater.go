package projection

import (
	"encoding/json"
	"strings"

	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/google/uuid"
)

// Event is an incremental projection input.
type Event struct {
	ID   string
	Type string
	Text string
	Data any
}

// Block mirrors the current projection block contract while preserving a
// light-weight text view for incremental text updates.
type Block struct {
	Text string `json:"text,omitempty"`

	ID           string                             `json:"id,omitempty"`
	Type         string                             `json:"type,omitempty"`
	Title        string                             `json:"title,omitempty"`
	Agent        string                             `json:"agent,omitempty"`
	EventIDs     []string                           `json:"event_ids,omitempty"`
	Data         map[string]any                     `json:"data,omitempty"`
	StartEventID string                             `json:"start_event_id,omitempty"`
	EndEventID   string                             `json:"end_event_id,omitempty"`
	Lazy         bool                               `json:"lazy,omitempty"`
	Items        []airuntime.ProjectionExecutorItem `json:"items,omitempty"`
}

// State stores the active incremental projection snapshot.
type State struct {
	Version   int                          `json:"version"`
	RunID     string                       `json:"run_id,omitempty"`
	SessionID string                       `json:"session_id,omitempty"`
	Status    string                       `json:"status,omitempty"`
	Summary   *airuntime.ProjectionSummary `json:"summary,omitempty"`
	Blocks    []Block                      `json:"blocks,omitempty"`

	Contents []*ai.AIRunContent `json:"-"`

	currentExecutorIndex int `json:"-"`
	currentContentID     string `json:"-"`
}

// ApplyEvent applies one event to the active projection state.
func ApplyEvent(state State, event Event) State {
	next := state
	next.Version++
	next.ensureIdentifiers(event)

	switch normalizeEventType(event.Type) {
	case "delta":
		next.applyDelta(event)
	case "tool_call":
		next.applyToolCall(event)
	case "tool_result":
		next.applyToolResult(event)
	case "tool_approval":
		next.closeExecutor()
		next.appendBlock(Block{
			ID:       blockID("tool_approval", len(next.Blocks)+1),
			Type:     "tool_approval",
			Title:    "等待审批",
			EventIDs: []string{next.eventID(event)},
			Data: map[string]any{
				"approval_id": stringValue(event.Data, "approval_id"),
				"call_id":     stringValue(event.Data, "call_id"),
				"tool_name":   stringValue(event.Data, "tool_name"),
			},
		})
	case "agent_handoff":
		next.closeExecutor()
		next.appendBlock(Block{
			ID:       blockID("handoff", len(next.Blocks)+1),
			Type:     "agent_handoff",
			Title:    "任务转交",
			EventIDs: []string{next.eventID(event)},
			Data: map[string]any{
				"from":   stringValue(event.Data, "from"),
				"to":     stringValue(event.Data, "to"),
				"intent": stringValue(event.Data, "intent"),
			},
		})
	case "delegation_node":
		next.closeExecutor()
		next.appendBlock(Block{
			ID:       blockID("delegation", len(next.Blocks)+1),
			Type:     "delegation.node",
			Title:    stringValue(event.Data, "title"),
			Agent:    stringValue(event.Data, "agent_name"),
			EventIDs: []string{next.eventID(event)},
			Data: map[string]any{
				"delegation_id": stringValue(event.Data, "delegation_id"),
				"status":        stringValue(event.Data, "status"),
				"summary":       stringValue(event.Data, "summary"),
				"risk_level":    stringValue(event.Data, "risk_level"),
			},
		})
	case "run_state":
		if status := strings.TrimSpace(stringValue(event.Data, "status")); status != "" {
			next.Status = status
		}
	case "done":
		next.closeExecutor()
		if status := strings.TrimSpace(stringValue(event.Data, "status")); status != "" {
			next.Status = status
		}
		if summary := strings.TrimSpace(stringValue(event.Data, "summary")); summary != "" {
			next.Summary = &airuntime.ProjectionSummary{
				Title:       "结论",
				ContentMode: "inline",
				Content:     summary,
			}
		}
	case "error":
		next.closeExecutor()
		next.Status = "failed_runtime"
		next.appendBlock(Block{
			ID:       blockID("error", len(next.Blocks)+1),
			Type:     "error",
			Title:    "执行错误",
			EventIDs: []string{next.eventID(event)},
			Data: map[string]any{
				"message": stringValue(event.Data, "message"),
				"code":    stringValue(event.Data, "code"),
			},
		})
	default:
		// Unknown events still advance version to preserve monotonicity.
	}

	return next
}

// Projection converts the incremental state to the public projection contract.
func (s State) Projection() *airuntime.RunProjection {
	blocks := make([]airuntime.ProjectionBlock, 0, len(s.Blocks))
	for _, block := range s.Blocks {
		blocks = append(blocks, block.ProjectionBlock())
	}
	return &airuntime.RunProjection{
		Version:   s.Version,
		RunID:     s.RunID,
		SessionID: s.SessionID,
		Status:    s.Status,
		Summary:   s.Summary,
		Blocks:    blocks,
	}
}

// FromProjection reconstructs incremental state from the public projection.
func FromProjection(projection *airuntime.RunProjection) State {
	if projection == nil {
		return State{}
	}
	state := State{
		Version:   projection.Version,
		RunID:     projection.RunID,
		SessionID: projection.SessionID,
		Status:    projection.Status,
		Summary:   projection.Summary,
		Blocks:    make([]Block, 0, len(projection.Blocks)),
	}
	for _, block := range projection.Blocks {
		state.Blocks = append(state.Blocks, blockFromProjection(block))
	}
	if len(state.Blocks) > 0 && state.Blocks[len(state.Blocks)-1].Type == "executor" {
		state.currentExecutorIndex = len(state.Blocks) - 1
	} else {
		state.currentExecutorIndex = -1
	}
	if executor := state.currentExecutor(); executor != nil && len(executor.Items) > 0 {
		last := executor.Items[len(executor.Items)-1]
		if last.Type == "content" {
			state.currentContentID = last.ContentID
		}
	}
	return state
}

func (s *State) ensureIdentifiers(event Event) {
	if strings.TrimSpace(s.RunID) == "" {
		if runID := stringValue(event.Data, "run_id"); runID != "" {
			s.RunID = runID
		}
	}
	if strings.TrimSpace(s.SessionID) == "" {
		if sessionID := stringValue(event.Data, "session_id"); sessionID != "" {
			s.SessionID = sessionID
		}
	}
}

func (s *State) applyDelta(event Event) {
	executor := s.ensureExecutor(event)
	text := event.Text
	if text == "" {
		text = stringValue(event.Data, "content")
	}
	if text == "" {
		return
	}
	executor.Text += text
	eventID := s.eventID(event)
	executor.EndEventID = eventID
	if content := s.currentContent(); content != nil && len(executor.Items) > 0 {
		lastIdx := len(executor.Items) - 1
		if executor.Items[lastIdx].Type == "content" && executor.Items[lastIdx].ContentID == content.ID {
			content.BodyText += text
			content.SummaryText = truncate(content.BodyText, 200)
			content.SizeBytes = int64(len(content.BodyText))
			executor.Items[lastIdx].EndEventID = eventID
			return
		}
	}
	contentID := uuid.NewString()
	content := &ai.AIRunContent{
		ID:          contentID,
		RunID:       s.RunID,
		SessionID:   s.SessionID,
		ContentKind: "executor_content",
		Encoding:    "text",
		SummaryText: truncate(text, 200),
		BodyText:    text,
		SizeBytes:   int64(len(text)),
	}
	s.Contents = append(s.Contents, content)
	s.currentContentID = contentID
	executor.Items = append(executor.Items, airuntime.ProjectionExecutorItem{
		ID:           uuid.NewString(),
		Type:         "content",
		ContentID:    contentID,
		StartEventID: eventID,
		EndEventID:   eventID,
	})
}

func (s *State) applyToolCall(event Event) {
	executor := s.ensureExecutor(event)
	argsJSON, _ := json.Marshal(normalizeMap(event.Data))
	contentID := uuid.NewString()
	s.Contents = append(s.Contents, &ai.AIRunContent{
		ID:          contentID,
		RunID:       s.RunID,
		SessionID:   s.SessionID,
		ContentKind: "tool_arguments",
		Encoding:    "json",
		SummaryText: stringValue(event.Data, "tool_name"),
		BodyJSON:    string(argsJSON),
		SizeBytes:   int64(len(argsJSON)),
	})
	executor.Items = append(executor.Items, airuntime.ProjectionExecutorItem{
		ID:                 uuid.NewString(),
		Type:               "tool_call",
		ToolCallID:         stringValue(event.Data, "call_id"),
		ToolName:           stringValue(event.Data, "tool_name"),
		EventID:            s.eventID(event),
		ArgumentsContentID: contentID,
	})
	executor.EndEventID = s.eventID(event)
}

func (s *State) applyToolResult(event Event) {
	executor := s.currentExecutor()
	if executor == nil {
		return
	}
	content := strings.TrimSpace(stringValue(event.Data, "content"))
	resultContentID := uuid.NewString()
	s.Contents = append(s.Contents, &ai.AIRunContent{
		ID:          resultContentID,
		RunID:       s.RunID,
		SessionID:   s.SessionID,
		ContentKind: "tool_result",
		Encoding:    "text",
		SummaryText: truncate(content, 200),
		BodyText:    content,
		SizeBytes:   int64(len(content)),
	})
	for i := len(executor.Items) - 1; i >= 0; i-- {
		if executor.Items[i].Type == "tool_call" && executor.Items[i].ToolCallID == stringValue(event.Data, "call_id") {
			executor.Items[i].Result = &airuntime.ProjectionToolResult{
				EventID:         s.eventID(event),
				Status:          stringValue(event.Data, "status"),
				Preview:         truncate(content, 200),
				ResultContentID: resultContentID,
			}
			break
		}
	}
	executor.EndEventID = s.eventID(event)
}

func (s *State) ensureExecutor(event Event) *Block {
	if executor := s.currentExecutor(); executor != nil {
		if executor.Type == "executor" {
			return executor
		}
	}
	block := Block{
		ID:           blockID("executor", len(s.Blocks)+1),
		Type:         "executor",
		Title:        "执行过程",
		Agent:        stringValue(event.Data, "agent"),
		StartEventID: s.eventID(event),
		EndEventID:   s.eventID(event),
		Lazy:         true,
		Items:        make([]airuntime.ProjectionExecutorItem, 0, 4),
	}
	s.Blocks = append(s.Blocks, block)
	s.currentExecutorIndex = len(s.Blocks) - 1
	return &s.Blocks[s.currentExecutorIndex]
}

func (s *State) currentExecutor() *Block {
	if s.currentExecutorIndex < 0 || s.currentExecutorIndex >= len(s.Blocks) {
		return nil
	}
	block := &s.Blocks[s.currentExecutorIndex]
	if block.Type != "executor" {
		return nil
	}
	return block
}

func (s *State) closeExecutor() {
	s.currentExecutorIndex = -1
	s.currentContentID = ""
}

func (s *State) appendBlock(block Block) {
	s.Blocks = append(s.Blocks, block)
}

func (s State) eventID(event Event) string {
	if strings.TrimSpace(event.ID) != "" {
		return event.ID
	}
	return uuid.NewString()
}

func (b Block) ProjectionBlock() airuntime.ProjectionBlock {
	return airuntime.ProjectionBlock{
		ID:           b.ID,
		Type:         b.Type,
		Title:        b.Title,
		Agent:        b.Agent,
		EventIDs:     append([]string(nil), b.EventIDs...),
		Data:         cloneMap(b.Data),
		StartEventID: b.StartEventID,
		EndEventID:   b.EndEventID,
		Lazy:         b.Lazy,
		Items:        append([]airuntime.ProjectionExecutorItem(nil), b.Items...),
	}
}

func blockFromProjection(block airuntime.ProjectionBlock) Block {
	return Block{
		ID:           block.ID,
		Type:         block.Type,
		Title:        block.Title,
		Agent:        block.Agent,
		EventIDs:     append([]string(nil), block.EventIDs...),
		Data:         cloneMap(block.Data),
		StartEventID: block.StartEventID,
		EndEventID:   block.EndEventID,
		Lazy:         block.Lazy,
		Items:        append([]airuntime.ProjectionExecutorItem(nil), block.Items...),
	}
}

func normalizeEventType(eventType string) string {
	eventType = strings.TrimSpace(strings.ToLower(eventType))
	if strings.HasSuffix(eventType, ".delta") {
		return "delta"
	}
	return strings.TrimPrefix(eventType, "assistant.")
}

func stringValue(data any, key string) string {
	m := normalizeMap(data)
	if m == nil {
		return ""
	}
	value, _ := m[key].(string)
	return value
}

func normalizeMap(data any) map[string]any {
	if data == nil {
		return nil
	}
	switch value := data.(type) {
	case map[string]any:
		return value
	default:
		return nil
	}
}

func cloneMap(data map[string]any) map[string]any {
	if len(data) == 0 {
		return nil
	}
	cp := make(map[string]any, len(data))
	for k, v := range data {
		cp[k] = v
	}
	return cp
}

func blockID(prefix string, index int) string {
	return "block_" + prefix + "_" + uuid.NewString()[:8]
}

func truncate(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func (s *State) currentContent() *ai.AIRunContent {
	if strings.TrimSpace(s.currentContentID) == "" {
		return nil
	}
	for i := range s.Contents {
		if s.Contents[i] != nil && s.Contents[i].ID == s.currentContentID {
			return s.Contents[i]
		}
	}
	return nil
}

func (s State) CurrentContentID() string {
	return strings.TrimSpace(s.currentContentID)
}
