// Package middleware 提供 AI 工具的中间件实现。
//
// 本文件实现审批中间件，用于拦截高风险工具调用并通过 Eino 的
// Interrupt/Resume 机制实现 Human-in-the-Loop (HITL) 工作流。
//
// 工作流程:
//  1. 工具调用时，中间件检查是否需要审批
//  2. 如需审批，调用 tool.StatefulInterrupt 暂停执行
//  3. Runner 检测到中断，通过 SSE 发送 tool_approval 事件给前端
//  4. 用户在前端审批，结果通过 API 携带 ApprovalResult 恢复
//  5. 中间件根据审批结果决定继续执行或返回拒绝消息
//
// 使用示例:
//
//	cfg := &ApprovalMiddlewareConfig{
//	    NeedsApproval:    DefaultNeedsApproval,
//	    PreviewGenerator: DefaultPreviewGenerator,
//	}
//	mw := ApprovalMiddleware(cfg)
//	agent := adk.WithMiddleware(baseAgent, mw)
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

// ApprovalMiddlewareConfig 审批中间件配置。
type ApprovalMiddlewareConfig struct {
	// Orchestrator evaluates DB-driven approval policy and creates approval snapshots.
	Orchestrator common.ApprovalEvaluator

	// NeedsApproval 判断工具是否需要审批
	// 返回 true 表示该工具需要人工审批后才能执行
	NeedsApproval func(toolName string) bool

	// PreviewGenerator 生成审批预览信息
	// 用于在前端展示操作的详细信息和潜在影响
	PreviewGenerator func(toolName, args string) common.ApprovalPreview

	// DefaultTimeout 默认审批超时时间（秒）
	// 超时后审批请求将自动失效
	DefaultTimeout int

	// ToolConfigs 特定工具的配置映射
	// key 为工具名称，value 为该工具的风险配置
	ToolConfigs map[string]*common.ToolRiskConfig
}

// ApprovalMiddleware 创建审批中间件。
//
// 该中间件拦截需要审批的工具调用，通过 StatefulInterrupt 暂停执行，
// 等待用户通过 ResumeWithParams 携带审批结果后继续。
//
// 参数:
//   - cfg: 中间件配置，如果为 nil 则使用默认配置
//
// 返回: 可应用到 Agent 的中间件实例
func ApprovalMiddleware(cfg *ApprovalMiddlewareConfig) adk.ChatModelAgentMiddleware {
	if cfg == nil {
		cfg = &ApprovalMiddlewareConfig{}
	}
	if cfg.NeedsApproval == nil {
		cfg.NeedsApproval = DefaultNeedsApproval
	}
	if cfg.PreviewGenerator == nil {
		cfg.PreviewGenerator = DefaultPreviewGenerator
	}
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = common.DefaultApprovalTimeout
	}
	if cfg.ToolConfigs == nil {
		cfg.ToolConfigs = DefaultToolConfigs()
	}

	return &approvalMiddleware{
		config: cfg,
	}
}

// NewApprovalToolMiddleware 创建用于 ToolsConfig 的审批中间件。
//
// 该函数直接返回 compose.ToolMiddleware，可以在 ToolsConfig.ToolCallMiddlewares 中使用。
//
// 参数:
//   - cfg: 中间件配置，如果为 nil 则使用默认配置
//
// 返回: 可添加到 ToolCallMiddlewares 的中间件
func NewApprovalToolMiddleware(cfg *ApprovalMiddlewareConfig) compose.ToolMiddleware {
	mw := ApprovalMiddleware(cfg).(*approvalMiddleware)
	return mw.AsToolMiddleware()
}

// approvalMiddleware 审批中间件实现。
type approvalMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	config *ApprovalMiddlewareConfig
}

type approvalInterruptState struct {
	ArgumentsInJSON string
	Decision        *common.ApprovalDecision
}

