package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cy77cc/OpsPilot/internal/ai/events"
	"github.com/redis/go-redis/v9"
)

type EventType = events.Name

const (
	EventMeta             EventType = events.Meta
	EventDelta            EventType = events.Delta
	EventThinkingDelta    EventType = events.ThinkingDelta
	EventToolCall         EventType = events.ToolCall
	EventToolResult       EventType = events.ToolResult
	EventStageDelta       EventType = events.StageDelta
	EventStepUpdate       EventType = events.StepUpdate
	EventApprovalRequired EventType = events.ApprovalRequired
	EventTurnStarted      EventType = events.TurnStarted
	EventTurnState        EventType = events.TurnState
	EventDone             EventType = events.Done
	EventError            EventType = events.Error
)

type StreamEvent struct {
	Type EventType      `json:"type"`
	Data map[string]any `json:"data,omitempty"`
}

type StreamEmitter func(StreamEvent) bool

type Runtime interface {
	Run(ctx context.Context, req RunRequest, emit StreamEmitter) error
	Resume(ctx context.Context, req ResumeRequest) (*ResumeResult, error)
	ResumeStream(ctx context.Context, req ResumeRequest, emit StreamEmitter) (*ResumeResult, error)
}

type RunRequest struct {
	SessionID      string         `json:"session_id,omitempty"`
	Message        string         `json:"message"`
	RuntimeContext RuntimeContext `json:"runtime_context,omitempty"`
}

type RuntimeContext struct {
	Scene             string             `json:"scene,omitempty"`
	SceneName         string             `json:"scene_name,omitempty"`
	Route             string             `json:"route,omitempty"`
	ProjectID         string             `json:"project_id,omitempty"`
	ProjectName       string             `json:"project_name,omitempty"`
	CurrentPage       string             `json:"current_page,omitempty"`
	SelectedResources []SelectedResource `json:"selected_resources,omitempty"`
	UserContext       map[string]any     `json:"user_context,omitempty"`
	Metadata          map[string]any     `json:"metadata,omitempty"`
}

type SelectedResource struct {
	Type      string `json:"type,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type ResumeRequest struct {
	SessionID    string `json:"session_id,omitempty"`
	PlanID       string `json:"plan_id,omitempty"`
	StepID       string `json:"step_id,omitempty"`
	Target       string `json:"target,omitempty"`
	CheckpointID string `json:"checkpoint_id,omitempty"`
	Approved     bool   `json:"approved"`
	Reason       string `json:"reason,omitempty"`
}

type ResumeResult struct {
	Resumed     bool   `json:"resumed"`
	Interrupted bool   `json:"interrupted"`
	SessionID   string `json:"session_id,omitempty"`
	PlanID      string `json:"plan_id,omitempty"`
	StepID      string `json:"step_id,omitempty"`
	TurnID      string `json:"turn_id,omitempty"`
	Message     string `json:"message,omitempty"`
	Status      string `json:"status,omitempty"`
}

type SceneConfig struct {
	Name           string               `json:"name"`
	Description    string               `json:"description"`
	Constraints    []string             `json:"constraints,omitempty"`
	AllowedTools   []string             `json:"allowed_tools,omitempty"`
	BlockedTools   []string             `json:"blocked_tools,omitempty"`
	Examples       []string             `json:"examples,omitempty"`
	ApprovalConfig *SceneApprovalConfig `json:"approval_config,omitempty"`
}

type SceneApprovalConfig struct {
	DefaultPolicy       ApprovalPolicy                  `json:"default_policy"`
	ToolOverrides       map[string]ToolApprovalOverride `json:"tool_overrides,omitempty"`
	EnvironmentPolicies map[string]ApprovalPolicy       `json:"environment_policies,omitempty"`
}

type ApprovalPolicy struct {
	RequireApprovalFor    []string        `json:"require_approval_for,omitempty"`
	RequireForAllMutating bool            `json:"require_for_all_mutating,omitempty"`
	SkipConditions        []SkipCondition `json:"skip_conditions,omitempty"`
}

type ToolApprovalOverride struct {
	ForceApproval   bool   `json:"force_approval,omitempty"`
	SkipApproval    bool   `json:"skip_approval,omitempty"`
	SummaryTemplate string `json:"summary_template,omitempty"`
}

type SkipCondition struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern"`
}

type SubSceneRule struct {
	IncludeTools     []string `json:"include_tools,omitempty"`
	ExcludeTools     []string `json:"exclude_tools,omitempty"`
	ExtraConstraints []string `json:"extra_constraints,omitempty"`
}

type ResolvedScene struct {
	SceneKey     string      `json:"scene_key"`
	Domain       string      `json:"domain"`
	SubScene     string      `json:"sub_scene,omitempty"`
	SceneConfig  SceneConfig `json:"scene_config"`
	AllowedTools []string    `json:"allowed_tools,omitempty"`
	BlockedTools []string    `json:"blocked_tools,omitempty"`
	Constraints  []string    `json:"constraints,omitempty"`
	ExampleIDs   []string    `json:"example_ids,omitempty"`
}

