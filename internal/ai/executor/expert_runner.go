// Package executor 实现 AI 编排的执行阶段。
//
// 本文件实现专家步骤运行器 (AgentStepRunner)，负责调用专家 Agent 执行单个步骤。
// 使用 Eino ADK 构建专家 Agent，支持工具调用和结构化输出。
package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cy77cc/OpsPilot/internal/ai/availability"
	"github.com/cy77cc/OpsPilot/internal/ai/experts"
	expertspec "github.com/cy77cc/OpsPilot/internal/ai/experts/spec"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
)

// AgentStepRunner 是专家步骤运行器，负责调用专家 Agent 执行步骤。
type AgentStepRunner struct {
	agents map[string]adk.Agent // 专家名称到 Agent 的映射
}

// expertResult 表示专家执行的输出结果。
type expertResult struct {
	Summary       string   `json:"summary"`                  // 执行摘要
	ObservedFacts []string `json:"observed_facts,omitempty"` // 观察到的事实
	Inferences    []string `json:"inferences,omitempty"`     // 推断结论
	NextActions   []string `json:"next_actions,omitempty"`   // 建议的后续动作
	Narrative     string   `json:"narrative,omitempty"`      // 详细叙述
}

type expertRequestEnvelope struct {
	UserMessage     string                 `json:"user_message"`
	PlanGoal        string                 `json:"plan_goal,omitempty"`
	Step            expertRequestStep      `json:"step"`
	RuntimeContext  map[string]any         `json:"runtime_context,omitempty"`
	HostConstraints expertHostConstraints  `json:"host_constraints"`
}