// WrapInvokableToolCall 包装同步工具调用。
//
// 对于需要审批的工具，在首次调用时触发中断，
// 在恢复时检查审批结果并决定是否继续执行。
func (m *approvalMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	// 只拦截需要审批的工具
	if !m.requiresApprovalGate(tCtx.Name) {
		return endpoint, nil
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		decision, storedArgs, wasInterrupted, err := m.evaluateApproval(ctx, tCtx.Name, args, tCtx.CallID)
		if err != nil {
			return "", err
		}

		if !wasInterrupted {
			if decision == nil || !decision.RequiresApproval {
				return endpoint(ctx, args, opts...)
			}
			return "", m.raiseApprovalInterrupt(ctx, tCtx.Name, tCtx.CallID, args, decision)
		}

		// 已中断过，检查恢复上下文
		isTarget, hasData, result := tool.GetResumeContext[*common.ApprovalResult](ctx)

		if isTarget && hasData {
			if result.Approved {
				// 审批通过，执行原始工具
				return endpoint(ctx, storedArgs, opts...)
			}
			// 审批拒绝
			return m.formatDisapproveMessage(tCtx.Name, result), nil
		}

		// 恢复上下文不匹配（可能是其他工具的中断），重新中断
		if decision == nil {
			decision = m.defaultDecision(tCtx.Name, storedArgs, tCtx.CallID)
		}
		return "", m.raiseApprovalInterrupt(ctx, tCtx.Name, tCtx.CallID, storedArgs, decision)
	}, nil
}

// WrapStreamableToolCall 包装流式工具调用。
//
// 流式工具的中断恢复逻辑与同步工具类似，
// 区别在于返回 StreamReader 而非字符串。
func (m *approvalMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	if !m.requiresApprovalGate(tCtx.Name) {
		return endpoint, nil
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		decision, storedArgs, wasInterrupted, err := m.evaluateApproval(ctx, tCtx.Name, args, tCtx.CallID)
		if err != nil {
			return nil, err
		}

		if !wasInterrupted {
			if decision == nil || !decision.RequiresApproval {
				return endpoint(ctx, args, opts...)
			}
			return nil, m.raiseApprovalInterrupt(ctx, tCtx.Name, tCtx.CallID, args, decision)
		}

		isTarget, hasData, result := tool.GetResumeContext[*common.ApprovalResult](ctx)

		if isTarget && hasData {
			if result.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			return singleChunkReader(m.formatDisapproveMessage(tCtx.Name, result)), nil
		}

		if decision == nil {
			decision = m.defaultDecision(tCtx.Name, storedArgs, tCtx.CallID)
		}
		return nil, m.raiseApprovalInterrupt(ctx, tCtx.Name, tCtx.CallID, storedArgs, decision)
	}, nil
}

// generatePreview 生成审批预览信息。
//
// 优先使用工具特定配置的生成器，否则使用默认生成器。
func (m *approvalMiddleware) generatePreview(toolName, args string) common.ApprovalPreview {
	// 检查是否有工具特定配置
	if cfg, ok := m.config.ToolConfigs[toolName]; ok && cfg.PreviewGenerator != nil {
		return cfg.PreviewGenerator(args)
	}
	// 使用默认生成器
	return m.config.PreviewGenerator(toolName, args)
}

func (m *approvalMiddleware) requiresApprovalGate(toolName string) bool {
	if m.config.Orchestrator != nil {
		return true
	}
	return m.config.NeedsApproval(toolName)
}

