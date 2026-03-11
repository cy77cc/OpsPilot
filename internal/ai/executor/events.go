// Package executor 实现 AI 编排的执行阶段。
//
// 本文件定义执行阶段的事件发射函数，用于向前端推送实时状态。
package executor

import (
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

// emitStepUpdate 发送步骤更新事件。
func emitStepUpdate(req Request, state *runtime.ExecutionState, step runtime.StepState) {
	if req.EmitEvent == nil {
		return
	}
	meta := req.EventMeta.WithDefaults()
	meta.SessionID = firstNonEmpty(req.SessionID, meta.SessionID)
	meta.TraceID = firstNonEmpty(req.TraceID, meta.TraceID)
	meta.PlanID = firstNonEmpty(statePlanID(state), meta.PlanID)
	meta.StepID = step.StepID
	req.EmitEvent("step_update", meta, map[string]any{
		"session_id":           meta.SessionID,
		"trace_id":             meta.TraceID,
		"plan_id":              meta.PlanID,
		"step_id":              step.StepID,
		"status":               step.Status,
		"title":                step.Title,
		"expert":               step.Expert,
		"user_visible_summary": step.UserVisibleSummary,
		"error_code":           step.ErrorCode,
		"error_message":        step.ErrorMessage,
	})
}

// emitToolCall 发送工具调用事件。
func emitToolCall(req Request, state *runtime.ExecutionState, step runtime.StepState, toolName, summary string) {
	if req.EmitEvent == nil {
		return
	}
	meta := req.EventMeta.WithDefaults()
	meta.SessionID = firstNonEmpty(req.SessionID, meta.SessionID)
	meta.TraceID = firstNonEmpty(req.TraceID, meta.TraceID)
	meta.PlanID = firstNonEmpty(statePlanID(state), meta.PlanID)
	meta.StepID = step.StepID
	req.EmitEvent("tool_call", meta, map[string]any{
		"session_id": meta.SessionID,
		"trace_id":   meta.TraceID,
		"plan_id":    meta.PlanID,
		"step_id":    step.StepID,
		"expert":     step.Expert,
		"call_id":    fmt.Sprintf("%s:%s:%s", meta.PlanID, step.StepID, strings.TrimSpace(toolName)),
		"tool_name":  strings.TrimSpace(toolName),
		"status":     "running",
		"summary":    strings.TrimSpace(summary),
		"ts":         time.Now().UTC().Format(time.RFC3339),
	})
}

// emitToolResult 发送工具调用结果事件。
func emitToolResult(req Request, state *runtime.ExecutionState, step runtime.StepState, toolName, status, summary, errMsg string, result map[string]any) {
	if req.EmitEvent == nil {
		return
	}
	meta := req.EventMeta.WithDefaults()
	meta.SessionID = firstNonEmpty(req.SessionID, meta.SessionID)
	meta.TraceID = firstNonEmpty(req.TraceID, meta.TraceID)
	meta.PlanID = firstNonEmpty(statePlanID(state), meta.PlanID)
	meta.StepID = step.StepID
	payload := map[string]any{
		"session_id": meta.SessionID,
		"trace_id":   meta.TraceID,
		"plan_id":    meta.PlanID,
		"step_id":    step.StepID,
		"expert":     step.Expert,
		"call_id":    fmt.Sprintf("%s:%s:%s", meta.PlanID, step.StepID, strings.TrimSpace(toolName)),
		"tool_name":  strings.TrimSpace(toolName),
		"status":     strings.TrimSpace(status),
		"summary":    strings.TrimSpace(summary),
		"ts":         time.Now().UTC().Format(time.RFC3339),
	}
	if strings.TrimSpace(errMsg) != "" {
		payload["error"] = strings.TrimSpace(errMsg)
	}
	if result != nil {
		payload["result"] = result
	}
	req.EmitEvent("tool_result", meta, payload)
}

// emitApprovalRequired 发送审批请求事件。
func emitApprovalRequired(req Request, state *runtime.ExecutionState, approval *runtime.PendingApproval, step runtime.StepState) {
	if req.EmitEvent == nil || approval == nil {
		return
	}
	meta := req.EventMeta.WithDefaults()
	meta.SessionID = firstNonEmpty(req.SessionID, meta.SessionID)
	meta.TraceID = firstNonEmpty(req.TraceID, meta.TraceID)
	meta.PlanID = firstNonEmpty(approval.PlanID, statePlanID(state), meta.PlanID)
	meta.StepID = approval.StepID
	req.EmitEvent("approval_required", meta, map[string]any{
		"session_id":           meta.SessionID,
		"trace_id":             meta.TraceID,
		"plan_id":              meta.PlanID,
		"step_id":              approval.StepID,
		"title":                approval.Title,
		"risk":                 approval.Risk,
		"mode":                 approval.Mode,
		"status":               approval.Status,
		"user_visible_summary": firstNonEmpty(approval.Summary, step.UserVisibleSummary),
	})
}

// statePlanID 从执行状态中获取计划 ID。
func statePlanID(state *runtime.ExecutionState) string {
	if state == nil {
		return ""
	}
	return strings.TrimSpace(state.PlanID)
}
