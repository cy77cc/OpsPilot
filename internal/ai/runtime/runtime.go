// Package runtime 定义 AI 运行时的核心类型和存储组件。
//
// 主要职责：
//   - 定义流式事件类型（StreamEvent/StreamEmitter）和 Runtime 接口
//   - 定义执行状态（ExecutionState）、步骤状态（StepState）、审批状态（PendingApproval）
//   - 提供 ExecutionStore（执行快照持久化）和 CheckpointStore（ADK 断点持久化）
//
// 两个 Store 均支持 Redis 主存储 + 内存缓存的双层架构，
// Redis 为 nil 时降级为纯内存模式（适用于单机或测试环境）。
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

// EventType 是 StreamEvent 的事件名类型，与 events.Name 完全兼容。
type EventType = events.Name

// 以下常量将 events 包的名称重新导出为 EventXxx 形式，供 runtime 包内使用。
// 主运行时路径只依赖 turn lifecycle、正文流和审批事件；旧 stage 事件仅做兼容保留。
const (
	EventMeta             EventType = events.Meta
	EventDelta            EventType = events.Delta
	EventThinkingDelta    EventType = events.ThinkingDelta
	EventToolCall         EventType = events.ToolCall
	EventToolResult       EventType = events.ToolResult
	EventTurnState        EventType = events.TurnState
	EventChainMeta        EventType = events.ChainMeta
	EventChainStarted     EventType = events.ChainStarted
	EventChainNodeOpen    EventType = events.ChainNodeOpen
	EventChainNodePatch   EventType = events.ChainNodePatch
	EventChainNodeReplace EventType = events.ChainNodeReplace
	EventChainNodeClose   EventType = events.ChainNodeClose
	EventChainPaused      EventType = events.ChainPaused
	EventChainResumed     EventType = events.ChainResumed
	EventChainCollapsed   EventType = events.ChainCollapsed
	EventChainCompleted   EventType = events.ChainCompleted
	EventChainError       EventType = events.ChainError
	EventFinalAnswerStart EventType = events.FinalAnswerStart
	EventFinalAnswerDelta EventType = events.FinalAnswerDelta
	EventFinalAnswerDone  EventType = events.FinalAnswerDone
	EventDone             EventType = events.Done
	EventError            EventType = events.Error
)

// StreamEvent 是推送给前端的 SSE 事件单元。
type StreamEvent struct {
	Type EventType      `json:"type"`
	Data map[string]any `json:"data,omitempty"`
}

// StreamEmitter 是事件推送回调。返回 false 表示调用方已断开连接，应停止推送。
type StreamEmitter func(StreamEvent) bool

// Runtime 是 AI 运行时的顶层接口，由 Orchestrator 实现。
type Runtime interface {
	// Run 发起一次新的 AI 对话，通过 emit 流式推送执行事件。
	Run(ctx context.Context, req RunRequest, emit StreamEmitter) error
	// Resume 处理审批结果（非流式）。
	Resume(ctx context.Context, req ResumeRequest) (*ResumeResult, error)
	// ResumeStream 处理审批结果，并通过 emit 流式推送后续执行事件。
	ResumeStream(ctx context.Context, req ResumeRequest, emit StreamEmitter) (*ResumeResult, error)
}

// RunRequest 是发起新对话的请求参数。
type RunRequest struct {
	SessionID      string         `json:"session_id,omitempty"`      // 可选；为空时自动生成
	Message        string         `json:"message"`                   // 用户消息，不可为空
	RuntimeContext RuntimeContext `json:"runtime_context,omitempty"` // 场景上下文
}

// RuntimeContext 携带前端传入的场景信息，用于工具过滤、审批策略和 Prompt 注入。
type RuntimeContext struct {
	Scene             string             `json:"scene,omitempty"`              // 场景 key（如 k8s、host）
	SceneName         string             `json:"scene_name,omitempty"`         // 场景展示名
	Route             string             `json:"route,omitempty"`              // 当前路由路径
	ProjectID         string             `json:"project_id,omitempty"`         // 项目 ID
	ProjectName       string             `json:"project_name,omitempty"`       // 项目名称
	CurrentPage       string             `json:"current_page,omitempty"`       // 当前页面标识
	SelectedResources []SelectedResource `json:"selected_resources,omitempty"` // 用户选中的资源
	UserContext       map[string]any     `json:"user_context,omitempty"`       // 扩展用户上下文
	Metadata          map[string]any     `json:"metadata,omitempty"`           // 其他元信息
}

// SelectedResource 表示前端用户选中的单个资源。
type SelectedResource struct {
	Type      string `json:"type,omitempty"`      // 资源类型（pod/service/cluster 等）
	ID        string `json:"id,omitempty"`        // 资源 ID
	Name      string `json:"name,omitempty"`      // 资源名称
	Namespace string `json:"namespace,omitempty"` // 命名空间（K8s 资源适用）
}

