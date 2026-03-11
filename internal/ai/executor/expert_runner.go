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

	"github.com/cy77cc/OpsPilot/internal/ai/experts"
	expertspec "github.com/cy77cc/OpsPilot/internal/ai/experts/spec"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
)

type AgentStepRunner struct {
	agents map[string]adk.Agent
}

type expertResult struct {
	Summary       string   `json:"summary"`
	ObservedFacts []string `json:"observed_facts,omitempty"`
	Inferences    []string `json:"inferences,omitempty"`
	NextActions   []string `json:"next_actions,omitempty"`
	Narrative     string   `json:"narrative,omitempty"`
}

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

func (r *AgentStepRunner) RunStep(ctx context.Context, req Request, step planner.PlanStep) (StepResult, error) {
	if r == nil {
		return StepResult{}, &ExecutionError{
			Code:        "expert_tool_stream_failed",
			Message:     "expert step runner is not configured",
			UserSummary: "专家执行链路未正确初始化，当前步骤无法执行。",
		}
	}
	agent, ok := r.agents[strings.TrimSpace(step.Expert)]
	if !ok || agent == nil {
		return StepResult{}, &ExecutionError{
			Code:        "expert_not_registered",
			Message:     fmt.Sprintf("expert %q is not registered", step.Expert),
			UserSummary: fmt.Sprintf("未找到 %s 专家，无法执行该步骤。", strings.TrimSpace(step.Expert)),
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
			UserSummary: "专家已执行，但返回结果格式不符合协议。",
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
					"raw_output":     strings.TrimSpace(raw),
				},
			},
		},
	}, nil
}

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
- When you are done, call emit_expert_result exactly once.
- Do not return the final result as plain text.

Expert capabilities:
%s

%s

The executor has already decided whether approval is required. You only execute the authorized step and report the result.`, exp.Name(), exp.Name(), string(caps), extra)
}

func buildExpertRequest(req Request, step planner.PlanStep) string {
	input, _ := json.Marshal(step.Input)
	runtimeCtx, _ := json.Marshal(req.RuntimeContext)
	return fmt.Sprintf(
		"message: %s\nplan_goal: %s\nstep_id: %s\nstep_title: %s\nexpert: %s\nintent: %s\ntask: %s\nmode: %s\nrisk: %s\nstep_input: %s\nruntime_context: %s",
		strings.TrimSpace(req.Message),
		strings.TrimSpace(req.Plan.Goal),
		strings.TrimSpace(step.StepID),
		strings.TrimSpace(step.Title),
		strings.TrimSpace(step.Expert),
		strings.TrimSpace(step.Intent),
		strings.TrimSpace(step.Task),
		strings.TrimSpace(step.Mode),
		strings.TrimSpace(step.Risk),
		string(input),
		string(runtimeCtx),
	)
}

func parseExpertResult(raw string) (expertResult, error) {
	if strings.TrimSpace(raw) == "" {
		return expertResult{}, fmt.Errorf("expert returned an empty result")
	}
	var out expertResult
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return expertResult{}, fmt.Errorf("expert returned non-JSON output: %w", err)
	}
	if strings.TrimSpace(out.Summary) == "" && strings.TrimSpace(out.Narrative) == "" {
		return expertResult{}, fmt.Errorf("expert result is missing summary and narrative")
	}
	return out, nil
}

type expertDecision struct {
	info *schema.ToolInfo
}

func (t expertDecision) Info(_ context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t expertDecision) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	return argumentsInJSON, nil
}

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
			Code:        "expert_tool_stream_failed",
			Message:     compactToolError(err.Error()),
			UserSummary: fmt.Sprintf("专家 %s 调用模型超时，请稍后重试。", strings.TrimSpace(step.Expert)),
		}
	}
	return &ExecutionError{
		Code:        "expert_tool_execution_failed",
		Message:     compactToolError(err.Error()),
		UserSummary: fmt.Sprintf("专家 %s 执行失败：%s", strings.TrimSpace(step.Expert), compactToolError(err.Error())),
	}
}

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

func isProviderTimeoutError(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	if lower == "" {
		return false
	}
	return strings.Contains(lower, "context deadline exceeded") ||
		strings.Contains(lower, "client.timeout exceeded while awaiting headers") ||
		strings.Contains(lower, "timeout exceeded while awaiting headers")
}

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
