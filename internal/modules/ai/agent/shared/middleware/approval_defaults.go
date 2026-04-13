package middleware

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"
)

// DefaultNeedsApproval 默认审批判断逻辑。
func DefaultNeedsApproval(toolName string) bool {
	approvalRequired := map[string]bool{
		"host_exec":                true,
		"k8s_scale_deployment":     true,
		"k8s_restart_deployment":   true,
		"k8s_delete_pod":           true,
		"k8s_rollback_deployment":  true,
		"k8s_delete_deployment":    true,
		"cicd_pipeline_trigger":    true,
		"job_run":                  true,
		"service_deploy_apply":     true,
		"service_deploy":           true,
	}
	return approvalRequired[toolName]
}

// DefaultPreviewGenerator 默认预览生成器。
func DefaultPreviewGenerator(toolName, args string) approval.ApprovalPreview {
	preview := approval.ApprovalPreview{
		Action:    toolName,
		RiskLevel: approval.RiskLevelMedium,
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(args), &params); err == nil {
		if target, ok := params["target"].(string); ok {
			preview.Target = target
		}
		if hostIDs, ok := params["host_ids"].([]any); ok {
			preview.Target = fmt.Sprintf("%d hosts", len(hostIDs))
		}
		if name, ok := params["name"].(string); ok {
			preview.Target = name
		}
		if ns, ok := params["namespace"].(string); ok {
			if preview.Target != "" {
				preview.Target = ns + "/" + preview.Target
			} else {
				preview.Target = ns
			}
		}
		if cmd, ok := params["command"].(string); ok {
			preview.Action = cmd
		}
		if action, ok := params["action"].(string); ok {
			preview.Action = action
		}
	}

	switch toolName {
	case "k8s_delete_pod":
		preview.RiskLevel = approval.RiskLevelHigh
		preview.Impact = "Pod 将被删除，控制器可能会重建新 Pod，可能导致短暂服务中断"
		preview.Warnings = append(preview.Warnings, "删除 Pod 不会影响 Deployment 的副本数")
	case "k8s_delete_deployment":
		preview.RiskLevel = approval.RiskLevelCritical
		preview.Impact = "Deployment 将被永久删除，服务将停止"
		preview.Warnings = append(preview.Warnings, "此操作不可逆，请确认是否真的需要删除")
	case "k8s_restart_deployment":
		preview.RiskLevel = approval.RiskLevelMedium
		preview.Impact = "Deployment 将滚动重启，可能导致短暂的服务不稳定"
	case "k8s_scale_deployment":
		preview.RiskLevel = approval.RiskLevelMedium
		preview.Impact = "副本数变更将影响服务容量和资源消耗"
	case "k8s_rollback_deployment":
		preview.RiskLevel = approval.RiskLevelMedium
		preview.Impact = "Deployment 将回滚到上一版本，可能导致功能变更"
	case "cicd_pipeline_trigger":
		preview.RiskLevel = approval.RiskLevelMedium
		preview.Impact = "将触发 CI/CD 流水线执行，可能影响部署环境"
	case "job_run":
		preview.RiskLevel = approval.RiskLevelMedium
		preview.Impact = "将手动触发计划作业执行，可能影响生产任务或外部系统"
	case "service_deploy_apply":
		preview.RiskLevel = approval.RiskLevelHigh
		preview.Impact = "将把服务部署到目标集群，可能引起版本变更、流量波动或短暂中断"
	case "service_deploy":
		preview.RiskLevel = approval.RiskLevelHigh
		preview.Impact = "统一部署工具在 apply 模式下会实际下发部署变更，请确认目标服务和集群"
	}

	return preview
}

// DefaultToolConfigs 默认工具配置。
func DefaultToolConfigs() map[string]*approval.ToolRiskConfig {
	return map[string]*approval.ToolRiskConfig{
		"host_exec": {
			ToolName:         "host_exec",
			RiskLevel:        approval.RiskLevelHigh,
			NeedsApproval:    true,
			PreviewGenerator: hostSingleExecPreviewGenerator,
		},
		"k8s_delete_pod": {
			ToolName:         "k8s_delete_pod",
			RiskLevel:        approval.RiskLevelHigh,
			NeedsApproval:    true,
			PreviewGenerator: k8sPodPreviewGenerator,
		},
		"k8s_restart_deployment": {
			ToolName:         "k8s_restart_deployment",
			RiskLevel:        approval.RiskLevelMedium,
			NeedsApproval:    true,
			PreviewGenerator: k8sDeploymentPreviewGenerator,
		},
		"k8s_scale_deployment": {
			ToolName:         "k8s_scale_deployment",
			RiskLevel:        approval.RiskLevelMedium,
			NeedsApproval:    true,
			PreviewGenerator: k8sScalePreviewGenerator,
		},
		"k8s_rollback_deployment": {
			ToolName:         "k8s_rollback_deployment",
			RiskLevel:        approval.RiskLevelMedium,
			NeedsApproval:    true,
			PreviewGenerator: k8sRollbackPreviewGenerator,
		},
		"k8s_delete_deployment": {
			ToolName:         "k8s_delete_deployment",
			RiskLevel:        approval.RiskLevelCritical,
			NeedsApproval:    true,
			PreviewGenerator: k8sDeleteDeploymentPreviewGenerator,
		},
		"cicd_pipeline_trigger": {
			ToolName:      "cicd_pipeline_trigger",
			RiskLevel:     approval.RiskLevelMedium,
			NeedsApproval: true,
		},
		"job_run": {
			ToolName:      "job_run",
			RiskLevel:     approval.RiskLevelMedium,
			NeedsApproval: true,
		},
		"service_deploy_apply": {
			ToolName:      "service_deploy_apply",
			RiskLevel:     approval.RiskLevelHigh,
			NeedsApproval: true,
		},
		"service_deploy": {
			ToolName:      "service_deploy",
			RiskLevel:     approval.RiskLevelHigh,
			NeedsApproval: true,
		},
	}
}