// ResumeRequest 是审批回调的请求参数。
type ResumeRequest struct {
	SessionID       string `json:"session_id,omitempty"`       // 会话 ID
	PlanID          string `json:"plan_id,omitempty"`          // 执行计划 ID
	StepID          string `json:"step_id,omitempty"`          // 待审批步骤 ID
	Target          string `json:"target,omitempty"`           // ADK 中断目标节点名
	CheckpointID    string `json:"checkpoint_id,omitempty"`    // ADK 断点 ID
	Approved        bool   `json:"approved"`                   // true=通过，false=拒绝
	Reason          string `json:"reason,omitempty"`           // 审批意见（可选）
	EditedArguments string `json:"edited_arguments,omitempty"` // 修改后的参数 JSON（可选）
}

// ResumeResult 是审批处理后的返回结果。
type ResumeResult struct {
	Resumed     bool   `json:"resumed"`     // true=已从断点继续执行
	Interrupted bool   `json:"interrupted"` // true=执行再次被中断
	SessionID   string `json:"session_id,omitempty"`
	PlanID      string `json:"plan_id,omitempty"`
	StepID      string `json:"step_id,omitempty"`
	TurnID      string `json:"turn_id,omitempty"`
	Message     string `json:"message,omitempty"` // 人类可读的结果摘要
	Status      string `json:"status,omitempty"`  // 执行最终状态
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

// ExecutionStatus 表示整次执行的生命周期状态。
type ExecutionStatus string

const (
	ExecutionStatusRunning         ExecutionStatus = "running"          // 执行中
	ExecutionStatusWaitingApproval ExecutionStatus = "waiting_approval" // 等待人工审批
	ExecutionStatusCompleted       ExecutionStatus = "completed"        // 执行完成
	ExecutionStatusRejected        ExecutionStatus = "rejected"         // 审批被拒绝
	ExecutionStatusFailed          ExecutionStatus = "failed"           // 执行失败
)

// StepStatus 表示单个步骤的执行状态。
type StepStatus string

const (
	StepPending         StepStatus = "pending"          // 等待执行
	StepRunning         StepStatus = "running"          // 执行中
	StepSucceeded       StepStatus = "success"          // 执行成功
	StepFailed          StepStatus = "error"            // 执行失败
	StepWaitingApproval StepStatus = "waiting_approval" // 等待审批
	StepRejected        StepStatus = "aborted"          // 审批被拒绝，已中止
)

// StepState 记录执行计划中单个步骤的实时状态，随执行过程更新并持久化。
type StepState struct {
	StepID             string         `json:"step_id"`
	Title              string         `json:"title,omitempty"`
	Expert             string         `json:"expert,omitempty"` // 负责该步骤的专家领域
	Status             StepStatus     `json:"status,omitempty"`
	Mode               string         `json:"mode,omitempty"`                 // readonly / mutating
	Risk               string         `json:"risk,omitempty"`                 // low / medium / high
	UserVisibleSummary string         `json:"user_visible_summary,omitempty"` // 展示给用户的摘要
	ToolName           string         `json:"tool_name,omitempty"`            // 触发审批的工具名
	ToolArgs           map[string]any `json:"tool_args,omitempty"`            // 工具调用参数
}

// PendingApproval 记录一次等待人工确认的审批请求。
// 由 orchestrator 在检测到中断事件时创建，恢复执行后更新状态。
type PendingApproval struct {
	ID              string         `json:"id,omitempty"`
	PlanID          string         `json:"plan_id,omitempty"`
	StepID          string         `json:"step_id,omitempty"`
	CheckpointID    string         `json:"checkpoint_id,omitempty"`
	Target          string         `json:"target,omitempty"`
	Status          string         `json:"status,omitempty"` // pending / approved / rejected
	Title           string         `json:"title,omitempty"`
	Mode            string         `json:"mode,omitempty"`
	Risk            string         `json:"risk,omitempty"`
	Summary         string         `json:"summary,omitempty"`      // 人类可读的操作摘要
	ApprovalKey     string         `json:"approval_key,omitempty"` // 用于幂等恢复的复合键
	ToolName        string         `json:"tool_name,omitempty"`
	ToolDisplayName string         `json:"tool_display_name,omitempty"`
	Params          map[string]any `json:"params,omitempty"`
	ArgumentsInJSON string         `json:"arguments_json,omitempty"`
	CreatedAt       time.Time      `json:"created_at,omitempty"`
	ExpiresAt       time.Time      `json:"expires_at,omitempty"` // 审批超时时间
}

// ExecutionState 是单次 AI 执行的完整状态快照，持久化到 ExecutionStore。
// 每次 Run/Resume 时创建或更新，用于支持会话恢复和状态查询。
type ExecutionState struct {
	TraceID         string               `json:"trace_id,omitempty"`
	SessionID       string               `json:"session_id,omitempty"`
	PlanID          string               `json:"plan_id,omitempty"`
	TurnID          string               `json:"turn_id,omitempty"`
	Message         string               `json:"message,omitempty"` // 用户原始消息
	Scene           string               `json:"scene,omitempty"`   // 场景 key
	Status          ExecutionStatus      `json:"status,omitempty"`
	Phase           string               `json:"phase,omitempty"` // 运行时生命周期阶段：running / waiting_approval / completed
	RuntimeContext  RuntimeContext       `json:"runtime_context,omitempty"`
	CheckpointID    string               `json:"checkpoint_id,omitempty"`    // 当前关联的 ADK 断点 ID
	InterruptTarget string               `json:"interrupt_target,omitempty"` // 最近一次中断的步骤 ID
	Steps           map[string]StepState `json:"steps,omitempty"`
	PendingApproval *PendingApproval     `json:"pending_approval,omitempty"`
	Metadata        map[string]any       `json:"metadata,omitempty"`
	UpdatedAt       time.Time            `json:"updated_at,omitempty"`
}

// ExecutionStore 持久化 ExecutionState，以 SessionID+PlanID 为 key。
// 支持 Redis（TTL=24h）+ 内存双层缓存，Redis 为 nil 时退化为纯内存。
// bySession 维护 SessionID → 最新 key 的索引，支持按会话查最新状态。
type ExecutionStore struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration

	mu        sync.RWMutex
	data      map[string]ExecutionState // key → state 内存缓存
	bySession map[string]string         // sessionID → 最新 key 索引
}