func (m *approvalMiddleware) evaluateApproval(ctx context.Context, toolName, args, callID string) (*common.ApprovalDecision, string, bool, error) {
	if m.config.Orchestrator == nil {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if wasInterrupted {
			return m.defaultDecision(toolName, storedArgs, callID), storedArgs, true, nil
		}
		if !m.config.NeedsApproval(toolName) {
			return &common.ApprovalDecision{RequiresApproval: false}, storedArgs, false, nil
		}
		return m.defaultDecision(toolName, args, callID), args, false, nil
	}

	wasInterrupted, hasDecisionState, state := tool.GetInterruptState[approvalInterruptState](ctx)
	if wasInterrupted && hasDecisionState {
		return state.Decision, state.ArgumentsInJSON, true, nil
	}
	if wasInterrupted {
		_, hasStringState, storedArgs := tool.GetInterruptState[string](ctx)
		if hasStringState {
			return m.defaultDecision(toolName, storedArgs, callID), storedArgs, true, nil
		}
		return m.defaultDecision(toolName, args, callID), args, true, nil
	}

	sceneMeta := runtimectx.AIMetadataFrom(ctx)
	meta := common.ApprovalEvalMeta{
		SessionID:      sceneMeta.SessionID,
		RunID:          sceneMeta.RunID,
		CheckpointID:   sceneMeta.CheckpointID,
		Scene:          sceneMeta.Scene,
		CallID:         callID,
		CommandClass:   commandClassForTool(toolName, args),
		UserID:         sceneMeta.UserID,
		TimeoutSeconds: m.config.DefaultTimeout,
	}

	decision, err := m.config.Orchestrator.Evaluate(ctx, toolName, args, meta)
	if err != nil {
		return nil, "", false, err
	}
	if decision == nil {
		return nil, "", false, nil
	}
	return decision, args, false, nil
}

func (m *approvalMiddleware) raiseApprovalInterrupt(ctx context.Context, toolName, callID, args string, decision *common.ApprovalDecision) error {
	if decision == nil {
		decision = m.defaultDecision(toolName, args, callID)
	}
	info := map[string]any{
		"approval_id":     decision.ApprovalID,
		"call_id":         callID,
		"tool_name":       toolName,
		"preview":         decision.Preview,
		"timeout_seconds": decision.TimeoutSeconds,
		"expires_at":      decision.ExpiresAt.UTC().Format(time.RFC3339Nano),
		"decision_source": decision.DecisionSource,
	}
	if decision.MatchedRuleID != nil {
		info["matched_rule_id"] = *decision.MatchedRuleID
	}
	if strings.TrimSpace(decision.PolicyVersion) != "" {
		info["policy_version"] = strings.TrimSpace(decision.PolicyVersion)
	}
	return tool.StatefulInterrupt(ctx, info, approvalInterruptState{
		ArgumentsInJSON: args,
		Decision:        decision,
	})
}

func (m *approvalMiddleware) defaultDecision(toolName, args, callID string) *common.ApprovalDecision {
	timeout := m.config.DefaultTimeout
	if timeout <= 0 {
		timeout = common.DefaultApprovalTimeout
	}
	expiresAt := time.Now().UTC().Add(time.Duration(timeout) * time.Second)
	approvalID := strings.TrimSpace(callID)
	if approvalID == "" {
		approvalID = fmt.Sprintf("approval-%d", time.Now().UTC().UnixNano())
	}
	return &common.ApprovalDecision{
		RequiresApproval: true,
		ApprovalID:       approvalID,
		Preview:          m.generatePreview(toolName, args),
		TimeoutSeconds:   timeout,
		DecisionSource:   "fallback_static",
		ExpiresAt:        expiresAt,
	}
}

func defaultCommandClass(toolName string) string {
	toolName = strings.ToLower(strings.TrimSpace(toolName))
	switch {
	case strings.Contains(toolName, "delete"),
		strings.Contains(toolName, "scale"),
		strings.Contains(toolName, "restart"),
		strings.Contains(toolName, "rollback"),
		strings.Contains(toolName, "trigger"),
		strings.Contains(toolName, "cancel"),
		strings.Contains(toolName, "update"),
		strings.Contains(toolName, "apply"),
		strings.Contains(toolName, "batch"):
		return "write"
	default:
		return ""
	}
}

