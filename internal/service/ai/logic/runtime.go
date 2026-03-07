// Package logic provides business logic for AI service
package logic

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"gorm.io/gorm"
)

// RecommendationRecord represents a recommendation for the user
type RecommendationRecord struct {
	ID             string    `json:"id"`
	UserID         uint64    `json:"userId"`
	Scene          string    `json:"scene"`
	Type           string    `json:"type"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	FollowupPrompt string    `json:"followup_prompt,omitempty"`
	Reasoning      string    `json:"reasoning,omitempty"`
	Relevance      float64   `json:"relevance"`
	CreatedAt      time.Time `json:"createdAt"`
}

// ApprovalTicket represents an approval request
type ApprovalTicket struct {
	ID         string         `json:"id"`
	Tool       string         `json:"tool"`
	Params     map[string]any `json:"params"`
	Risk       tools.ToolRisk `json:"risk"`
	Mode       tools.ToolMode `json:"mode"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"createdAt"`
	ExpiresAt  time.Time      `json:"expiresAt"`
	RequestUID uint64         `json:"requestUid"`
	ReviewUID  uint64         `json:"reviewUid,omitempty"`
	Meta       tools.ToolMeta `json:"-"`
}

// ExecutionRecord represents a tool execution record
type ExecutionRecord struct {
	ID         string            `json:"id"`
	Tool       string            `json:"tool"`
	Params     map[string]any    `json:"params"`
	Mode       tools.ToolMode    `json:"mode"`
	Status     string            `json:"status"`
	Result     *tools.ToolResult `json:"result,omitempty"`
	ApprovalID string            `json:"approvalId,omitempty"`
	RequestUID uint64            `json:"requestUid"`
	CreatedAt  time.Time         `json:"createdAt"`
	FinishedAt *time.Time        `json:"finishedAt,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// RuntimeStore holds runtime state for AI service
type RuntimeStore struct {
	mu                sync.RWMutex
	db                *gorm.DB
	approvals         map[string]*ApprovalTicket
	executions        map[string]*ExecutionRecord
	recommendations   map[string][]RecommendationRecord
	toolParams        map[string]map[string]any
	referencedContext map[string]map[string]any
}

// NewRuntimeStore creates a new RuntimeStore
func NewRuntimeStore(db *gorm.DB) *RuntimeStore {
	return &RuntimeStore{
		db:                db,
		approvals:         map[string]*ApprovalTicket{},
		executions:        map[string]*ExecutionRecord{},
		recommendations:   map[string][]RecommendationRecord{},
		toolParams:        map[string]map[string]any{},
		referencedContext: map[string]map[string]any{},
	}
}

func (s *RuntimeStore) dbEnabled() bool { return s != nil && s.db != nil }

// SetRecommendations stores recommendations for a user/scene
func (s *RuntimeStore) SetRecommendations(userID uint64, scene string, recs []RecommendationRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := recommendationKey(userID, scene)
	s.recommendations[key] = recs
}

// GetRecommendations retrieves recommendations for a user/scene
func (s *RuntimeStore) GetRecommendations(userID uint64, scene string, limit int) []RecommendationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := recommendationKey(userID, scene)
	list := s.recommendations[key]
	if len(list) == 0 {
		return nil
	}
	cp := make([]RecommendationRecord, 0, len(list))
	cp = append(cp, list...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].CreatedAt.After(cp[j].CreatedAt) })
	if limit > 0 && len(cp) > limit {
		cp = cp[:limit]
	}
	return cp
}

// RememberContext stores context for a user/scene
func (s *RuntimeStore) RememberContext(userID uint64, scene string, ctx map[string]any) {
	if s == nil || len(ctx) == 0 {
		return
	}
	key := referencedContextKey(userID, scene)
	s.mu.Lock()
	if s.referencedContext[key] == nil {
		s.referencedContext[key] = map[string]any{}
	}
	for k, v := range ctx {
		if strings.TrimSpace(k) == "" || v == nil {
			continue
		}
		if strings.TrimSpace(ToString(v)) == "" {
			continue
		}
		s.referencedContext[key][k] = v
	}
	s.mu.Unlock()
}