// NewExecutionStore 创建 ExecutionStore。
func NewExecutionStore(client redis.UniversalClient, prefix string) *ExecutionStore {
	return &ExecutionStore{
		client:    client,
		prefix:    ensurePrefix(prefix, "ai:execution:"),
		ttl:       24 * time.Hour,
		data:      make(map[string]ExecutionState),
		bySession: make(map[string]string),
	}
}

// Save 持久化执行状态，同时更新内存缓存和 bySession 索引。
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

// Load 按 SessionID+PlanID 精确加载执行状态，优先从 Redis 读取。
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
		} else {
			if err != redis.Nil {
				return ExecutionState{}, false, err
			}
		}
	}
	s.mu.RLock()
	state, ok := s.data[key]
	s.mu.RUnlock()
	return state, ok, nil
}

// LoadLatestBySession 按 SessionID 加载最近一次执行状态，优先使用内存索引。
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

// CheckpointStore 持久化 ADK Agent 的执行断点（Checkpoint），支持审批后续跑。
//
// BindIdentity 将 SessionID+PlanID+StepID 三元组映射到具体断点 ID，
// Resolve 在恢复时反查该映射，找到对应断点后通过 ADK ResumeWithParams 继续执行。
type CheckpointStore struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration

	mu          sync.RWMutex
	checkpoints map[string][]byte // checkpointID → 序列化断点数据
	aliases     map[string]string // resumeIdentity → checkpointID
	targets     map[string]string // resumeIdentity → ADK 中断目标节点名
}

// NewCheckpointStore 创建 CheckpointStore。
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

// Get 按 checkpointID 读取断点数据，优先从 Redis 读取。
func (s *CheckpointStore) Get(ctx context.Context, checkpointID string) ([]byte, bool, error) {
	checkpointID = strings.TrimSpace(checkpointID)
	if checkpointID == "" {
		return nil, false, nil
	}
	if s.client != nil {
		raw, err := s.client.Get(ctx, s.prefix+checkpointID).Bytes()
		if err == nil {
			return append([]byte(nil), raw...), true, nil
		} else {
			if err != redis.Nil {
				return nil, false, err
			}
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

// Set 保存断点数据到 Redis 和内存缓存。
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

// BindIdentity 将三元组（sessionID+planID+stepID）映射到断点 ID 和 ADK 中断目标节点。
// 审批中断时调用，在 Resolve 时反查以定位需要恢复的断点。
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

// Resolve 查找指定步骤对应的断点 ID 和目标节点名。
// 返回值：(checkpointID, target, found, error)
// 若找不到 identity 但有 fallback，则直接使用 fallback checkpointID。
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
		} else {
			if err != redis.Nil {
				return "", "", false, err
			}
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

// ResumeIdentity 生成用于断点索引的复合键（sessionID:planID:stepID）。
// 任一字段为空时返回空字符串（表示无法构成合法 identity）。
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
