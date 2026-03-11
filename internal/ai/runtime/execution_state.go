// Package runtime 提供 AI 编排的运行时状态管理。
//
// 本文件定义执行状态结构和存储，包括计划状态、步骤状态和审批状态。
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultExecutionTTL = 24 * time.Hour // 默认执行状态过期时间

// ExecutionStatus 定义执行状态类型。
type ExecutionStatus string

const (
	ExecutionStatusRunning         ExecutionStatus = "running"          // 执行中
	ExecutionStatusWaitingApproval ExecutionStatus = "waiting_approval" // 等待审批
	ExecutionStatusCompleted       ExecutionStatus = "completed"        // 已完成
	ExecutionStatusFailed          ExecutionStatus = "failed"           // 已失败
)

// StepStatus 定义步骤状态类型。
type StepStatus string

const (
	StepPending         StepStatus = "pending"          // 等待中
	StepReady           StepStatus = "ready"            // 就绪
	StepRunning         StepStatus = "running"          // 执行中
	StepWaitingApproval StepStatus = "waiting_approval" // 等待审批
	StepCompleted       StepStatus = "completed"        // 已完成
	StepFailed          StepStatus = "failed"           // 已失败
	StepBlocked         StepStatus = "blocked"          // 被阻塞
	StepCancelled       StepStatus = "cancelled"        // 已取消
)

// ContextSnapshot 表示运行时上下文快照。
type ContextSnapshot struct {
	Scene       string   `json:"scene,omitempty"`        // 场景
	Route       string   `json:"route,omitempty"`        // 路由
	ProjectID   string   `json:"project_id,omitempty"`   // 项目 ID
	CurrentPage string   `json:"current_page,omitempty"` // 当前页面
	ResourceIDs []string `json:"resource_ids,omitempty"` // 资源 ID 列表
}

// StepState 表示步骤状态。
type StepState struct {
	StepID             string         `json:"step_id"`                    // 步骤 ID
	Title              string         `json:"title,omitempty"`            // 标题
	Expert             string         `json:"expert,omitempty"`           // 专家名称
	Intent             string         `json:"intent,omitempty"`           // 意图
	Task               string         `json:"task,omitempty"`             // 任务
	Input              map[string]any `json:"input,omitempty"`            // 输入参数
	DependsOn          []string       `json:"depends_on,omitempty"`       // 依赖步骤
	Status             StepStatus     `json:"status"`                     // 状态
	Mode               string         `json:"mode,omitempty"`             // 操作模式
	Risk               string         `json:"risk,omitempty"`             // 风险等级
	Attempts           int            `json:"attempts,omitempty"`         // 尝试次数
	MaxAttempts        int            `json:"max_attempts,omitempty"`     // 最大尝试次数
	IdempotencyKey     string         `json:"idempotency_key,omitempty"`  // 幂等键
	ApprovalSatisfied  bool           `json:"approval_satisfied,omitempty"` // 审批是否已满足
	UserVisibleSummary string         `json:"user_visible_summary,omitempty"` // 用户可见摘要
	ErrorCode          string         `json:"error_code,omitempty"`       // 错误码
	ErrorMessage       string         `json:"error_message,omitempty"`    // 错误消息
	UpdatedAt          time.Time      `json:"updated_at"`                 // 更新时间
}

// PendingApproval 表示待审批状态。
type PendingApproval struct {
	PlanID       string    `json:"plan_id,omitempty"`       // 计划 ID
	StepID       string    `json:"step_id,omitempty"`       // 步骤 ID
	ApprovalKey  string    `json:"approval_key,omitempty"`  // 审批键
	Status       string    `json:"status,omitempty"`        // 状态
	Title        string    `json:"title,omitempty"`         // 标题
	Mode         string    `json:"mode,omitempty"`          // 操作模式
	Risk         string    `json:"risk,omitempty"`          // 风险等级
	Summary      string    `json:"summary,omitempty"`       // 摘要
	Approved     *bool     `json:"approved,omitempty"`      // 是否批准
	Reason       string    `json:"reason,omitempty"`        // 原因
	RequestedAt  time.Time `json:"requested_at,omitempty"`  // 请求时间
	ResolvedAt   time.Time `json:"resolved_at,omitempty"`   // 解决时间
	DecisionHash string    `json:"decision_hash,omitempty"` // 决策哈希
}

type ExecutionState struct {
	TraceID         string               `json:"trace_id"`
	SessionID       string               `json:"session_id"`
	PlanID          string               `json:"plan_id,omitempty"`
	Message         string               `json:"message,omitempty"`
	Status          ExecutionStatus      `json:"status"`
	Phase           string               `json:"phase,omitempty"`
	RuntimeContext  ContextSnapshot      `json:"runtime_context,omitempty"`
	Steps           map[string]StepState `json:"steps,omitempty"`
	PendingApproval *PendingApproval     `json:"pending_approval,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

type ExecutionStore struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewExecutionStore(client redis.UniversalClient, prefix string) *ExecutionStore {
	if prefix == "" {
		prefix = "ai:execution:"
	}
	return &ExecutionStore{client: client, prefix: prefix, ttl: defaultExecutionTTL}
}

func (s *ExecutionStore) Save(ctx context.Context, st ExecutionState) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("execution store is not initialized")
	}
	now := time.Now().UTC()
	if st.CreatedAt.IsZero() {
		st.CreatedAt = now
	}
	st.UpdatedAt = now
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("marshal execution state: %w", err)
	}
	return s.client.Set(ctx, s.key(st.SessionID), data, s.ttl).Err()
}

func (s *ExecutionStore) Load(ctx context.Context, sessionID string) (*ExecutionState, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("execution store is not initialized")
	}
	raw, err := s.client.Get(ctx, s.key(sessionID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var out ExecutionState
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal execution state: %w", err)
	}
	return &out, nil
}

func (s *ExecutionStore) key(sessionID string) string {
	return s.prefix + sessionID
}

func ApprovalDecisionHash(planID, stepID string, approved bool) string {
	return fmt.Sprintf("%s:%s:%t", planID, stepID, approved)
}