func hostSingleExecPreviewGenerator(args string) approval.ApprovalPreview {
	preview := approval.ApprovalPreview{
		Action:    "host_execute",
		RiskLevel: approval.RiskLevelHigh,
	}

	var params struct {
		HostID  int    `json:"host_id"`
		Command string `json:"command"`
		Script  string `json:"script"`
		Reason  string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(args), &params); err == nil {
		target := ""
		if params.HostID > 0 {
			target = fmt.Sprintf("host %d", params.HostID)
		}
		preview.Target = target

		execText := strings.TrimSpace(params.Command)
		if execText == "" {
			execText = strings.TrimSpace(params.Script)
		}
		if execText != "" {
			preview.Action = execText
		}

		preview.Extra = map[string]any{
			"host_id": params.HostID,
			"reason":  params.Reason,
		}

		cmdLower := strings.ToLower(execText)
		if strings.Contains(cmdLower, "rm ") ||
			strings.Contains(cmdLower, "delete") ||
			strings.Contains(cmdLower, "shutdown") ||
			strings.Contains(cmdLower, "reboot") ||
			strings.Contains(cmdLower, "mkfs") {
			preview.RiskLevel = approval.RiskLevelCritical
			preview.Warnings = append(preview.Warnings, "命令具有破坏性，请仔细确认")
		}

		if execText != "" && target != "" {
			preview.Impact = fmt.Sprintf("将在 %s 上执行命令: %s", target, execText)
		} else if execText != "" {
			preview.Impact = fmt.Sprintf("将在目标主机上执行命令: %s", execText)
		} else {
			preview.Impact = "将在目标主机上执行命令，请确认命令内容和影响范围"
		}
	}

	return preview
}

func k8sPodPreviewGenerator(args string) approval.ApprovalPreview {
	preview := approval.ApprovalPreview{
		Action:    "delete_pod",
		RiskLevel: approval.RiskLevelHigh,
	}

	var params struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}

	if err := json.Unmarshal([]byte(args), &params); err == nil {
		if params.Namespace != "" {
			preview.Target = params.Namespace + "/" + params.Name
		} else {
			preview.Target = params.Name
		}
		preview.Impact = fmt.Sprintf("Pod %s 将被删除，控制器可能会重建新 Pod", preview.Target)
		preview.Warnings = append(preview.Warnings,
			"删除 Pod 不会影响 Deployment 副本数",
			"新 Pod 可能调度到不同节点",
		)
	}

	return preview
}

func k8sDeploymentPreviewGenerator(args string) approval.ApprovalPreview {
	preview := approval.ApprovalPreview{
		Action:    "restart_deployment",
		RiskLevel: approval.RiskLevelMedium,
	}

	var params struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}

	if err := json.Unmarshal([]byte(args), &params); err == nil {
		if params.Namespace != "" {
			preview.Target = params.Namespace + "/" + params.Name
		} else {
			preview.Target = params.Name
		}
		preview.Impact = fmt.Sprintf("Deployment %s 将滚动重启", preview.Target)
	}

	return preview
}

func k8sScalePreviewGenerator(args string) approval.ApprovalPreview {
	preview := approval.ApprovalPreview{
		Action:    "scale_deployment",
		RiskLevel: approval.RiskLevelMedium,
	}

	var params struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Replicas  int    `json:"replicas"`
	}

	if err := json.Unmarshal([]byte(args), &params); err == nil {
		if params.Namespace != "" {
			preview.Target = params.Namespace + "/" + params.Name
		} else {
			preview.Target = params.Name
		}
		preview.Extra = map[string]any{
			"replicas": params.Replicas,
		}
		preview.Impact = fmt.Sprintf("Deployment %s 副本数将调整为 %d", preview.Target, params.Replicas)
	}

	return preview
}

func k8sRollbackPreviewGenerator(args string) approval.ApprovalPreview {
	preview := approval.ApprovalPreview{
		Action:    "rollback_deployment",
		RiskLevel: approval.RiskLevelMedium,
	}

	var params struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Revision  int64  `json:"revision"`
	}

	if err := json.Unmarshal([]byte(args), &params); err == nil {
		if params.Namespace != "" {
			preview.Target = params.Namespace + "/" + params.Name
		} else {
			preview.Target = params.Name
		}
		if params.Revision > 0 {
			preview.Extra = map[string]any{
				"target_revision": params.Revision,
			}
			preview.Impact = fmt.Sprintf("Deployment %s 将回滚到版本 %d", preview.Target, params.Revision)
		} else {
			preview.Impact = fmt.Sprintf("Deployment %s 将回滚到上一版本", preview.Target)
		}
		preview.Warnings = append(preview.Warnings, "回滚可能导致功能变更，请确认版本差异")
	}

	return preview
}

func k8sDeleteDeploymentPreviewGenerator(args string) approval.ApprovalPreview {
	preview := approval.ApprovalPreview{
		Action:    "delete_deployment",
		RiskLevel: approval.RiskLevelCritical,
	}

	var params struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}

	if err := json.Unmarshal([]byte(args), &params); err == nil {
		if params.Namespace != "" {
			preview.Target = params.Namespace + "/" + params.Name
		} else {
			preview.Target = params.Name
		}
		preview.Impact = fmt.Sprintf("Deployment %s 将被永久删除，服务将停止", preview.Target)
		preview.Warnings = append(preview.Warnings,
			"此操作不可逆，请确认是否真的需要删除",
			"删除 Deployment 将同时删除关联的 ReplicaSet 和 Pod",
		)
	}

	return preview
}