// GetRememberedContext retrieves stored context for a user/scene
func (s *RuntimeStore) GetRememberedContext(userID uint64, scene string) map[string]any {
	if s == nil {
		return map[string]any{}
	}
	key := referencedContextKey(userID, scene)
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw := s.referencedContext[key]
	out := map[string]any{}
	for k, v := range raw {
		out[k] = v
	}
	return out
}

// NewApproval creates a new approval ticket
func (s *RuntimeStore) NewApproval(uid uint64, metaTool ApprovalTicket) *ApprovalTicket {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := metaTool
	t.ID = fmt.Sprintf("apv-%d", time.Now().UnixNano())
	t.CreatedAt = time.Now()
	t.ExpiresAt = time.Now().Add(10 * time.Minute)
	t.Status = "pending"
	t.RequestUID = uid
	s.approvals[t.ID] = &t
	return &t
}

// GetApproval retrieves an approval ticket by ID
func (s *RuntimeStore) GetApproval(id string) (*ApprovalTicket, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.approvals[id]
	if !ok {
		return nil, false
	}
	cp := *t
	return &cp, true
}

// SetApprovalStatus updates approval status
func (s *RuntimeStore) SetApprovalStatus(id, status string, reviewUID uint64) (*ApprovalTicket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.approvals[id]
	if !ok {
		return nil, false
	}
	t.Status = status
	t.ReviewUID = reviewUID
	cp := *t
	return &cp, true
}

// SaveExecution saves an execution record
func (s *RuntimeStore) SaveExecution(rec *ExecutionRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executions[rec.ID] = rec
}

// GetExecution retrieves an execution record by ID
func (s *RuntimeStore) GetExecution(id string) (*ExecutionRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.executions[id]
	if !ok {
		return nil, false
	}
	cp := *rec
	return &cp, true
}

// ToolMemoryAccessor provides access to tool memory
type ToolMemoryAccessor struct {
	store *RuntimeStore
	uid   uint64
	scene string
}

// NewToolMemoryAccessor creates a new ToolMemoryAccessor
func NewToolMemoryAccessor(store *RuntimeStore, uid uint64, scene string) *ToolMemoryAccessor {
	return &ToolMemoryAccessor{store: store, uid: uid, scene: scene}
}

// GetLastToolParams retrieves last tool params
func (a *ToolMemoryAccessor) GetLastToolParams(toolName string) map[string]any {
	if a == nil || a.store == nil {
		return nil
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	v := a.store.toolParams[toolParamKey(a.uid, a.scene, toolName)]
	if len(v) == 0 {
		return nil
	}
	out := map[string]any{}
	for k, val := range v {
		out[k] = val
	}
	return out
}

// SetLastToolParams stores tool params
func (a *ToolMemoryAccessor) SetLastToolParams(toolName string, params map[string]any) {
	if a == nil || a.store == nil || strings.TrimSpace(toolName) == "" || len(params) == 0 {
		return
	}
	key := toolParamKey(a.uid, a.scene, toolName)
	cp := map[string]any{}
	for k, v := range params {
		cp[k] = v
	}
	a.store.mu.Lock()
	a.store.toolParams[key] = cp
	a.store.mu.Unlock()
}

func recommendationKey(userID uint64, scene string) string {
	return fmt.Sprintf("%d:%s", userID, NormalizeScene(scene))
}

func toolParamKey(userID uint64, scene, tool string) string {
	return fmt.Sprintf("%d:%s:%s", userID, NormalizeScene(scene), strings.TrimSpace(tool))
}

func referencedContextKey(userID uint64, scene string) string {
	return fmt.Sprintf("%d:%s", userID, NormalizeScene(scene))
}

// NormalizeScene normalizes scene name
func NormalizeScene(scene string) string {
	v := strings.TrimSpace(scene)
	if v == "" {
		return "global"
	}
	return v
}

// ToString converts any value to string
func ToString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
