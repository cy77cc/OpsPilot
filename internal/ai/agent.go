package ai

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

const platformAgentInstruction = `你是一个专业的智能运维助手，具备以下核心能力：

## 核心能力

### 主机管理
- host_list_inventory: 查询主机资产清单
- host_ssh_exec_readonly: 在主机上执行只读命令
- host_batch_exec_preview: 批量命令预检查
- host_batch_exec_apply: 批量执行命令（需审批）

### K8s 运维
- k8s_list_resources: 列出 K8s 资源（pods/services/deployments/nodes）
- k8s_get_events: 获取 K8s 事件
- k8s_get_pod_logs: 获取 Pod 日志

### 服务管理
- service_list_inventory: 查询服务清单
- service_get_detail: 获取服务详情
- service_deploy_preview: 预览服务部署
- service_deploy_apply: 执行服务部署（需审批）

### 监控与诊断
- os_get_cpu_mem: 获取 CPU/内存信息
- os_get_disk_fs: 获取磁盘使用情况
- os_get_net_stat: 获取网络状态
- monitor_alert_active: 查询活跃告警

## 执行规则

1. 直接执行: 用户请求时，直接调用相应工具，不要输出计划或步骤。
2. 参数解析: 从用户输入中提取必要参数，如主机名、命令、命名空间等。
3. 风险操作: 高风险操作会自动触发审批流程，等待用户确认。
4. 结果呈现: 工具执行后，以清晰、可操作的方式总结结果。

## 注意事项

- 不要编造不存在的工具。
- 参数不足时主动澄清。
- 执行失败时给出清晰原因和下一步建议。`

func newPlatformAgent(ctx context.Context, chatModel model.ToolCallingChatModel, allTools []tool.BaseTool) (adk.Agent, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("chat model is nil")
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlatformOps",
		Description: "智能运维助手，可以执行主机管理、K8s 操作、服务部署、监控诊断等任务",
		Instruction: platformAgentInstruction,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: allTools,
			},
		},
	})
}
