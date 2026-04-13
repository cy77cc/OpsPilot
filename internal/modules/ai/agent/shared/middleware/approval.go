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
	"encoding/gob"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"
	host "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/hostpolicy"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

// ApprovalMiddlewareConfig 审批中间件配置。
type ApprovalMiddlewareConfig struct {
	// Orchestrator evaluates DB-driven approval policy and creates approval snapshots.
	Orchestrator approval.ApprovalEvaluator

	// NeedsApproval 判断工具是否需要审批
	// 返回 true 表示该工具需要人工审批后才能执行
	NeedsApproval func(toolName string) bool

	// PreviewGenerator 生成审批预览信息
	// 用于在前端展示操作的详细信息和潜在影响
	PreviewGenerator func(toolName, args string) approval.ApprovalPreview

	// DefaultTimeout 默认审批超时时间（秒）
	// 超时后审批请求将自动失效
	DefaultTimeout int

	// ToolConfigs 特定工具的配置映射
	// key 为工具名称，value 为该工具的风险配置
	ToolConfigs map[string]*approval.ToolRiskConfig
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
		cfg.DefaultTimeout = approval.DefaultApprovalTimeout
	}
	if cfg.ToolConfigs == nil {
		cfg.ToolConfigs = DefaultToolConfigs()
	}

	return &approvalMiddleware{
		config: cfg,
	}
}

// approvalMiddleware 审批中间件实现。
type approvalMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	config *ApprovalMiddlewareConfig
}

type approvalInterruptState struct {
	ArgumentsInJSON string
	Decision        *approval.ApprovalDecision
	SessionID       string
	AgentRole       string
}

func init() {
	schema.RegisterName[approvalInterruptState]("_opspilot_approval_interrupt_state")
	gob.Register(map[string]any{})
	gob.Register([]any{})
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
		isTarget, hasData, result := tool.GetResumeContext[*approval.ApprovalResult](ctx)

		if isTarget && hasData {
			if result.Approved && m.resumeBindingMatches(ctx, decision) {
				// 审批通过，执行原始工具
				return endpoint(ctx, storedArgs, opts...)
			}
			if result.Approved {
				if decision == nil {
					decision = m.defaultDecision(ctx, tCtx.Name, storedArgs, tCtx.CallID)
				}
				return "", m.raiseApprovalInterrupt(ctx, tCtx.Name, tCtx.CallID, storedArgs, decision)
			}
			// 审批拒绝
			return m.formatDisapproveMessage(tCtx.Name, result), nil
		}

		// 恢复上下文不匹配（可能是其他工具的中断），重新中断
		if decision == nil {
			decision = m.defaultDecision(ctx, tCtx.Name, storedArgs, tCtx.CallID)
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

		isTarget, hasData, result := tool.GetResumeContext[*approval.ApprovalResult](ctx)

		if isTarget && hasData {
			if result.Approved && m.resumeBindingMatches(ctx, decision) {
				return endpoint(ctx, storedArgs, opts...)
			}
			if result.Approved {
				if decision == nil {
					decision = m.defaultDecision(ctx, tCtx.Name, storedArgs, tCtx.CallID)
				}
				return nil, m.raiseApprovalInterrupt(ctx, tCtx.Name, tCtx.CallID, storedArgs, decision)
			}
			return singleChunkReader(m.formatDisapproveMessage(tCtx.Name, result)), nil
		}

		if decision == nil {
			decision = m.defaultDecision(ctx, tCtx.Name, storedArgs, tCtx.CallID)
		}
		return nil, m.raiseApprovalInterrupt(ctx, tCtx.Name, tCtx.CallID, storedArgs, decision)
	}, nil
}

// generatePreview 生成审批预览信息。
//
// 优先使用工具特定配置的生成器，否则使用默认生成器。
func (m *approvalMiddleware) generatePreview(toolName, args string) approval.ApprovalPreview {
	// 检查是否有工具特定配置
	if cfg, ok := m.config.ToolConfigs[toolName]; ok && cfg.PreviewGenerator != nil {
		return cfg.PreviewGenerator(args)
	}
	// 使用默认生成器
	return m.config.PreviewGenerator(toolName, args)
}

func (m *approvalMiddleware) requiresApprovalGate(toolName string) bool {
	if isApprovalBypassTool(toolName) {
		return false
	}
	if m.config.Orchestrator != nil {
		return true
	}
	return m.config.NeedsApproval(toolName)
}

