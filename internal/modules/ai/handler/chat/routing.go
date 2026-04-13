package chathandler

import (
	"context"
	"fmt"
)

// RouteMode 路由模式。
type RouteMode string

const (
	ModeConversation RouteMode = "conversation"
	ModeTask         RouteMode = "task"
)

// TaskAction 任务动作。
type TaskAction string

const (
	TaskActionCreateTask TaskAction = "create_task"
	TaskActionNone       TaskAction = ""
)

// RunAction Run 动作。
type RunAction string

const (
	RunActionCreateRun RunAction = "create_run"
	RunActionNone      RunAction = ""
)

// ExecutionShape 执行形状。
type ExecutionShape string

const (
	ExecutionShapeSingleAgent         ExecutionShape = "single_agent"
	ExecutionShapeDelegatedSpecialist ExecutionShape = "delegated_specialist"
)

// Domain 领域。
type Domain string

const (
	DomainGeneral Domain = "general"
)

// ToolStrategy 工具策略。
type ToolStrategy string

const (
	ToolStrategyDirect ToolStrategy = "direct"
)

// ContextPlan 上下文计划。
type ContextPlan struct {
	ToolStrategy ToolStrategy `json:"tool_strategy"`
}

// RouteDecision 路由决策结果。
type RouteDecision struct {
	Mode           RouteMode      `json:"mode"`
	TaskAction     TaskAction     `json:"task_action"`
	RunAction      RunAction      `json:"run_action"`
	ExecutionShape ExecutionShape `json:"execution_shape"`
	Domain         Domain         `json:"domain"`
	ContextPlan    ContextPlan    `json:"context_plan"`
}

// Validate 验证路由决策是否有效。
func (d RouteDecision) Validate() error {
	if d.Mode == ModeConversation {
		if d.TaskAction != TaskActionNone || d.RunAction != RunActionNone {
			return fmt.Errorf("conversation route cannot schedule task or run actions")
		}
	}
	return nil
}

// RouteInput 路由决策的输入。
type RouteInput struct {
	Message string `json:"message"`
}

// RouteService 定义路由决策服务接口。
type RouteService interface {
	Decide(ctx context.Context, input RouteInput) (RouteDecision, error)
}

// routeService 是默认的路由服务实现（目前仅支持 defer 到 conversation）。
type routeService struct{}

// NewRouteService 创建默认路由服务。
func NewRouteService() RouteService {
	return &routeService{}
}

func (r *routeService) Decide(_ context.Context, _ RouteInput) (RouteDecision, error) {
	// 默认路由到 conversation（deferred）
	return RouteDecision{
		Mode:           ModeConversation,
		TaskAction:     TaskActionNone,
		RunAction:      RunActionNone,
		ExecutionShape: ExecutionShapeSingleAgent,
		Domain:         DomainGeneral,
		ContextPlan: ContextPlan{
			ToolStrategy: ToolStrategyDirect,
		},
	}, nil
}