func commandClassForTool(toolName, args string) string {
	toolName = strings.ToLower(strings.TrimSpace(toolName))
	switch toolName {
	case "host_batch", "host_batch_exec_apply", "host_batch_exec_preview":
		if class := hostBatchCommandClass(args); class != "" {
			return class
		}
		return "service_control"
	case "host_batch_status_update":
		return "service_control"
	default:
		return defaultCommandClass(toolName)
	}
}

func hostBatchCommandClass(args string) string {
	var params map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(args)), &params); err != nil {
		return ""
	}
	cmd, _ := params["command"].(string)
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	class, _, _ := classifyHostCommand(cmd)
	return class
}

func classifyHostCommand(cmd string) (class string, risk string, blocked bool) {
	trimmed := strings.ToLower(strings.TrimSpace(cmd))
	if isReadonlyHostCommand(cmd) {
		return "readonly", "low", false
	}
	dangerous := []string{
		"rm -rf /", "mkfs", "shutdown", "poweroff", "reboot", "init 0",
		"dd if=", "iptables -f", "userdel", "chown -r /", "chmod -r 777 /",
	}
	for _, keyword := range dangerous {
		if strings.Contains(trimmed, keyword) {
			return "dangerous", "high", true
		}
	}
	return "service_control", "medium", false
}

func isReadonlyHostCommand(cmd string) bool {
	switch strings.TrimSpace(cmd) {
	case "hostname", "uptime", "df -h", "free -m", "ps aux --sort=-%cpu":
		return true
	default:
		return false
	}
}

// formatDisapproveMessage 格式化拒绝消息。
func (m *approvalMiddleware) formatDisapproveMessage(toolName string, result *common.ApprovalResult) string {
	if result.DisapproveReason != nil && strings.TrimSpace(*result.DisapproveReason) != "" {
		return fmt.Sprintf("tool '%s' disapproved: %s", toolName, *result.DisapproveReason)
	}
	return fmt.Sprintf("tool '%s' disapproved by user", toolName)
}

// singleChunkReader 创建单次读取的 StreamReader。
//
// 用于在审批拒绝时返回简单的字符串消息。
func singleChunkReader(content string) *schema.StreamReader[string] {
	sr, sw := schema.Pipe[string](1)
	go func() {
		defer sw.Close()
		sw.Send(content, nil)
	}()
	return sr
}

// =============================================================================
// compose.ToolMiddleware 适配器
// =============================================================================

// AsToolMiddleware 将审批中间件转换为 compose.ToolMiddleware 格式。
//
// 用于在 ToolsConfig.ToolCallMiddlewares 中注册审批中间件。
//
// 使用示例:
//
//	mw := ApprovalMiddleware(cfg)
//	toolMW := AsToolMiddleware(mw)
//	toolsConfig := adk.ToolsConfig{
//	    ToolsNodeConfig: compose.ToolsNodeConfig{
//	        Tools: toolset,
//	        ToolCallMiddlewares: []compose.ToolMiddleware{toolMW},
//	    },
//	}
func (m *approvalMiddleware) AsToolMiddleware() compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable:  m.wrapInvokableForCompose,
		Streamable: m.wrapStreamableForCompose,
	}
}

// wrapInvokableForCompose 为 compose.ToolMiddleware 适配同步工具调用。
func (m *approvalMiddleware) wrapInvokableForCompose(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
	return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
		// 检查是否需要审批
		if !m.requiresApprovalGate(input.Name) {
			return next(ctx, input)
		}

		decision, storedArgs, wasInterrupted, err := m.evaluateApproval(ctx, input.Name, input.Arguments, input.CallID)
		if err != nil {
			return nil, err
		}

		if !wasInterrupted {
			if decision == nil || !decision.RequiresApproval {
				return next(ctx, input)
			}
			return nil, m.raiseApprovalInterrupt(ctx, input.Name, input.CallID, input.Arguments, decision)
		}

		// 已中断过，检查恢复上下文
		isTarget, hasData, result := tool.GetResumeContext[*common.ApprovalResult](ctx)

		if isTarget && hasData {
			if result.Approved {
				// 审批通过，执行原始工具
				return next(ctx, &compose.ToolInput{
					Name:        input.Name,
					Arguments:   storedArgs,
					CallID:      input.CallID,
					CallOptions: input.CallOptions,
				})
			}
			// 审批拒绝
			return &compose.ToolOutput{
				Result: m.formatDisapproveMessage(input.Name, result),
			}, nil
		}

		// 恢复上下文不匹配，重新中断
		if decision == nil {
			decision = m.defaultDecision(input.Name, storedArgs, input.CallID)
		}
		return nil, m.raiseApprovalInterrupt(ctx, input.Name, input.CallID, storedArgs, decision)
	}
}

