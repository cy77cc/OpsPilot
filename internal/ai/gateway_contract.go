// Package ai 定义 AI 编排层的网关契约。
//
// 本文件定义了 HTTP Gateway 与 AI Orchestrator 之间的请求/响应结构。
// Gateway 只负责 transport/auth/session shell，不涉及 AI 编排语义。
package ai

import "github.com/cy77cc/OpsPilot/internal/ai/events"

// RunRequest 是启动 AI 编排流水线的请求结构。
//
// 字段说明:
//   - SessionID: 会话 ID，为空时自动生成新会话
//   - Message: 用户消息内容，必填
//   - RuntimeContext: 运行时上下文，包含场景、路由、资源等信息
type RunRequest struct {
	SessionID      string         `json:"session_id,omitempty"` // 会话 ID，为空则新建
	Message        string         `json:"message"`              // 用户消息，必填
	RuntimeContext RuntimeContext `json:"runtime_context,omitempty"` // 运行时上下文
}

// ResumeRequest 是恢复等待审批执行的请求结构。
//
// 当执行计划中某个步骤需要用户审批时，执行会暂停等待。
// 用户确认后使用此请求恢复执行。
//
// 字段说明:
//   - SessionID: 会话 ID
//   - PlanID: 计划 ID
//   - StepID/Target: 待审批步骤的标识
//   - Approved: 是否批准
//   - Reason: 拒绝原因 (拒绝时填写)
type ResumeRequest struct {
	SessionID string `json:"session_id,omitempty"` // 会话 ID
	PlanID    string `json:"plan_id,omitempty"`    // 计划 ID
	StepID    string `json:"step_id,omitempty"`    // 步骤 ID
	Target    string `json:"target,omitempty"`     // 目标 (StepID 的别名)
	Approved  bool   `json:"approved"`             // 是否批准
	Reason    string `json:"reason,omitempty"`     // 拒绝原因
}

// RuntimeContext 包含请求的运行时上下文信息。
//
// 这些信息由前端提供，用于帮助 AI 更好地理解用户意图和上下文。
type RuntimeContext struct {
	Scene             string             `json:"scene,omitempty"`             // 场景标识 (如: host, cluster, service)
	Route             string             `json:"route,omitempty"`             // 当前路由
	ProjectID         string             `json:"project_id,omitempty"`        // 项目 ID
	CurrentPage       string             `json:"current_page,omitempty"`      // 当前页面
	SelectedResources []SelectedResource `json:"selected_resources,omitempty"` // 用户选中的资源
	UserContext       map[string]any     `json:"user_context,omitempty"`      // 用户上下文
	Metadata          map[string]any     `json:"metadata,omitempty"`          // 扩展元数据
}

// SelectedResource 表示用户选中的资源。
type SelectedResource struct {
	Type string `json:"type"`          // 资源类型 (如: host, service, deployment)
	ID   string `json:"id,omitempty"`  // 资源 ID
	Name string `json:"name,omitempty"` // 资源名称
}

// StreamEvent 是流式事件结构。
//
// 通过 SSE 推送给前端，用于实时展示 AI 执行状态。
//
// 字段说明:
//   - Type: 事件类型 (如: meta, delta, tool_call, done, error)
//   - Audience: 目标受众 (用户或系统)
//   - Meta: 事件元数据 (sessionID, traceID 等)
//   - Data: 事件载荷
type StreamEvent struct {
	Type     events.Name      `json:"type"`              // 事件类型
	Audience events.Audience  `json:"audience"`          // 目标受众
	Meta     events.EventMeta `json:"meta"`              // 事件元数据
	Data     map[string]any   `json:"data,omitempty"`    // 事件载荷
}

// ResumeResult 是恢复执行的响应结构。
//
// 字段说明:
//   - Resumed: 是否成功恢复
//   - Interrupted: 是否被中断 (拒绝审批)
//   - Status: 恢复状态 (approved/rejected/idempotent/noop/missing)
//   - Message: 状态消息
type ResumeResult struct {
	Resumed     bool   `json:"resumed"`               // 是否成功恢复
	Interrupted bool   `json:"interrupted,omitempty"` // 是否被中断
	SessionID   string `json:"session_id,omitempty"`  // 会话 ID
	PlanID      string `json:"plan_id,omitempty"`     // 计划 ID
	StepID      string `json:"step_id,omitempty"`     // 步骤 ID
	Status      string `json:"status,omitempty"`      // 状态码
	Message     string `json:"message,omitempty"`     // 状态消息
}
