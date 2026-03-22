package logic

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

const toolFailureCircuitBreakerThreshold = 2

type recoverableToolInvocationError struct {
	CallID   string
	ToolName string
	Message  string
}

type recoverableToolResult struct {
	Event *adk.AgentEvent
	Info  *recoverableToolInvocationError
}

type toolFailureKey struct {
	ToolName  string
	ArgsShape string
}

type toolFailureTracker struct {
	counts     map[toolFailureKey]int
	callShapes map[string]toolFailureKey
}

var (
	streamToolCallErrPattern = regexp.MustCompile(`failed to stream tool call (\S+): .*toolName=([^,\s]+), err=(.+)`)
	invokeToolErrPattern     = regexp.MustCompile(`failed to invoke tool\[name:([^\s\]]+) id:([^\s\]]+)\]: (.+)`)
)

func newToolFailureTracker() *toolFailureTracker {
	return &toolFailureTracker{
		counts:     make(map[toolFailureKey]int),
		callShapes: make(map[string]toolFailureKey),
	}
}

func classifyRecoverableToolInvocationError(err error) (*recoverableToolInvocationError, bool) {
	if err == nil {
		return nil, false
	}

	trimmed := strings.TrimSpace(err.Error())
	if trimmed == "" {
		return nil, false
	}

	if matches := streamToolCallErrPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		callID := strings.TrimSpace(matches[1])
		toolName := strings.TrimSpace(matches[2])
		if callID == "" || toolName == "" {
			return nil, false
		}
		return &recoverableToolInvocationError{
			CallID:   callID,
			ToolName: toolName,
			Message:  trimmed,
		}, true
	}

	if matches := invokeToolErrPattern.FindStringSubmatch(trimmed); len(matches) == 4 {
		toolName := strings.TrimSpace(matches[1])
		callID := strings.TrimSpace(matches[2])
		if callID == "" || toolName == "" {
			return nil, false
		}
		return &recoverableToolInvocationError{
			CallID:   callID,
			ToolName: toolName,
			Message:  trimmed,
		}, true
	}

	return nil, false
}

func buildSyntheticToolResultEvent(agentName string, info *recoverableToolInvocationError) (*adk.AgentEvent, bool) {
	if info == nil {
		return nil, false
	}

	payload, err := json.Marshal(map[string]any{
		"ok":         false,
		"status":     "error",
		"tool_name":  info.ToolName,
		"call_id":    info.CallID,
		"message":    info.Message,
		"error_type": "tool_invocation",
	})
	if err != nil {
		return nil, false
	}

	message := schema.ToolMessage(string(payload), info.CallID, schema.WithToolName(info.ToolName))
	synthetic := adk.EventFromMessage(message, nil, schema.Tool, info.ToolName)
	synthetic.AgentName = agentName
	return synthetic, true
}

func recoverableToolErrorFromErr(err error, agentName string) (*recoverableToolResult, bool) {
	info, ok := classifyRecoverableToolInvocationError(err)
	if !ok {
		return nil, false
	}

	event, ok := buildSyntheticToolResultEvent(agentName, info)
	if !ok {
		return nil, false
	}
	event.Err = err
	return &recoverableToolResult{
		Event: event,
		Info:  info,
	}, true
}

func recoverableToolErrorFromEvent(event *adk.AgentEvent) (*recoverableToolResult, bool) {
	if event == nil || event.Err == nil || isBusinessToolResultEvent(event) {
		return nil, false
	}
	return recoverableToolErrorFromErr(event.Err, event.AgentName)
}

func isBusinessToolResultEvent(event *adk.AgentEvent) bool {
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
		callID := strings.TrimSpace(stringValue(data, "call_id"))
		toolName := strings.TrimSpace(stringValue(data, "tool_name"))
		if callID == "" && toolName == "" {
			continue
		}
		key := toolFailureKey{
			ToolName:  toolName,
			ArgsShape: toolFailureArgsShape(data["arguments"]),
		}
		if callID != "" {
			t.callShapes[callID] = key
		}
	}
}

func (t *toolFailureTracker) recordFailure(info *recoverableToolInvocationError) (toolFailureKey, int, bool) {
	if t == nil || info == nil {
		return toolFailureKey{}, 0, false
	}

	key := t.keyFor(info)
	if strings.TrimSpace(key.ToolName) == "" {
		return toolFailureKey{}, 0, false
	}

	t.counts[key]++
	count := t.counts[key]
	return key, count, count >= toolFailureCircuitBreakerThreshold
}

func (t *toolFailureTracker) keyFor(info *recoverableToolInvocationError) toolFailureKey {
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