// wrapStreamableForCompose 为 compose.ToolMiddleware 适配流式工具调用。
func (m *approvalMiddleware) wrapStreamableForCompose(next compose.StreamableToolEndpoint) compose.StreamableToolEndpoint {
	return func(ctx context.Context, input *compose.ToolInput) (*compose.StreamToolOutput, error) {
		// 检查是否需要审批
		if !m.requiresApprovalGate(input.Name) {
			return next(ctx, input)
		}

		decision, storedArgs, wasInterrupted, err := m.evaluateApproval(ctx, input.Name, input.Arguments, input.CallID)
		if err != nil {
			return nil, err
		}

		if !wasInterrupted {
			if decision == nil || !decision.RequiresApproval {
				return next(ctx, input)
			}
			return nil, m.raiseApprovalInterrupt(ctx, input.Name, input.CallID, input.Arguments, decision)
		}

		// 已中断过，检查恢复上下文
		isTarget, hasData, result := tool.GetResumeContext[*common.ApprovalResult](ctx)

		if isTarget && hasData {
			if result.Approved {
				// 审批通过，执行原始工具
				return next(ctx, &compose.ToolInput{
					Name:        input.Name,
					Arguments:   storedArgs,
					CallID:      input.CallID,
					CallOptions: input.CallOptions,
				})
			}
			// 审批拒绝
			return &compose.StreamToolOutput{
				Result: singleChunkReader(m.formatDisapproveMessage(input.Name, result)),
			}, nil
		}

		// 恢复上下文不匹配，重新中断
		if decision == nil {
			decision = m.defaultDecision(input.Name, storedArgs, input.CallID)
		}
		return nil, m.raiseApprovalInterrupt(ctx, input.Name, input.CallID, storedArgs, decision)
	}
}

// AsToolMiddlewares 将审批中间件列表转换为 compose.ToolMiddleware 列表。
//
// 用于在 ToolsConfig.ToolCallMiddlewares 中注册多个中间件。
func AsToolMiddlewares(middlewares ...adk.ChatModelAgentMiddleware) []compose.ToolMiddleware {
	result := make([]compose.ToolMiddleware, 0, len(middlewares))
	for _, mw := range middlewares {
		if am, ok := mw.(*approvalMiddleware); ok {
			result = append(result, am.AsToolMiddleware())
		}
	}
	return result
}

// =============================================================================
// 默认配置
// =============================================================================

// DefaultNeedsApproval 默认审批判断逻辑。
//
// 返回给定工具是否需要人工审批。
// 高风险操作（批量执行、删除、重启等）需要审批。
func DefaultNeedsApproval(toolName string) bool {
	// 需要审批的高风险工具列表
	approvalRequired := map[string]bool{
		// 主机操作 - 批量执行和状态变更
		"host_batch":               true,
		"host_batch_exec_apply":    true,
		"host_batch_status_update": true,

		// K8s 操作 - 变更类操作
		"k8s_scale_deployment":    true,
		"k8s_restart_deployment":  true,
		"k8s_delete_pod":          true,
		"k8s_rollback_deployment": true,
		"k8s_delete_deployment":   true,

		// CI/CD 操作 - 触发流水线
		"cicd_trigger_pipeline": true,
		"cicd_cancel_pipeline":  true,

		// 服务操作 - 变更类操作
		"service_restart":       true,
		"service_scale":         true,
		"service_update_config": true,
	}
	return approvalRequired[toolName]
}

