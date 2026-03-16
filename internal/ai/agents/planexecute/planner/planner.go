// Package planner 封装 plan-execute 架构中的规划子 Agent。
//
// NewPlanner 创建 Planner Agent，其职责是将用户输入分解为可执行的步骤计划。
// 通过 ContextProcessor.BuildPlannerInput 注入场景约束、工具清单等上下文信息。
package planner

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
)

// NewPlanner 创建规划 Agent 实例。
// processor 为 nil 时直接透传用户输入，不注入额外上下文。
func NewPlanner(ctx context.Context) (adk.Agent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.1,
	})
	if err != nil {
		return nil, err
	}

	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: model,
	})
}