func (r ResolvedScene) EffectiveAllowedTools() []string {
	if len(r.AllowedTools) > 0 {
		return cloneStrings(r.AllowedTools)
	}
	return cloneStrings(r.SceneConfig.AllowedTools)
}

type ExecutionStatus string

const (
	ExecutionStatusRunning         ExecutionStatus = "running"
	ExecutionStatusWaitingApproval ExecutionStatus = "waiting_approval"
	ExecutionStatusCompleted       ExecutionStatus = "completed"
	ExecutionStatusRejected        ExecutionStatus = "rejected"
	ExecutionStatusFailed          ExecutionStatus = "failed"
)

type StepStatus string

const (
	StepPending         StepStatus = "pending"
	StepRunning         StepStatus = "running"
	StepSucceeded       StepStatus = "success"
	StepFailed          StepStatus = "error"
	StepWaitingApproval StepStatus = "waiting_approval"
	StepRejected        StepStatus = "aborted"
)

type StepState struct {
	StepID             string         `json:"step_id"`
	Title              string         `json:"title,omitempty"`
	Expert             string         `json:"expert,omitempty"`
	Status             StepStatus     `json:"status,omitempty"`
	Mode               string         `json:"mode,omitempty"`
	Risk               string         `json:"risk,omitempty"`
	UserVisibleSummary string         `json:"user_visible_summary,omitempty"`
	ToolName           string         `json:"tool_name,omitempty"`
	ToolArgs           map[string]any `json:"tool_args,omitempty"`
}

type PendingApproval struct {
	ID          string         `json:"id,omitempty"`
	PlanID      string         `json:"plan_id,omitempty"`
	StepID      string         `json:"step_id,omitempty"`
	Status      string         `json:"status,omitempty"`
	Title       string         `json:"title,omitempty"`
	Mode        string         `json:"mode,omitempty"`
	Risk        string         `json:"risk,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	ApprovalKey string         `json:"approval_key,omitempty"`
	ToolName    string         `json:"tool_name,omitempty"`
	Params      map[string]any `json:"params,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
	ExpiresAt   time.Time      `json:"expires_at,omitempty"`
}

type ExecutionState struct {
	TraceID         string               `json:"trace_id,omitempty"`
	SessionID       string               `json:"session_id,omitempty"`
	PlanID          string               `json:"plan_id,omitempty"`
	TurnID          string               `json:"turn_id,omitempty"`
	Message         string               `json:"message,omitempty"`
	Scene           string               `json:"scene,omitempty"`
	Status          ExecutionStatus      `json:"status,omitempty"`
	Phase           string               `json:"phase,omitempty"`
	RuntimeContext  RuntimeContext       `json:"runtime_context,omitempty"`
	CheckpointID    string               `json:"checkpoint_id,omitempty"`
	InterruptTarget string               `json:"interrupt_target,omitempty"`
	Steps           map[string]StepState `json:"steps,omitempty"`
	PendingApproval *PendingApproval     `json:"pending_approval,omitempty"`
	Metadata        map[string]any       `json:"metadata,omitempty"`
	UpdatedAt       time.Time            `json:"updated_at,omitempty"`
}

type ExecutionStore struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration

	mu        sync.RWMutex
	data      map[string]ExecutionState
	bySession map[string]string
}

func NewExecutionStore(client redis.UniversalClient, prefix string) *ExecutionStore {
	return &ExecutionStore{
		client:    client,
		prefix:    ensurePrefix(prefix, "ai:execution:"),
		ttl:       24 * time.Hour,
		data:      make(map[string]ExecutionState),
		bySession: make(map[string]string),
	}
}

func (s *ExecutionStore) Save(ctx context.Context, state ExecutionState) error {
	state.UpdatedAt = time.Now().UTC()
	key := s.storageKey(state.SessionID, state.PlanID)
	if key == "" {
		return fmt.Errorf("execution state identity is empty")
	}
	if s.client != nil {
		payload, err := json.Marshal(state)
		if err != nil {
			return err
		}
		if err := s.client.Set(ctx, key, payload, s.ttl).Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.data[key] = state
	if state.SessionID != "" {
		s.bySession[state.SessionID] = key
	}
	s.mu.Unlock()
	return nil
}

func (s *ExecutionStore) Load(ctx context.Context, sessionID, planID string) (ExecutionState, bool, error) {
	key := s.storageKey(sessionID, planID)
	if key == "" {
		return ExecutionState{}, false, nil
	}
	if s.client != nil {
		raw, err := s.client.Get(ctx, key).Bytes()
		if err == nil {
			var state ExecutionState
			if err := json.Unmarshal(raw, &state); err != nil {
				return ExecutionState{}, false, err
			}
			s.mu.Lock()
			s.data[key] = state
			if state.SessionID != "" {
				s.bySession[state.SessionID] = key
			}
			s.mu.Unlock()
			return state, true, nil
		}
		if err != nil && err != redis.Nil {
			return ExecutionState{}, false, err
		}
	}
	s.mu.RLock()
	state, ok := s.data[key]
	s.mu.RUnlock()
	return state, ok, nil
}

func (s *ExecutionStore) LoadLatestBySession(ctx context.Context, sessionID string) (ExecutionState, bool, error) {
	s.mu.RLock()
	key := s.bySession[sessionID]
	s.mu.RUnlock()
	if key != "" {
		s.mu.RLock()
		state, ok := s.data[key]
		s.mu.RUnlock()
		if ok {
			return state, true, nil
		}
	}
	return s.Load(ctx, sessionID, "")
}

func (s *ExecutionStore) storageKey(sessionID, planID string) string {
	sessionID = strings.TrimSpace(sessionID)
	planID = strings.TrimSpace(planID)
	switch {
	case sessionID == "" && planID == "":
		return ""
	case sessionID != "" && planID != "":
		return s.prefix + sessionID + ":" + planID
	case sessionID != "":
		return s.prefix + sessionID
	default:
		return s.prefix + planID
	}
}

type CheckpointStore struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration

	mu          sync.RWMutex
	checkpoints map[string][]byte
	aliases     map[string]string
	targets     map[string]string
}

