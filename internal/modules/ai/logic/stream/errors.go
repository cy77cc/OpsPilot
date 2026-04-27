// Package stream 实现工具错误分类和失败追踪。
package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
)

const toolFailureCircuitBreakerThreshold = 2

// RecoverableToolInvocationError 定义可恢复的工具调用错误。
type RecoverableToolInvocationError struct {
	CallID   string
	ToolName string
	Message  string
}

// RecoverableToolResult 定义可恢复的工具结果。
type RecoverableToolResult struct {
	Event *adk.AgentEvent
	Info  *RecoverableToolInvocationError
}

type toolFailureKey struct{ ToolName, ArgsShape string }

type toolFailureTracker struct {
	counts     map[toolFailureKey]int
	callShapes map[string]toolFailureKey
	threshold  int
}

var (
	streamToolCallErrPattern = regexp.MustCompile(`failed to stream tool call (\S+): .*toolName=([^,\s]+), err=(.+)`)
	invokeToolErrPattern     = regexp.MustCompile(`failed to invoke tool\[name:([^\s\]]+) id:([^\s\]]+)\]: (.+)`)
	genericCallIDPattern     = regexp.MustCompile(`(?:tool call|id[:=]|call_id[=:])\s*([A-Za-z0-9._:-]+)`)
	genericToolNamePattern   = regexp.MustCompile(`(?:toolName=|tool_name=|name:)\s*([A-Za-z0-9._:-]+)`)
)

func newToolFailureTracker(threshold int) *toolFailureTracker {
	if threshold <= 0 {
		threshold = toolFailureCircuitBreakerThreshold
	}
	return &toolFailureTracker{counts: make(map[toolFailureKey]int), callShapes: make(map[string]toolFailureKey), threshold: threshold}
}

// ClassifyRecoverableToolInvocationError 分类可恢复的工具调用错误。
func ClassifyRecoverableToolInvocationError(err error) (*RecoverableToolInvocationError, bool) {
	if err == nil {
		return nil, false
	}
	trimmed := strings.TrimSpace(err.Error())
	if trimmed == "" {
		return nil, false
	}
	if matches := streamToolCallErrPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		callID, toolName := strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
		if callID == "" || toolName == "" {
			return nil, false
		}
		return &RecoverableToolInvocationError{CallID: callID, ToolName: toolName, Message: trimmed}, true
	}
	if matches := invokeToolErrPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		toolName, callID := strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
		if callID == "" || toolName == "" {
			return nil, false
		}
		return &RecoverableToolInvocationError{CallID: callID, ToolName: toolName, Message: trimmed}, true
	}
	lower := strings.ToLower(trimmed)
	if !strings.Contains(lower, "failed to invoke tool") && !strings.Contains(lower, "failed to stream tool call") {
		return nil, false
	}
	callID, toolName := extractGenericToolInvocationIdentity(trimmed)
	if callID == "" || toolName == "" {
		return nil, false
	}
	return &RecoverableToolInvocationError{CallID: callID, ToolName: toolName, Message: trimmed}, true
}

func extractGenericToolInvocationIdentity(message string) (string, string) {
	callID, toolName := "", ""
	if matches := genericCallIDPattern.FindStringSubmatch(message); len(matches) == 2 {
		callID = strings.TrimSpace(matches[1])
	}
	if matches := genericToolNamePattern.FindStringSubmatch(message); len(matches) == 2 {
		toolName = strings.TrimSpace(matches[1])
	}
	return callID, toolName
}

// RecoverableToolErrorFromErr 从错误构建可恢复工具错误。
func RecoverableToolErrorFromErr(err error, agentName string) (*RecoverableToolResult, bool) {
	info, ok := ClassifyRecoverableToolInvocationError(err)
	if !ok {
		return nil, false
	}
	event, ok := BuildSyntheticToolResultEvent(agentName, info)
	if !ok {
		return nil, false
	}
	event.Err = err
	return &RecoverableToolResult{Event: event, Info: info}, true
}

// RecoverableToolErrorFromEvent 从事件提取可恢复工具错误。
func RecoverableToolErrorFromEvent(event *adk.AgentEvent) (*RecoverableToolResult, bool) {
	if event == nil || event.Err == nil || IsBusinessToolResultEvent(event) {
		return nil, false
	}
	return RecoverableToolErrorFromErr(event.Err, event.AgentName)
}

// RecoverableInterruptEventFromEvent 从事件提取可恢复中断。
func RecoverableInterruptEventFromEvent(event *adk.AgentEvent) (*adk.AgentEvent, bool) {
	if event == nil {
		return nil, false
	}
	if event.Action != nil && event.Action.Interrupted != nil {
		if payload, ok := interruptPayloadFromInfo(event.Action.Interrupted); ok {
			event.Action.Interrupted.Data = payload
		}
		return event, true
	}
	if event.Err == nil {
		return nil, false
	}
	return RecoverableInterruptEventFromErr(event.Err, event.AgentName)
}