func isApprovalBypassTool(toolName string) bool {
	switch strings.ToLower(strings.TrimSpace(toolName)) {
	case "task":
		return true
	default:
		return false
	}
}

func (m *approvalMiddleware) evaluateApproval(ctx context.Context, toolName, args, callID string) (*approval.ApprovalDecision, string, bool, error) {
	if m.config.Orchestrator == nil {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if wasInterrupted {
			return m.defaultDecision(ctx, toolName, storedArgs, callID), storedArgs, true, nil
		}
		if !m.config.NeedsApproval(toolName) {
			return &approval.ApprovalDecision{RequiresApproval: false}, storedArgs, false, nil
		}
		return m.defaultDecision(ctx, toolName, args, callID), args, false, nil
	}

	wasInterrupted, hasDecisionState, state := tool.GetInterruptState[approvalInterruptState](ctx)
	if wasInterrupted && hasDecisionState {
		storedArgs := strings.TrimSpace(state.ArgumentsInJSON)
		if storedArgs == "" {
			storedArgs = args
		}
		if state.Decision == nil {
			return m.defaultDecision(ctx, toolName, storedArgs, callID), storedArgs, true, nil
		}
		if state.Decision != nil {
			if state.Decision.BoundSessionID == "" {
				state.Decision.BoundSessionID = strings.TrimSpace(state.SessionID)
			}
			if state.Decision.BoundAgentRole == "" {
				state.Decision.BoundAgentRole = strings.TrimSpace(state.AgentRole)
			}
		}
		return state.Decision, storedArgs, true, nil
	}
	if wasInterrupted {
		_, hasStringState, storedArgs := tool.GetInterruptState[string](ctx)
		if hasStringState {
			if strings.TrimSpace(storedArgs) == "" {
				storedArgs = args
			}
			return m.defaultDecision(ctx, toolName, storedArgs, callID), storedArgs, true, nil
		}
		return m.defaultDecision(ctx, toolName, args, callID), args, true, nil
	}

	sceneMeta := runtimectx.AIMetadataFrom(ctx)
	agentRole := currentAgentRole(ctx)
	meta := approval.ApprovalEvalMeta{
		SessionID:      sceneMeta.SessionID,
		RunID:          sceneMeta.RunID,
		CheckpointID:   sceneMeta.CheckpointID,
		Scene:          sceneMeta.Scene,
		AgentRole:      agentRole,
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
	if decision.BoundSessionID == "" {
		decision.BoundSessionID = strings.TrimSpace(sceneMeta.SessionID)
	}
	if decision.BoundAgentRole == "" {
		decision.BoundAgentRole = agentRole
	}
	if !decision.RequiresApproval {
		if prechecked := hostPolicyPrecheck(toolName, args); prechecked != nil && prechecked.DecisionType == host.DecisionRequireApprovalInterrupt {
			forced := m.defaultDecision(ctx, toolName, args, callID)
			forced.DecisionSource = "host_policy_precheck"
			violations := make([]string, 0, len(prechecked.ReasonCodes)+len(prechecked.Violations))
			violations = append(violations, prechecked.ReasonCodes...)
			for _, v := range prechecked.Violations {
				if item := strings.TrimSpace(v.Type); item != "" {
					violations = append(violations, item)
				}
			}
			forced.PolicyViolations = violations
			return forced, args, false, nil
		}
	}
	return decision, args, false, nil
}

func hostPolicyPrecheck(toolName, args string) *host.PolicyDecision {
	switch strings.ToLower(strings.TrimSpace(toolName)) {
	case "host_exec":
	default:
		return nil
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(args)), &params); err != nil {
		return nil
	}
	cmd, _ := params["command"].(string)
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		script, _ := params["script"].(string)
		cmd = strings.TrimSpace(script)
		if cmd == "" {
			return nil
		}
	}
	target, _ := params["target"].(string)

	engine := host.NewHostCommandPolicyEngine(host.DefaultReadonlyAllowlist())
	decision := engine.Evaluate(host.PolicyInput{
		ToolName:   strings.TrimSpace(toolName),
		Target:     strings.TrimSpace(target),
		CommandRaw: cmd,
	})
	return &decision
}

func (m *approvalMiddleware) raiseApprovalInterrupt(ctx context.Context, toolName, callID, args string, decision *approval.ApprovalDecision) error {
	if decision == nil {
		decision = m.defaultDecision(ctx, toolName, args, callID)
	}
	info := buildApprovalInterruptInfo(toolName, callID, decision)
	sceneMeta := runtimectx.AIMetadataFrom(ctx)
	return tool.StatefulInterrupt(ctx, info, approvalInterruptState{
		ArgumentsInJSON: args,
		Decision:        decision,
		SessionID:       strings.TrimSpace(sceneMeta.SessionID),
		AgentRole:       currentAgentRole(ctx),
	})
}

