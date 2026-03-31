package change

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/cicd"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/deployment"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/host"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/kubernetes"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/monitor"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/service"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/ai/common/middleware"
)

func New(ctx context.Context) (adk.Agent, error) {
	model, err := newModel(ctx)
	if err != nil {
		return nil, err
	}
	tools := newTools(ctx)
	handlers, err := middleware.BuildAgentHandlers(ctx, tools)
	if err != nil {
		return nil, err
	}
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "ChangeAgent",
		Description:   "Handles risky change workflows with approval.",
		Instruction:   agentPrompt,
		Model:         model,
		ToolsConfig:   adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools}},
		Handlers:      handlers,
		MaxIterations: 20,
	})
}

func newTools(ctx context.Context) []tool.BaseTool {
	k8sTools := kubernetes.NewKubernetesTools(ctx)
	monitorTools := monitor.NewMonitorTools(ctx)
	hostTools := host.NewHostTools(ctx)
	deploymentTools := []tool.InvokableTool{
		deployment.ClusterListInventory(ctx),
		deployment.ServiceListInventory(ctx),
	}

	readonly := make([]tool.BaseTool, 0, len(k8sTools)+len(monitorTools)+len(hostTools)+len(deploymentTools))
	for _, t := range k8sTools {
		readonly = append(readonly, t)
	}
	for _, t := range monitorTools {
		readonly = append(readonly, t)
	}
	for _, t := range hostTools {
		readonly = append(readonly, t)
	}
	for _, t := range deploymentTools {
		readonly = append(readonly, t)
	}

	writeTools := []tool.InvokableTool{
		kubernetes.K8sScaleDeployment(ctx),
		kubernetes.K8sRestartDeployment(ctx),
		kubernetes.K8sDeletePod(ctx),
		kubernetes.K8sRollbackDeployment(ctx),
		kubernetes.K8sDeleteDeployment(ctx),
		cicd.CICDPipelineTrigger(ctx),
		cicd.JobRun(ctx),
		service.ServiceDeployApply(ctx),
		service.ServiceDeploy(ctx),
	}

	result := make([]tool.BaseTool, 0, len(readonly)+len(writeTools))
	result = append(result, readonly...)
	for _, t := range writeTools {
		result = append(result, t)
	}
	return result
}

func newModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	return chatmodel.GetDefaultChatModel(ctx, nil, chatmodel.ChatModelConfig{
		Timeout: 45 * time.Second,
		Temp:    0.2,
	})
}
