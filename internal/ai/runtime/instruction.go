// Package runtime 定义 AI 运行时的核心类型和组件。
//
// 本文件提供 System Prompt 模板和动态渲染能力，
// 支持根据 RuntimeContext 注入场景、项目、选中资源等上下文信息。
package runtime

import (
	"fmt"
	"strings"
)

// InstructionTemplate 是 OpsPilot 智能运维助手的系统提示词模板。
//
// 占位符 {xxx} 会在运行时被 RuntimeContext 中的对应值替换。
const InstructionTemplate = `你是 OpsPilot 智能运维助手，负责协助用户管理 Kubernetes 集群、主机、服务等基础设施资源。

## 核心能力
- 集群管理：查询集群状态、节点信息、资源使用情况
- 主机运维：批量执行命令、查看日志、监控状态
- 服务管理：部署、扩缩容、重启、查看状态
- 故障排查：分析日志、诊断问题、提供建议

## 工作原则
1. 优先使用只读工具收集信息，确认后再执行变更操作
2. 变更操作需要用户确认后才可执行
3. 操作前说明目的和预期影响
4. 遇到错误时分析原因并提供解决建议

## 当前上下文
- 场景: {scene_name}
- 项目: {project_name}
- 页面: {current_page}
- 选中资源: {selected_resources}

请根据用户需求，合理使用工具完成任务。`

// BuildInstruction 根据 RuntimeContext 渲染系统提示词。
//
// 空值字段会被替换为默认文本，确保模板完整有效。
func BuildInstruction(ctx RuntimeContext) string {
	result := InstructionTemplate

	result = strings.ReplaceAll(result, "{scene_name}", instructionSceneName(ctx))
	result = strings.ReplaceAll(result, "{project_name}", instructionProjectName(ctx))
	result = strings.ReplaceAll(result, "{current_page}", instructionCurrentPage(ctx))
	result = strings.ReplaceAll(result, "{selected_resources}", formatSelectedResources(ctx.SelectedResources))

	return result
}

func instructionSceneName(ctx RuntimeContext) string {
	for _, candidate := range []string{
		ctx.SceneName,
		stringMetadata(ctx.Metadata, "scene_name"),
		stringMetadata(ctx.Metadata, "scene"),
		ctx.Scene,
	} {
		if value := strings.TrimSpace(candidate); value != "" {
			return value
		}
	}
	return "通用"
}

func instructionProjectName(ctx RuntimeContext) string {
	for _, candidate := range []string{
		ctx.ProjectName,
		stringMetadata(ctx.Metadata, "project_name"),
		ctx.ProjectID,
	} {
		if value := strings.TrimSpace(candidate); value != "" {
			return value
		}
	}
	return "未指定"
}

func instructionCurrentPage(ctx RuntimeContext) string {
	for _, candidate := range []string{
		ctx.CurrentPage,
		ctx.Route,
		stringMetadata(ctx.Metadata, "current_page"),
	} {
		if value := strings.TrimSpace(candidate); value != "" {
			return value
		}
	}
	return "未指定"
}

func stringMetadata(meta map[string]any, key string) string {
	if len(meta) == 0 {
		return ""
	}
	value, _ := meta[key]
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

// formatSelectedResources 格式化选中资源列表为可读文本。
func formatSelectedResources(resources []SelectedResource) string {
	if len(resources) == 0 {
		return "无"
	}

	var sb strings.Builder
	for i, resource := range resources {
		if i > 0 {
			sb.WriteString(", ")
		}
		name := strings.TrimSpace(resource.Name)
		if name == "" {
			name = strings.TrimSpace(resource.ID)
		}
		if strings.TrimSpace(resource.Type) != "" {
			sb.WriteString(fmt.Sprintf("%s(%s)", name, strings.TrimSpace(resource.Type)))
			continue
		}
		sb.WriteString(name)
	}
	return sb.String()
}
