// Package planner 实现 AI 编排的规划阶段。
//
// 本文件定义规划阶段的所有核心类型。
package planner

import (
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/ai/availability"
	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
)

// DecisionType 定义决策类型。
type DecisionType string

const (
	DecisionClarify     DecisionType = "clarify"      // 需要用户澄清
	DecisionReject      DecisionType = "reject"       // 拒绝执行
	DecisionDirectReply DecisionType = "direct_reply" // 直接回复
	DecisionPlan        DecisionType = "plan"         // 生成执行计划
)

// Input 是规划器的输入结构。
type Input struct {
	Message string        // 用户原始消息
	Rewrite rewrite.Output // Rewrite 阶段的输出
}

// Decision 表示规划器的决策输出。
type Decision struct {
	Type       DecisionType     `json:"type"`                // 决策类型
	Message    string           `json:"message,omitempty"`   // 消息内容
	Reason     string           `json:"reason,omitempty"`    // 原因说明
	Candidates []map[string]any `json:"candidates,omitempty"` // 候选项 (澄清时使用)
	Plan       *ExecutionPlan   `json:"plan,omitempty"`      // 执行计划
	Narrative  string           `json:"narrative"`           // 自然语言描述
}

// PlanningError 表示规划错误。
type PlanningError struct {
	Code              string // 错误码
	UserVisibleReason string // 用户可见原因
	Cause             error  // 原始错误
}

// Error 实现 error 接口。
func (e *PlanningError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", strings.TrimSpace(e.Code), e.Cause)
	}
	return firstNonEmpty(e.Code, "planning_unavailable")
}

// Unwrap 返回原始错误。
func (e *PlanningError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// UserVisibleMessage 返回用户可见的错误消息。
func (e *PlanningError) UserVisibleMessage() string {
	if e == nil {
		return availability.UnavailableMessage(availability.LayerPlanner)
	}
	return firstNonEmpty(e.UserVisibleReason, availability.UnavailableMessage(availability.LayerPlanner))
}

// ExecutionPlan 表示执行计划。
type ExecutionPlan struct {
	PlanID    string            `json:"plan_id"`       // 计划唯一标识
	Goal      string            `json:"goal"`          // 执行目标
	Resolved  ResolvedResources `json:"resolved"`      // 已解析的资源
	Narrative string            `json:"narrative"`     // 自然语言描述
	Steps     []PlanStep        `json:"steps"`         // 执行步骤列表
}

// ResourceRef 表示资源引用。
type ResourceRef struct {
	ID   int    `json:"id,omitempty"`   // 资源 ID
	Name string `json:"name,omitempty"` // 资源名称
}

// PodRef 表示 Pod 引用。
type PodRef struct {
	Name      string `json:"name,omitempty"`      // Pod 名称
	Namespace string `json:"namespace,omitempty"` // 命名空间
	ClusterID int    `json:"cluster_id,omitempty"` // 集群 ID
}

// ResourceScope 表示资源范围。
type ResourceScope struct {
	Kind         string         `json:"kind,omitempty"`         // 范围类型 (all/filtered/single)
	ResourceType string         `json:"resource_type,omitempty"` // 资源类型
	Selector     map[string]any `json:"selector,omitempty"`     // 选择器
}

// ResolvedResources 表示已解析的资源引用。
// 所有资源统一用列表字段表示，parseResolvedResources 负责将 LLM 输出的平铺字段归并到列表中。
type ResolvedResources struct {
	Namespace string         `json:"namespace,omitempty"` // 命名空间
	Services  []ResourceRef  `json:"services,omitempty"`  // 服务引用列表
	Clusters  []ResourceRef  `json:"clusters,omitempty"`  // 集群引用列表
	Hosts     []ResourceRef  `json:"hosts,omitempty"`     // 主机引用列表
	Pods      []PodRef       `json:"pods,omitempty"`      // Pod 引用列表
	Scope     *ResourceScope `json:"scope,omitempty"`     // 资源范围
}

// PlanStep 表示执行计划中的单个步骤。
type PlanStep struct {
	StepID    string         `json:"step_id"`              // 步骤唯一标识
	Title     string         `json:"title"`                // 步骤标题
	Expert    string         `json:"expert"`               // 专家名称
	Intent    string         `json:"intent"`               // 意图
	Task      string         `json:"task"`                 // 任务描述
	Input     map[string]any `json:"input,omitempty"`      // 输入参数
	DependsOn []string       `json:"depends_on,omitempty"` // 依赖的步骤 ID
	Mode      string         `json:"mode"`                 // 操作模式 (readonly/mutating)
	Risk      string         `json:"risk"`                 // 风险等级 (low/medium/high)
	Narrative string         `json:"narrative,omitempty"`  // 自然语言描述
}