// RecoverableInterruptEventFromErr 从错误提取可恢复中断。
func RecoverableInterruptEventFromErr(err error, agentName string) (*adk.AgentEvent, bool) {
	if err == nil {
		return nil, false
	}
	var signal *adk.InterruptSignal
	if !errors.As(err, &signal) || signal == nil {
		return nil, false
	}
	payload, ok := signal.InterruptInfo.Info.(map[string]any)
	if !ok || payload == nil {
		return nil, false
	}
	copied := make(map[string]any, len(payload))
	for k, v := range payload {
		copied[k] = v
	}
	return &adk.AgentEvent{AgentName: agentName, Action: &adk.AgentAction{Interrupted: &adk.InterruptInfo{Data: copied, InterruptContexts: interruptContextsFromSignal(signal)}}}, true
}

// IsBusinessToolResultEvent 判断事件是否为业务工具结果。
func IsBusinessToolResultEvent(event *adk.AgentEvent) bool {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return false
	}
	if event.Output.MessageOutput.Role != schema.Tool {
		return false
	}
	normalized := airuntime.NormalizeAgentEvent(event)
	for _, item := range normalized {
		if item.Kind == airuntime.NormalizedKindToolResult {
			return true
		}
	}
	return false
}

func (t *toolFailureTracker) recordProjectedEvents(events []airuntime.PublicStreamEvent) {
	if t == nil {
		return
	}
	for _, projected := range events {
		if projected.Event != "tool_call" {
			continue
		}
		data, ok := projected.Data.(map[string]any)
		if !ok {
			continue
		}
		callID := strings.TrimSpace(StringValue(data, "call_id"))
		toolName := strings.TrimSpace(StringValue(data, "tool_name"))
		if callID == "" && toolName == "" {
			continue
		}
		key := toolFailureKey{ToolName: toolName, ArgsShape: toolFailureArgsShape(data["arguments"])}
		if callID != "" {
			t.callShapes[callID] = key
		}
	}
}

func (t *toolFailureTracker) recordFailure(info *RecoverableToolInvocationError) (toolFailureKey, int, bool) {
	if t == nil || info == nil {
		return toolFailureKey{}, 0, false
	}
	key := t.keyFor(info)
	if strings.TrimSpace(key.ToolName) == "" {
		return toolFailureKey{}, 0, false
	}
	t.counts[key]++
	count := t.counts[key]
	return key, count, count >= t.threshold
}

func (t *toolFailureTracker) keyFor(info *RecoverableToolInvocationError) toolFailureKey {
	if t == nil || info == nil {
		return toolFailureKey{}
	}
	callID := strings.TrimSpace(info.CallID)
	if callID != "" {
		if key, ok := t.callShapes[callID]; ok && strings.TrimSpace(key.ToolName) != "" {
			return key
		}
	}
	toolName := strings.TrimSpace(info.ToolName)
	if toolName == "" {
		return toolFailureKey{}
	}
	return toolFailureKey{ToolName: toolName}
}

func toolFailureArgsShape(arguments any) string {
	switch typed := arguments.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%T", typed)
		}
		return string(raw)
	}
}

// BuildSyntheticToolResultEvent 构建合成工具结果事件。
func BuildSyntheticToolResultEvent(agentName string, info *RecoverableToolInvocationError) (*adk.AgentEvent, bool) {
	if info == nil {
		return nil, false
	}
	payload, err := json.Marshal(map[string]any{"ok": false, "status": "error", "tool_name": info.ToolName, "call_id": info.CallID, "message": info.Message, "error_type": "tool_invocation"})
	if err != nil {
		return nil, false
	}
	message := schema.ToolMessage(string(payload), info.CallID, schema.WithToolName(info.ToolName))
	synthetic := adk.EventFromMessage(message, nil, schema.Tool, info.ToolName)
	synthetic.AgentName = agentName
	return synthetic, true
}

func interruptContextsFromSignal(signal *adk.InterruptSignal) []*adk.InterruptCtx {
	if signal == nil {
		return nil
	}
	rootCauses := make([]*adk.InterruptCtx, 0, 1)
	var build func(cur *adk.InterruptSignal, parent *adk.InterruptCtx)
	build = func(cur *adk.InterruptSignal, parent *adk.InterruptCtx) {
		if cur == nil {
			return
		}
		ctx := &adk.InterruptCtx{ID: cur.ID, Address: cur.Address, Info: cur.InterruptInfo.Info, IsRootCause: cur.InterruptInfo.IsRootCause, Parent: parent}
		if ctx.IsRootCause {
			rootCauses = append(rootCauses, ctx)
		}
		for _, child := range cur.Subs {
			build(child, ctx)
		}
	}
	build(signal, nil)
	return rootCauses
}

func interruptPayloadFromInfo(interrupted *adk.InterruptInfo) (map[string]any, bool) {
	if interrupted == nil {
		return nil, false
	}
	if payload, ok := interrupted.Data.(map[string]any); ok && payload != nil {
		return payload, true
	}
	for _, ctx := range interrupted.InterruptContexts {
		if ctx == nil {
			continue
		}
		if payload, ok := ctx.Info.(map[string]any); ok && payload != nil {
			return payload, true
		}
	}
	return nil, false
}
