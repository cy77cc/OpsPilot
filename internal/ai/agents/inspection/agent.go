// Package inspection 实现 K8s 集群定时巡检助手。
//
// 架构：ChatModelAgent + 只读工具集（K8s + 监控 + 服务目录）
//
// 该 Agent 由调度器触发（非用户主动请求），执行预定义的健康检查清单并输出结构化巡检报告。
// 不通过 IntentRouter 路由，直接由定时任务调用 Run() 方法。
//
// 与 Diagnosis Agent 的区别：
//   - Diagnosis：由用户触发，针对特定故障进行深度调查，使用 PlanExecute 动态规划步骤
//   - Inspection：由调度器触发，按固定清单全面扫描集群健康状态，使用 ChatModelAgent 执行固定 Prompt
package inspection

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/prompt"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/ai/tools"
)

// NewInspectionAgent 创建定时巡检 Agent 实例（ChatModelAgent 架构）。
//
// 巡检 Agent 挂载只读 K8s 工具、监控工具和服务目录工具，
// 通过固定的 INSPECTION_SYSTEM Prompt 驱动模型执行巡检清单。
//
// 最大迭代轮次为 10（巡检步骤固定，无需过多轮次）。
//
// 参数:
//   - ctx: 上下文（应携带 common.PlatformDeps 供工具使用）
//
// 返回: Agent 和初始化错误
func NewInspectionAgent(ctx context.Context) (adk.Agent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  120 * time.Second,
		Thinking: false,
		Temp:     0.0,
	})
	if err != nil {
		return nil, fmt.Errorf("inspection agent: init model: %w", err)
	}

	toolset := tools.NewInspectionTools(ctx)

	baseTool := make([]tool.BaseTool, 0, len(toolset))
	for _, t := range toolset {
		baseTool = append(baseTool, t)
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "InspectionAgent",
		Description: "Scheduled Kubernetes cluster health inspection assistant",
		Instruction: prompt.INSPECTION_SYSTEM,
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: baseTool,
			},
		},
		MaxIterations: 10,
	})
}