type expertRequestStep struct {
	ID        string         `json:"id"`
	Title     string         `json:"title,omitempty"`
	Expert    string         `json:"expert,omitempty"`
	Intent    string         `json:"intent,omitempty"`
	Task      string         `json:"task,omitempty"`
	Mode      string         `json:"mode,omitempty"`
	Risk      string         `json:"risk,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	DependsOn []string       `json:"depends_on,omitempty"`
}

type expertHostConstraints struct {
	ApprovalAlreadyDecided bool `json:"approval_already_decided"`
	UseOnlyProvidedTools   bool `json:"use_only_provided_tools"`
	StayInAssignedDomain   bool `json:"stay_in_assigned_domain"`
}

// NewAgentStepRunner 创建新的专家步骤运行器。
// 从注册表中加载所有专家并为每个专家构建 Agent。
//
// 参数:
//   - ctx: 上下文
//   - model: 聊天模型
//   - registry: 专家注册表
func NewAgentStepRunner(ctx context.Context, model einomodel.BaseChatModel, registry *experts.Registry) (*AgentStepRunner, error) {
	if model == nil {
		return nil, fmt.Errorf("expert model is required")
	}
	if registry == nil {
		return nil, fmt.Errorf("expert registry is required")
	}

	items := make(map[string]adk.Agent)
	for _, exp := range registry.List() {
		if exp == nil {
			continue
		}
		agent, err := buildExpertAgent(ctx, model, exp)
		if err != nil {
			return nil, err
		}
		items[exp.Name()] = agent
	}
	return &AgentStepRunner{agents: items}, nil
}

// RunStep 执行单个步骤，调用对应的专家 Agent。
//
// 参数:
//   - ctx: 上下文
//   - req: 执行请求
//   - step: 计划步骤
//
// 返回: 步骤结果和可能的错误
func (r *AgentStepRunner) RunStep(ctx context.Context, req Request, step planner.PlanStep) (StepResult, error) {
	if r == nil {
		return StepResult{}, &ExecutionError{
			Code:        "expert_runner_unavailable",
			Message:     "expert step runner is not configured",
			UserSummary: availability.UnavailableMessage(availability.LayerExpert),
		}
	}
	agent, ok := r.agents[strings.TrimSpace(step.Expert)]
	if !ok || agent == nil {
		return StepResult{}, &ExecutionError{
			Code:        "expert_not_registered",
			Message:     fmt.Sprintf("expert %q is not registered", step.Expert),
			UserSummary: fmt.Sprintf("未找到 %s 专家，当前无法执行该步骤。", strings.TrimSpace(step.Expert)),
		}
	}
	raw, err := runExpertAgent(ctx, agent, buildExpertRequest(req, step))
	if err != nil {
		return StepResult{}, classifyExpertRunError(step, err)
	}

	out, err := parseExpertResult(strings.TrimSpace(raw))
	if err != nil {
		return StepResult{}, &ExecutionError{
			Code:        "expert_result_invalid",
			Message:     err.Error(),
			UserSummary: availability.InvalidOutputMessage(availability.LayerExpert),
		}
	}
	return StepResult{
		StepID:  step.StepID,
		Summary: firstNonEmpty(out.Summary, out.Narrative),
		Evidence: []Evidence{
			{
				Kind:   "expert_result",
				Source: step.Expert,
				Data: map[string]any{
					"summary":        out.Summary,
					"observed_facts": out.ObservedFacts,
					"inferences":     out.Inferences,
					"next_actions":   out.NextActions,
					"narrative":      out.Narrative,
					"structured_result": map[string]any{
						"summary":        out.Summary,
						"observed_facts": out.ObservedFacts,
						"inferences":     out.Inferences,
						"next_actions":   out.NextActions,
						"narrative":      out.Narrative,
					},
					"raw_output":     strings.TrimSpace(raw),
					"input_envelope": buildExpertRequestEnvelope(req, step),
				},
			},
		},
	}, nil
}

// buildExpertAgent 为单个专家构建 Agent。
// 将专家的工具集和系统提示配置到 Agent 中。
func buildExpertAgent(ctx context.Context, model einomodel.BaseChatModel, exp expertspec.Expert) (adk.Agent, error) {
	baseTools := exp.Tools(ctx)
	toolset := make([]tool.BaseTool, 0, len(baseTools)+1)
	for _, item := range baseTools {
		if item != nil {
			toolset = append(toolset, item)
		}
	}
	toolset = append(toolset, expertDecisionTool())

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          exp.Name(),
		Description:   exp.Description(),
		Instruction:   expertSystemPrompt(exp),
		Model:         model,
		MaxIterations: 6,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolset,
			},
			ReturnDirectly: map[string]bool{
				"emit_expert_result": true,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}

// expertSystemPrompt 生成专家的系统提示词。
// 包含专家能力描述、执行约束和领域特定规则。
func expertSystemPrompt(exp expertspec.Expert) string {
	caps, _ := json.Marshal(exp.Capabilities())
	extra := ""
	if strings.TrimSpace(exp.Name()) == "hostops" {
		extra = `

Host-specific rules:
- If step_input.scope.kind is "all" or "filtered" for resource_type "host" and the step is readonly status/inventory work, use host_list_inventory first.
- Do NOT use host_batch for fleet status summaries unless step_input.host_ids is explicitly present and the task clearly requires running the same command on each host.
- If the user asks for all hosts / fleet status / host inventory, prefer inventory facts over remote command execution.`
	}
	return fmt.Sprintf(`You are the %s expert in an AI operations orchestrator.

Your responsibility is to execute exactly one executor step using only your own domain tools.

Guardrails:
- You MUST stay inside the %s domain.
- You MUST use only the tools provided to you in this expert agent.
- You MUST NOT assume planner support tools, hidden tools, or tools from other experts exist.
- You MUST NOT fabricate resource IDs, permissions, tool results, logs, or execution outcomes.
- You MUST distinguish observed facts from inferred conclusions.
- If the available evidence is incomplete, say so explicitly in inferences or next_actions.
- Prefer calling emit_expert_result exactly once with structured JSON.
- If structured emission fails, return a compact half-structured result that clearly separates summary, observed facts, inferences, and next actions.

Expert capabilities:
%s

%s

The executor has already decided whether approval is required. You only execute the authorized step and report the result.`, exp.Name(), exp.Name(), string(caps), extra)
}

// buildExpertRequest 构建发送给专家 Agent 的结构化 envelope。
func buildExpertRequest(req Request, step planner.PlanStep) string {
	payload, _ := json.Marshal(buildExpertRequestEnvelope(req, step))
	return string(payload)
}

func buildExpertRequestEnvelope(req Request, step planner.PlanStep) expertRequestEnvelope {
	runtimeCtx, _ := json.Marshal(req.RuntimeContext)
	var runtimeMap map[string]any
	_ = json.Unmarshal(runtimeCtx, &runtimeMap)
	return expertRequestEnvelope{
		UserMessage: strings.TrimSpace(req.Message),
		PlanGoal:    strings.TrimSpace(req.Plan.Goal),
		Step: expertRequestStep{
			ID:        strings.TrimSpace(step.StepID),
			Title:     strings.TrimSpace(step.Title),
			Expert:    strings.TrimSpace(step.Expert),
			Intent:    strings.TrimSpace(step.Intent),
			Task:      strings.TrimSpace(step.Task),
			Mode:      strings.TrimSpace(step.Mode),
			Risk:      strings.TrimSpace(step.Risk),
			Input:     cloneExpertInput(step.Input),
			DependsOn: append([]string(nil), step.DependsOn...),
		},
		RuntimeContext: runtimeMap,
		HostConstraints: expertHostConstraints{
			ApprovalAlreadyDecided: true,
			UseOnlyProvidedTools:   true,
			StayInAssignedDomain:   true,
		},
	}
}

func cloneExpertInput(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

// parseExpertResult 解析专家返回的 JSON 结果。
// 如果解析失败，尝试从半结构化文本中恢复。
func parseExpertResult(raw string) (expertResult, error) {
	if strings.TrimSpace(raw) == "" {
		return expertResult{}, fmt.Errorf("expert returned an empty result")
	}
	var out expertResult
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		recovered, recoverErr := recoverHalfStructuredExpertResult(raw)
		if recoverErr != nil {
			return expertResult{}, fmt.Errorf("expert returned non-JSON output: %w", err)
		}
		return recovered, nil
	}
	if strings.TrimSpace(out.Summary) == "" && strings.TrimSpace(out.Narrative) == "" {
		recovered, err := recoverHalfStructuredExpertResult(raw)
		if err != nil {
			return expertResult{}, fmt.Errorf("expert result is missing summary and narrative")
		}
		return recovered, nil
	}
	return out, nil
}

// recoverHalfStructuredExpertResult 从半结构化文本中恢复专家结果。
// 尝试识别 observed_facts、inferences、next_actions 等部分。
func recoverHalfStructuredExpertResult(raw string) (expertResult, error) {
	lines := splitNonEmptyLines(raw)
	if len(lines) == 0 {
		return expertResult{}, fmt.Errorf("expert returned empty text")
	}
	out := expertResult{
		Summary:   lines[0],
		Narrative: strings.TrimSpace(raw),
	}
	for _, line := range lines[1:] {
		normalized := strings.TrimLeft(strings.TrimSpace(line), "-* ")
		switch {
		case hasAnyPrefix(normalized, "observed:", "observed facts:", "事实:", "观察:"):
			out.ObservedFacts = append(out.ObservedFacts, trimSectionPrefix(normalized))
		case hasAnyPrefix(normalized, "inference:", "inferences:", "推断:", "判断:"):
			out.Inferences = append(out.Inferences, trimSectionPrefix(normalized))
		case hasAnyPrefix(normalized, "next:", "next actions:", "建议:", "后续:"):
			out.NextActions = append(out.NextActions, trimSectionPrefix(normalized))
		default:
			if out.Summary == "" {
				out.Summary = normalized
				continue
			}
			out.ObservedFacts = append(out.ObservedFacts, normalized)
		}
	}
	if strings.TrimSpace(out.Summary) == "" {
		out.Summary = lines[0]
	}
	return out, nil
}

func splitNonEmptyLines(raw string) []string {
	parts := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func hasAnyPrefix(value string, prefixes ...string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, strings.ToLower(strings.TrimSpace(prefix))) {
			return true
		}
	}
	return false
}

func trimSectionPrefix(value string) string {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{"observed:", "observed facts:", "事实:", "观察:", "inference:", "inferences:", "推断:", "判断:", "next:", "next actions:", "建议:", "后续:"} {
		if strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix)) {
			return strings.TrimSpace(value[len(prefix):])
		}
	}
	return value
}

// expertDecision 是专家决策工具，用于输出结构化结果。
type expertDecision struct {
	info *schema.ToolInfo
}

func (t expertDecision) Info(_ context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t expertDecision) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	return argumentsInJSON, nil
}

// expertDecisionTool 创建专家决策工具。
// 专家通过调用此工具输出结构化的执行结果。
func expertDecisionTool() tool.BaseTool {
	return expertDecision{
		info: &schema.ToolInfo{
			Name: "emit_expert_result",
			Desc: "Emit the final expert step result as structured JSON.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"summary": {
					Type:     schema.String,
					Required: true,
					Desc:     "Short user-visible summary of what the expert found or did.",
				},
				"observed_facts": {
					Type: schema.Array,
					ElemInfo: &schema.ParameterInfo{
						Type: schema.String,
					},
					Desc: "Observed facts directly supported by tool output.",
				},
				"inferences": {
					Type: schema.Array,
					ElemInfo: &schema.ParameterInfo{
						Type: schema.String,
					},
					Desc: "Inferences or judgments that are not fully proven facts.",
				},
				"next_actions": {
					Type: schema.Array,
					ElemInfo: &schema.ParameterInfo{
						Type: schema.String,
					},
					Desc: "Recommended follow-up actions if any.",
				},
				"narrative": {
					Type: schema.String,
					Desc: "Additional explanatory narrative for the executor and summarizer.",
				},
			}),
		},
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// runExpertAgent 运行专家 Agent 并收集最终输出。
func runExpertAgent(ctx context.Context, agent adk.Agent, request string) (string, error) {
	iter := agent.Run(ctx, &adk.AgentInput{
		Messages: []adk.Message{
			schema.UserMessage(request),
		},
	})
	var last string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		msg, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(msg.Content) != "" {
			last = msg.Content
		}
	}
	if strings.TrimSpace(last) == "" {
		return "", fmt.Errorf("expert returned no final output")
	}
	return strings.TrimSpace(last), nil
}

// classifyExpertRunError 对专家执行错误进行分类和包装。
// 将原始错误转换为用户友好的 ExecutionError。
func classifyExpertRunError(step planner.PlanStep, err error) error {
	if err == nil {
		return nil
	}
	if execErr, ok := err.(*ExecutionError); ok {
		return execErr
	}
	if summary, field, ok := summarizeMissingPrerequisite(err.Error()); ok {
		return &ExecutionError{
			Code:        "missing_execution_prerequisite",
			Message:     compactToolError(err.Error()),
			UserSummary: fmt.Sprintf("%s。缺少前置上下文：%s", summary, field),
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || isProviderTimeoutError(err.Error()) {
		return &ExecutionError{
			Code:        "expert_model_unavailable",
			Message:     compactToolError(err.Error()),
			UserSummary: availability.UnavailableMessage(availability.LayerExpert),
		}
	}
	return &ExecutionError{
		Code:        "expert_tool_execution_failed",
		Message:     compactToolError(err.Error()),
		UserSummary: fmt.Sprintf("专家 %s 执行失败：%s", strings.TrimSpace(step.Expert), compactToolError(err.Error())),
	}
}

// compactToolError 压缩工具错误消息，移除冗余信息。
func compactToolError(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	if isProviderTimeoutError(message) {
		return "调用模型超时"
	}
	if idx := strings.Index(message, "err="); idx >= 0 {
		message = strings.TrimSpace(message[idx+4:])
	}
	if idx := strings.Index(message, "------------------------"); idx >= 0 {
		message = strings.TrimSpace(message[:idx])
	}
	return message
}

// isProviderTimeoutError 判断是否为模型提供商超时错误。
func isProviderTimeoutError(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}
	return strings.Contains(lower, "context deadline exceeded") ||
		strings.Contains(lower, "client.timeout exceeded while awaiting headers") ||
		strings.Contains(lower, "timeout exceeded while awaiting headers")
}

// summarizeMissingPrerequisite 从错误消息中提取缺失的前置条件。
// 返回用户友好的摘要和缺失字段名。
func summarizeMissingPrerequisite(message string) (string, string, bool) {
	message = compactToolError(message)
	switch {
	case strings.Contains(message, "cluster_id is required"):
		return "当前没有可执行的集群上下文", "cluster_id", true
	case strings.Contains(message, "service_id is required"):
		return "当前没有可执行的服务上下文", "service_id", true
	case strings.Contains(message, "host_id is required"):
		return "当前没有可执行的主机上下文", "host_id", true
	case strings.Contains(message, "host_ids is required"):
		return "当前没有可执行的主机上下文", "host_ids", true
	case strings.Contains(message, "pipeline_id is required"):
		return "当前没有可执行的流水线上下文", "pipeline_id", true
	case strings.Contains(message, "job_id is required"):
		return "当前没有可执行的任务上下文", "job_id", true
	case strings.Contains(message, "target_id is required"):
		return "当前没有可执行的部署目标上下文", "target_id", true
	case strings.Contains(message, "credential_id is required"):
		return "当前没有可执行的凭据上下文", "credential_id", true
	case strings.Contains(message, "pod is required"):
		return "当前没有可执行的 Pod 上下文", "pod", true
	default:
		return "", "", false
	}
}