// BuildApprovalInterruptInfo 构建审批中断信息（导出用于测试）。
func BuildApprovalInterruptInfo(toolName, callID string, decision *approval.ApprovalDecision) map[string]any {
	return buildApprovalInterruptInfo(toolName, callID, decision)
}

func buildApprovalInterruptInfo(toolName, callID string, decision *approval.ApprovalDecision) map[string]any {
	preview := toJSONMap(decision.Preview)
	info := map[string]any{
		"status":          "suspended",
		"approval_id":     decision.ApprovalID,
		"call_id":         callID,
		"tool_name":       toolName,
		"preview":         preview,
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
	if approver := strings.TrimSpace(decision.ApproverID); approver != "" {
		info["approver_id"] = approver
	}
	if decision.ApprovalTimestamp != nil {
		info["approval_timestamp"] = decision.ApprovalTimestamp.UTC().Format(time.RFC3339Nano)
	}
	if reason := strings.TrimSpace(decision.RejectReason); reason != "" {
		info["reject_reason"] = reason
	}
	if len(decision.PolicyViolations) > 0 {
		info["policy_violations"] = decision.PolicyViolations
	}
	return info
}

func toJSONMap(v any) map[string]any {
	payload, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}

	out := map[string]any{}
	if err := json.Unmarshal(payload, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func (m *approvalMiddleware) defaultDecision(ctx context.Context, toolName, args, callID string) *approval.ApprovalDecision {
	timeout := m.config.DefaultTimeout
	if timeout <= 0 {
		timeout = approval.DefaultApprovalTimeout
	}
	expiresAt := time.Now().UTC().Add(time.Duration(timeout) * time.Second)
	approvalID := strings.TrimSpace(callID)
	if approvalID == "" {
		approvalID = fmt.Sprintf("approval-%d", time.Now().UTC().UnixNano())
	}
	sceneMeta := runtimectx.AIMetadataFrom(ctx)
	return &approval.ApprovalDecision{
		RequiresApproval: true,
		ApprovalID:       approvalID,
		Preview:          m.generatePreview(toolName, args),
		TimeoutSeconds:   timeout,
		DecisionSource:   "fallback_static",
		ExpiresAt:        expiresAt,
		BoundSessionID:   strings.TrimSpace(sceneMeta.SessionID),
		BoundAgentRole:   currentAgentRole(ctx),
	}
}

func (m *approvalMiddleware) resumeBindingMatches(ctx context.Context, decision *approval.ApprovalDecision) bool {
	if decision == nil {
		return false
	}
	sceneMeta := runtimectx.AIMetadataFrom(ctx)
	if boundSession := strings.TrimSpace(decision.BoundSessionID); boundSession != "" && strings.TrimSpace(sceneMeta.SessionID) != boundSession {
		return false
	}
	if boundRole := strings.TrimSpace(decision.BoundAgentRole); boundRole != "" && currentAgentRole(ctx) != boundRole {
		return false
	}
	return true
}

func currentAgentRole(ctx context.Context) string {
	runtime := runtimectx.FromContext(ctx)
	if runtime == nil {
		return ""
	}
	return strings.TrimSpace(runtime.Role)
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
	case "host_exec":
		return unknownWhenEmpty(hostExecCommandClass(args))
	case "host_batch_status_update":
		return "service_control"
	default:
		return unknownWhenEmpty(defaultCommandClass(toolName))
	}
}

func hostExecCommandClass(args string) string {
	var params map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(args)), &params); err != nil {
		return ""
	}
	if script, _ := params["script"].(string); strings.TrimSpace(script) != "" {
		return ""
	}
	return hostCommandClassFromMap(params)
}

func hostCommandClassFromMap(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	cmd, _ := params["command"].(string)
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		script, _ := params["script"].(string)
		cmd = strings.TrimSpace(script)
		if cmd == "" {
			return "unknown"
		}
	}
	class, _, _ := classifyHostCommand(cmd)
	return class
}

func unknownWhenEmpty(class string) string {
	if strings.TrimSpace(class) == "" {
		return "unknown"
	}
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
func (m *approvalMiddleware) formatDisapproveMessage(toolName string, result *approval.ApprovalResult) string {
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

// 默认配置与预览生成器拆分到 approval_defaults.go。