// DefaultPreviewGenerator 默认预览生成器。
//
// 根据工具名称和参数生成审批预览信息。
// 特定工具应该注册自己的预览生成器以提供更详细的信息。
func DefaultPreviewGenerator(toolName, args string) common.ApprovalPreview {
	preview := common.ApprovalPreview{
		Action:    toolName,
		RiskLevel: common.RiskLevelMedium,
	}

	// 解析参数
	var params map[string]any
	if err := json.Unmarshal([]byte(args), &params); err == nil {
		// 提取目标信息
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

		// 提取操作信息
		if cmd, ok := params["command"].(string); ok {
			preview.Action = cmd
		}
		if action, ok := params["action"].(string); ok {
			preview.Action = action
		}
	}

	// 根据工具名称设置风险等级和影响描述
	switch toolName {
	case "host_batch", "host_batch_exec_apply":
		preview.RiskLevel = common.RiskLevelHigh
		preview.Impact = "命令将在多个主机上执行，请确认影响范围"
		if strings.Contains(preview.Action, "rm ") {
			preview.RiskLevel = common.RiskLevelCritical
			preview.Warnings = append(preview.Warnings, "命令包含删除操作，请仔细核对")
		}

	case "host_batch_status_update":
		preview.RiskLevel = common.RiskLevelMedium
		preview.Impact = "主机状态变更可能影响自动化运维流程"

	case "k8s_delete_pod":
		preview.RiskLevel = common.RiskLevelHigh
		preview.Impact = "Pod 将被删除，控制器可能会重建新 Pod，可能导致短暂服务中断"
		preview.Warnings = append(preview.Warnings, "删除 Pod 不会影响 Deployment 的副本数")

	case "k8s_delete_deployment":
		preview.RiskLevel = common.RiskLevelCritical
		preview.Impact = "Deployment 将被永久删除，服务将停止"
		preview.Warnings = append(preview.Warnings, "此操作不可逆，请确认是否真的需要删除")

	case "k8s_restart_deployment":
		preview.RiskLevel = common.RiskLevelMedium
		preview.Impact = "Deployment 将滚动重启，可能导致短暂的服务不稳定"

	case "k8s_scale_deployment":
		preview.RiskLevel = common.RiskLevelMedium
		preview.Impact = "副本数变更将影响服务容量和资源消耗"

	case "k8s_rollback_deployment":
		preview.RiskLevel = common.RiskLevelMedium
		preview.Impact = "Deployment 将回滚到上一版本，可能导致功能变更"

	case "cicd_trigger_pipeline":
		preview.RiskLevel = common.RiskLevelMedium
		preview.Impact = "将触发 CI/CD 流水线执行，可能影响部署环境"

	case "cicd_cancel_pipeline":
		preview.RiskLevel = common.RiskLevelLow
		preview.Impact = "将取消正在运行的流水线"
	}

	return preview
}

