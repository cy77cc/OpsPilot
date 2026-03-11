// Package model 提供数据库模型定义。
//
// 本文件定义 AI 审批任务相关的数据模型，用于 AI 主导的操作审批流程。
package model

import (
	"encoding/json"
	"time"
)

// ExecutionStep 表示执行步骤，用于描述审批任务中的具体操作。
type ExecutionStep struct {
	Title       string `json:"title"`                  // 步骤标题
	Description string `json:"description,omitempty"` // 步骤描述
}

// RiskAssessment 表示风险评估结果，用于量化审批操作的风险等级。
type RiskAssessment struct {
	Level   string   `json:"level"`             // 风险等级: critical/high/medium/low
	Summary string   `json:"summary,omitempty"` // 风险摘要
	Items   []string `json:"items,omitempty"`   // 风险项列表
}

// TaskDetail 表示任务详情，包含执行步骤和风险评估。
type TaskDetail struct {
	Summary        string          `json:"summary"`                 // 任务摘要
	Steps          []ExecutionStep `json:"steps,omitempty"`         // 执行步骤列表
	RiskAssessment RiskAssessment  `json:"risk_assessment"`         // 风险评估结果
	RollbackPlan   string          `json:"rollback_plan,omitempty"` // 回滚计划
}

// ApprovalToolCall 表示审批工具调用，记录 AI 请求执行的具体工具。
type ApprovalToolCall struct {
	Name      string         `json:"name"`               // 工具名称
	Arguments map[string]any `json:"arguments,omitempty"` // 工具参数
}

// AIApprovalTask 是 AI 审批任务表模型，存储可执行的审批工作项。
//
// 表名: ai_approval_tickets
// 关联: User (通过 request_user_id 和 approver_user_id)
//
// 状态流转:
//   - pending: 等待审批
//   - approved: 已批准，等待执行
//   - rejected: 已拒绝
//   - executed: 已执行完成
//   - expired: 已过期
type AIApprovalTask struct {
	ID                 string     `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`                                           // 审批任务唯一标识
	ConfirmationID     string     `gorm:"column:confirmation_id;type:varchar(64);index" json:"confirmation_id"`                      // 关联的确认请求 ID
	RequestUserID      uint64     `gorm:"column:request_user_id;index:idx_ai_approval_request_created" json:"request_user_id"`       // 请求用户 ID
	ApprovalToken      string     `gorm:"column:approval_token;type:varchar(128);uniqueIndex" json:"approval_token"`                 // 审批令牌 (用于邮件链接)
	ToolName           string     `gorm:"column:tool_name;type:varchar(128);index" json:"tool_name"`                                  // 工具名称
	TargetResourceType string     `gorm:"column:target_resource_type;type:varchar(64);index" json:"target_resource_type"`            // 目标资源类型 (如: host, deployment)
	TargetResourceID   string     `gorm:"column:target_resource_id;type:varchar(128);index" json:"target_resource_id"`               // 目标资源 ID
	TargetResourceName string     `gorm:"column:target_resource_name;type:varchar(128);index" json:"target_resource_name"`           // 目标资源名称
	RiskLevel          string     `gorm:"column:risk_level;type:varchar(16);index" json:"risk_level"`                                // 风险等级: critical/high/medium/low
	Status             string     `gorm:"column:status;type:varchar(32);index" json:"status"`                                        // 状态: pending/approved/rejected/executed/expired
	ApproverUserID     uint64     `gorm:"column:approver_user_id;index" json:"approver_user_id"`                                     // 审批人用户 ID
	RejectReason       string     `gorm:"column:reject_reason;type:varchar(255)" json:"reject_reason"`                               // 拒绝原因
	ParamsJSON         string     `gorm:"column:params_json;type:longtext" json:"params_json"`                                       // 工具参数 (JSON 格式)
	PreviewJSON        string     `gorm:"column:preview_json;type:longtext" json:"preview_json"`                                     // 预览数据 (JSON 格式)
	TaskDetailJSON     string     `gorm:"column:task_detail_json;type:longtext" json:"task_detail_json"`                             // 任务详情 (JSON 格式)
	ToolCallsJSON      string     `gorm:"column:tool_calls_json;type:longtext" json:"tool_calls_json"`                               // 工具调用列表 (JSON 格式)
	ExecutedAt         *time.Time `gorm:"column:executed_at" json:"executed_at,omitempty"`                                           // 执行时间
	ExpiresAt          time.Time  `gorm:"column:expires_at;index" json:"expires_at"`                                                 // 过期时间
	ApprovedAt         *time.Time `gorm:"column:approved_at" json:"approved_at,omitempty"`                                           // 批准时间
	RejectedAt         *time.Time `gorm:"column:rejected_at" json:"rejected_at,omitempty"`                                           // 拒绝时间
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_ai_approval_request_created" json:"created_at"`  // 创建时间
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                                        // 更新时间
}

// TableName 返回 AI 审批任务表名。
func (AIApprovalTask) TableName() string { return "ai_approval_tickets" }

// SetTaskDetail 设置任务详情 JSON。
//
// 参数:
//   - detail: 任务详情对象
//
// 返回: 序列化错误
func (m *AIApprovalTask) SetTaskDetail(detail TaskDetail) error {
	raw, err := json.Marshal(detail)
	if err != nil {
		return err
	}
	m.TaskDetailJSON = string(raw)
	return nil
}

// TaskDetail 解析任务详情 JSON。
//
// 返回: 任务详情对象和解析错误
func (m *AIApprovalTask) TaskDetail() (TaskDetail, error) {
	if m == nil || m.TaskDetailJSON == "" {
		return TaskDetail{}, nil
	}
	var detail TaskDetail
	err := json.Unmarshal([]byte(m.TaskDetailJSON), &detail)
	return detail, err
}

// SetToolCalls 设置工具调用 JSON。
//
// 参数:
//   - calls: 工具调用列表
//
// 返回: 序列化错误
func (m *AIApprovalTask) SetToolCalls(calls []ApprovalToolCall) error {
	raw, err := json.Marshal(calls)
	if err != nil {
		return err
	}
	m.ToolCallsJSON = string(raw)
	return nil
}

// ToolCalls 解析工具调用 JSON。
//
// 返回: 工具调用列表和解析错误
func (m *AIApprovalTask) ToolCalls() ([]ApprovalToolCall, error) {
	if m == nil || m.ToolCallsJSON == "" {
		return nil, nil
	}
	var calls []ApprovalToolCall
	err := json.Unmarshal([]byte(m.ToolCallsJSON), &calls)
	return calls, err
}

// AIApprovalTicket 是 AIApprovalTask 的别名，保持向后兼容。
type AIApprovalTicket = AIApprovalTask