func NewCheckpointStore(client redis.UniversalClient, prefix string) *CheckpointStore {
	return &CheckpointStore{
		client:      client,
		prefix:      ensurePrefix(prefix, "ai:checkpoint:"),
		ttl:         24 * time.Hour,
		checkpoints: make(map[string][]byte),
		aliases:     make(map[string]string),
		targets:     make(map[string]string),
	}
}

func (s *CheckpointStore) Get(ctx context.Context, checkpointID string) ([]byte, bool, error) {
	checkpointID = strings.TrimSpace(checkpointID)
	if checkpointID == "" {
		return nil, false, nil
	}
	if s.client != nil {
		raw, err := s.client.Get(ctx, s.prefix+checkpointID).Bytes()
		if err == nil {
			return append([]byte(nil), raw...), true, nil
		}
		if err != nil && err != redis.Nil {
			return nil, false, err
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw, ok := s.checkpoints[checkpointID]
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), raw...), true, nil
}

func (s *CheckpointStore) Set(ctx context.Context, checkpointID string, checkpoint []byte) error {
	checkpointID = strings.TrimSpace(checkpointID)
	if checkpointID == "" {
		return fmt.Errorf("checkpoint id is empty")
	}
	copied := append([]byte(nil), checkpoint...)
	if s.client != nil {
		if err := s.client.Set(ctx, s.prefix+checkpointID, copied, s.ttl).Err(); err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.checkpoints[checkpointID] = copied
	s.mu.Unlock()
	return nil
}

func (s *CheckpointStore) BindIdentity(ctx context.Context, sessionID, planID, stepID, checkpointID, target string) error {
	identity := ResumeIdentity(sessionID, planID, stepID)
	if identity == "" || strings.TrimSpace(checkpointID) == "" {
		return nil
	}
	if s.client != nil {
		if err := s.client.Set(ctx, s.prefix+"alias:"+identity, checkpointID, s.ttl).Err(); err != nil {
			return err
		}
		if strings.TrimSpace(target) != "" {
			if err := s.client.Set(ctx, s.prefix+"target:"+identity, target, s.ttl).Err(); err != nil {
				return err
			}
		}
	}
	s.mu.Lock()
	s.aliases[identity] = checkpointID
	if strings.TrimSpace(target) != "" {
		s.targets[identity] = strings.TrimSpace(target)
	}
	s.mu.Unlock()
	return nil
}

func (s *CheckpointStore) Resolve(ctx context.Context, sessionID, planID, stepID, fallback string) (string, string, bool, error) {
	identity := ResumeIdentity(sessionID, planID, stepID)
	if identity == "" && strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback), strings.TrimSpace(stepID), true, nil
	}
	if identity == "" {
		return "", "", false, nil
	}
	if s.client != nil {
		checkpointID, err := s.client.Get(ctx, s.prefix+"alias:"+identity).Result()
		if err == nil {
			target, _ := s.client.Get(ctx, s.prefix+"target:"+identity).Result()
			if strings.TrimSpace(target) == "" {
				target = strings.TrimSpace(stepID)
			}
			return checkpointID, target, true, nil
		}
		if err != nil && err != redis.Nil {
			return "", "", false, err
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	checkpointID, ok := s.aliases[identity]
	if !ok {
		return "", "", false, nil
	}
	target := s.targets[identity]
	if target == "" {
		target = strings.TrimSpace(stepID)
	}
	return checkpointID, target, true, nil
}

func ResumeIdentity(sessionID, planID, stepID string) string {
	sessionID = strings.TrimSpace(sessionID)
	planID = strings.TrimSpace(planID)
	stepID = strings.TrimSpace(stepID)
	if sessionID == "" || planID == "" || stepID == "" {
		return ""
	}
	return sessionID + ":" + planID + ":" + stepID
}

func ensurePrefix(prefix, fallback string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return fallback
	}
	return prefix
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