// DefaultToolConfigs 默认工具配置。
//
// 返回各工具的风险配置映射。
func DefaultToolConfigs() map[string]*common.ToolRiskConfig {
	return map[string]*common.ToolRiskConfig{
		"host_batch": {
			ToolName:         "host_batch",
			RiskLevel:        common.RiskLevelHigh,
			NeedsApproval:    true,
			PreviewGenerator: hostBatchPreviewGenerator,
		},
		"host_batch_exec_apply": {
			ToolName:         "host_batch_exec_apply",
			RiskLevel:        common.RiskLevelHigh,
			NeedsApproval:    true,
			PreviewGenerator: hostBatchPreviewGenerator,
		},
		"k8s_delete_pod": {
			ToolName:         "k8s_delete_pod",
			RiskLevel:        common.RiskLevelHigh,
			NeedsApproval:    true,
			PreviewGenerator: k8sPodPreviewGenerator,
		},
		"k8s_restart_deployment": {
			ToolName:         "k8s_restart_deployment",
			RiskLevel:        common.RiskLevelMedium,
			NeedsApproval:    true,
			PreviewGenerator: k8sDeploymentPreviewGenerator,
		},
		"k8s_scale_deployment": {
			ToolName:         "k8s_scale_deployment",
			RiskLevel:        common.RiskLevelMedium,
			NeedsApproval:    true,
			PreviewGenerator: k8sScalePreviewGenerator,
		},
		"k8s_rollback_deployment": {
			ToolName:         "k8s_rollback_deployment",
			RiskLevel:        common.RiskLevelMedium,
			NeedsApproval:    true,
			PreviewGenerator: k8sRollbackPreviewGenerator,
		},
		"k8s_delete_deployment": {
			ToolName:         "k8s_delete_deployment",
			RiskLevel:        common.RiskLevelCritical,
			NeedsApproval:    true,
			PreviewGenerator: k8sDeleteDeploymentPreviewGenerator,
		},
	}
}

// =============================================================================
// 特定工具的预览生成器
// =============================================================================

// hostBatchPreviewGenerator 主机批量执行预览生成器。
func hostBatchPreviewGenerator(args string) common.ApprovalPreview {
	preview := common.ApprovalPreview{
		Action:    "batch_execute",
		RiskLevel: common.RiskLevelHigh,
	}

	var params struct {
		HostIDs []int  `json:"host_ids"`
		Command string `json:"command"`
		Reason  string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(args), &params); err == nil {
		preview.Target = fmt.Sprintf("%d hosts", len(params.HostIDs))
		preview.Action = params.Command
		preview.Extra = map[string]any{
			"host_count": len(params.HostIDs),
			"reason":     params.Reason,
		}

		// 根据命令判断风险
		cmdLower := strings.ToLower(params.Command)
		if strings.Contains(cmdLower, "rm ") ||
			strings.Contains(cmdLower, "delete") ||
			strings.Contains(cmdLower, "shutdown") ||
			strings.Contains(cmdLower, "reboot") {
			preview.RiskLevel = common.RiskLevelCritical
			preview.Warnings = append(preview.Warnings, "命令具有破坏性，请仔细确认")
		}

		preview.Impact = fmt.Sprintf("将在 %d 台主机上执行命令: %s", len(params.HostIDs), params.Command)
	}

	return preview
}

// k8sPodPreviewGenerator K8s Pod 操作预览生成器。
func k8sPodPreviewGenerator(args string) common.ApprovalPreview {
	preview := common.ApprovalPreview{
		Action:    "delete_pod",
		RiskLevel: common.RiskLevelHigh,
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

// k8sDeploymentPreviewGenerator K8s Deployment 操作预览生成器。
func k8sDeploymentPreviewGenerator(args string) common.ApprovalPreview {
	preview := common.ApprovalPreview{
		Action:    "restart_deployment",
		RiskLevel: common.RiskLevelMedium,
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

// k8sScalePreviewGenerator K8s 扩缩容预览生成器。
func k8sScalePreviewGenerator(args string) common.ApprovalPreview {
	preview := common.ApprovalPreview{
		Action:    "scale_deployment",
		RiskLevel: common.RiskLevelMedium,
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

// k8sRollbackPreviewGenerator K8s 回滚预览生成器。
func k8sRollbackPreviewGenerator(args string) common.ApprovalPreview {
	preview := common.ApprovalPreview{
		Action:    "rollback_deployment",
		RiskLevel: common.RiskLevelMedium,
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

// k8sDeleteDeploymentPreviewGenerator K8s 删除 Deployment 预览生成器。
func k8sDeleteDeploymentPreviewGenerator(args string) common.ApprovalPreview {
	preview := common.ApprovalPreview{
		Action:    "delete_deployment",
		RiskLevel: common.RiskLevelCritical,
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
