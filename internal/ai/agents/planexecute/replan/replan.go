// Package replan 封装 plan-execute 架构中的重规划子 Agent。
//
// NewReplanner 创建 Replanner Agent，其职责是在 Executor 完成部分步骤后，
// 根据已有执行结果判断是否需要调整剩余计划。
// 相比 Planner 使用更高 Temp（0.5）以允许更灵活的推理。
package replan

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
)

// NewReplanner 创建重规划 Agent 实例。
// processor 为 nil 时返回空输入，ADK 框架会使用默认 Replanner 行为。
func NewReplanner(ctx context.Context) (adk.Agent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.5,
	})
	if err != nil {
		return nil, err
	}
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model,
	})
}
