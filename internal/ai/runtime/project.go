package runtime

import (
	"fmt"
	"strings"
)

// ProjectionState 跟踪流式投影状态。
//
// 包含两部分状态：
//   - 解析状态：用于增量解析 SSE 事件（PendingPlannerJSON 等）
//   - 持久化状态：用于存储到数据库（Persisted）
type ProjectionState struct {
	TotalPlanSteps         int
	ReplanIteration        int
	RunPhase               string
	PendingPlannerJSON     string
	PendingReplannerJSON   string
	ReplannerResponseState *ResponseExtractState // 用于增量提取 response 字段

	// Persisted 累积的持久化状态，用于存储到数据库。
	Persisted *PersistedRuntime
}

// ResponseExtractState 跟踪 response 字段的增量提取状态。
type ResponseExtractState struct {
	InResponse    bool   // 是否在 response 字段值内
	ContentStart  int    // response 内容开始位置
	QuoteClosed   bool   // response 值的引号是否已关闭
	EscapeNext    bool   // 下一个字符是否是转义字符
	BufferContent string // 缓冲的内容，累积到一定量后发送
	ReplanSent    bool   // 是否已发送 replan 事件
}

// ResponseBufferConfig 缓冲配置
const (
	ResponseMinChunkSize = 50  // 最小累积字符数
	ResponseMaxWaitChars = 200 // 最大累积字符数（超过则强制发送）
)

func projectNormalizedEvents(events []NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	projected := make([]PublicStreamEvent, 0, len(events))
	for _, event := range events {
		projected = append(projected, projectNormalizedEvent(event, state)...)
	}
	return projected
}

func projectNormalizedEvent(event NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	// 确保 Persisted 已初始化
	if state.Persisted == nil {
		state.Persisted = &PersistedRuntime{}
	}

	switch event.Kind {
	case NormalizedKindHandoff:
		if event.Handoff == nil {
			return nil
		}
		// 更新持久化状态
		state.Persisted.Phase = "executing"
		state.Persisted.PhaseLabel = fmt.Sprintf("%s 开始处理", strings.TrimSpace(event.Handoff.To))
		state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
			ID:     fmt.Sprintf("handoff:%s", strings.TrimSpace(event.Handoff.To)),
			Kind:   "agent_handoff",
			Label:  strings.TrimSpace(event.Handoff.To),
			Status: "done",
		})
		return []PublicStreamEvent{{
			Event: "agent_handoff",
			Data: map[string]any{
				"from":   strings.TrimSpace(event.Handoff.From),
				"to":     strings.TrimSpace(event.Handoff.To),
				"intent": mapAgentNameToIntentType(strings.TrimSpace(event.Handoff.To)),
			},
		}}
	case NormalizedKindInterrupt:
		if event.Interrupt == nil {
			return nil
		}
		state.RunPhase = "waiting_approval"
		// 更新持久化状态
		state.Persisted.Phase = "waiting_approval"
		state.Persisted.PhaseLabel = "等待审批"
		state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
			ID:     event.Interrupt.CallID,
			Kind:   "tool_approval",
			Label:  event.Interrupt.ToolName,
			Detail: fmt.Sprintf("等待审批 %ds", event.Interrupt.TimeoutSeconds),
			Status: "pending",
		})
		return []PublicStreamEvent{
			{
				Event: "tool_approval",
				Data: map[string]any{
					"approval_id":     event.Interrupt.ApprovalID,
					"call_id":         event.Interrupt.CallID,
					"tool_name":       event.Interrupt.ToolName,
					"preview":         event.Interrupt.Preview,
					"timeout_seconds": event.Interrupt.TimeoutSeconds,
				},
			},
			NewRunStateEvent("waiting_approval", map[string]any{
				"agent": event.AgentName,
			}),
		}
	case NormalizedKindToolCall:
		if event.Tool == nil {
			return nil
		}
		// 更新持久化状态
		activeStepIndex := 0
		if state.Persisted.Plan != nil {
			activeStepIndex = state.Persisted.Plan.ActiveStepIndex
		}
		state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
			ID:        event.Tool.CallID,
			Kind:      "tool_call",
			Label:     event.Tool.ToolName,
			Status:    "active",
			StepIndex: activeStepIndex,
		})
		return []PublicStreamEvent{{
			Event: "tool_call",
			Data: map[string]any{
				"call_id":    event.Tool.CallID,
				"tool_name":  event.Tool.ToolName,
				"arguments":  event.Tool.Arguments,
				"agent":      strings.TrimSpace(event.AgentName),
			},
		}}
	case NormalizedKindToolResult:
		if event.Tool == nil {
			return nil
		}
		// 更新持久化状态：找到对应的 activity 并更新
		for i := range state.Persisted.Activities {
			if state.Persisted.Activities[i].ID == event.Tool.CallID {
				state.Persisted.Activities[i].Status = "done"
				state.Persisted.Activities[i].Kind = "tool_result"
				state.Persisted.Activities[i].Detail = truncateString(event.Tool.Content, 200)
			}
		}
		return []PublicStreamEvent{{
			Event: "tool_result",
			Data: map[string]any{
				"call_id":    event.Tool.CallID,
				"tool_name":  event.Tool.ToolName,
				"content":    event.Tool.Content,
				"agent":      strings.TrimSpace(event.AgentName),
			},
		}}
	case NormalizedKindMessage:
		return projectNormalizedMessage(event, state)
	default:
		return nil
	}
}

func projectNormalizedMessage(event NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	if event.Message == nil {
		return nil
	}

	trimmedAgent := strings.TrimSpace(event.AgentName)
	trimmedContent := strings.TrimSpace(event.Message.Content)

	// 确保 Persisted 已初始化
	if state.Persisted == nil {
		state.Persisted = &PersistedRuntime{}
	}

	if trimmedAgent == "planner" {
		raw := appendAgentJSONBuffer(state.PendingPlannerJSON, event.Message.Content)
		if steps, ok := decodeStepsEnvelope(strings.TrimSpace(raw)); ok {
			state.PendingPlannerJSON = ""
			state.TotalPlanSteps = len(steps)
			state.ReplanIteration = 0
			state.RunPhase = "planning"
			// 更新持久化状态
			state.Persisted.Plan = buildPersistedPlanFromSteps(steps, 0)
			state.Persisted.Phase = "planning"
			state.Persisted.PhaseLabel = "正在规划处理方式"
			state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
				ID:     "planning",
				Kind:   "plan",
				Label:  "规划处理步骤",
				Status: "done",
			})
			return []PublicStreamEvent{{
				Event: "plan",
				Data: map[string]any{
					"steps":     steps,
					"iteration": 0,
				},
			}}
		}

		if shouldBufferAgentEnvelope(state.PendingPlannerJSON, raw) {
			state.PendingPlannerJSON = raw
			return nil
		}

		state.PendingPlannerJSON = ""
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		return []PublicStreamEvent{{
			Event: "delta",
			Data: map[string]any{
				"content": raw,
				"agent":   trimmedAgent,
			},
		}}
	}

	if trimmedAgent == "replanner" {
		// 追加到 buffer
		prevLen := len(state.PendingReplannerJSON)
		raw := appendAgentJSONBuffer(state.PendingReplannerJSON, event.Message.Content)
		state.PendingReplannerJSON = raw

		// 尝试增量提取 response 字段
		events := extractResponseStreaming(state, raw, prevLen, trimmedAgent)
		if len(events) > 0 {
			return events
		}

		// 检查是否是 steps envelope
		if steps, ok := decodeStepsEnvelope(strings.TrimSpace(raw)); ok {
			state.PendingReplannerJSON = ""
			state.ReplannerResponseState = nil
			state.ReplanIteration++
			completed := state.TotalPlanSteps - len(steps)
			if completed < 0 {
				completed = 0
			}
			state.RunPhase = "planning"
			// 更新持久化状态
			state.Persisted.Plan = buildPersistedPlanFromSteps(steps, completed)
			state.Persisted.Phase = "planning"
			state.Persisted.PhaseLabel = "正在调整处理计划"
			state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
				ID:     "planning",
				Kind:   "replan",
				Label:  "更新处理计划",
				Status: "active",
			})
			return []PublicStreamEvent{{
				Event: "replan",
				Data: map[string]any{
					"steps":     steps,
					"completed": completed,
					"iteration": state.ReplanIteration,
					"is_final":  false,
				},
			}}
		}

		// 继续等待
		if shouldBufferAgentEnvelope(state.PendingReplannerJSON, raw) {
			return nil
		}

		// 不是有效的 JSON envelope，直接发送
		state.PendingReplannerJSON = ""
		state.ReplannerResponseState = nil
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		return []PublicStreamEvent{{
			Event: "delta",
			Data: map[string]any{
				"content": raw,
				"agent":   trimmedAgent,
			},
		}}
	}

	projected := make([]PublicStreamEvent, 0, 1)
	if trimmedContent != "" {
		projected = append(projected, PublicStreamEvent{
			Event: "delta",
			Data: map[string]any{
				"content": event.Message.Content,
				"agent":   trimmedAgent,
			},
		})
	}
	return projected
}

// FlushReplannerBuffer 刷新 replanner response 缓冲区
func FlushReplannerBuffer(state *ProjectionState, agent string) []PublicStreamEvent {
	if state.ReplannerResponseState == nil || state.ReplannerResponseState.BufferContent == "" {
		return nil
	}

	content := state.ReplannerResponseState.BufferContent
	state.ReplannerResponseState.BufferContent = ""

	return []PublicStreamEvent{{
		Event: "delta",
		Data: map[string]any{
			"content": content,
			"agent":   agent,
		},
	}}
}

func appendAgentJSONBuffer(existing, chunk string) string {
	return existing + chunk
}

func shouldBufferAgentEnvelope(existing, raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if existing != "" {
		return true
	}
	return strings.HasPrefix(trimmed, "{")
}

// extractResponseStreaming 增量提取 response 字段内容。
// 解析 JSON 字符串转义（\n -> 换行符等），去掉 {"response": " 和结尾的 "}
// 返回值：需要发送的事件列表（可能包含 replan + delta，或只有 delta，或空）
func extractResponseStreaming(state *ProjectionState, raw string, prevLen int, agent string) []PublicStreamEvent {
	// 初始化状态
	if state.ReplannerResponseState == nil {
		state.ReplannerResponseState = &ResponseExtractState{}
	}

	rs := state.ReplannerResponseState

	// 如果 response 值的引号已关闭，不再提取
	if rs.QuoteClosed {
		return nil
	}

	// 查找 response 字段开始
	if !rs.InResponse {
		// 尝试两种格式："response": " 和 "response":"
		responseKeys := []string{`"response": "`, `"response":"`}
		var idx int
		var keyLen int
		for _, key := range responseKeys {
			if i := strings.Index(raw, key); i != -1 {
				idx = i
				keyLen = len(key)
				break
			}
		}
		if keyLen == 0 {
			return nil
		}
		// 找到 response 字段开始
		rs.InResponse = true
		rs.ContentStart = idx + keyLen
	}

	// 从上次位置开始提取新内容，并解析 JSON 转义
	startPos := max(rs.ContentStart, prevLen)
	var newContent strings.Builder

	for i := startPos; i < len(raw); i++ {
		ch := raw[i]

		// 处理转义字符
		if rs.EscapeNext {
			switch ch {
			case 'n':
				newContent.WriteByte('\n')
			case 't':
				newContent.WriteByte('\t')
			case 'r':
				newContent.WriteByte('\r')
			case '"':
				newContent.WriteByte('"')
			case '\\':
				newContent.WriteByte('\\')
			case '/':
				newContent.WriteByte('/')
			default:
				// 未知的转义序列，保留反斜杠
				newContent.WriteByte('\\')
				newContent.WriteByte(ch)
			}
			rs.EscapeNext = false
			continue
		}

		if ch == '\\' {
			rs.EscapeNext = true
			continue
		}

		// 检查是否是结束引号
		if ch == '"' {
			rs.QuoteClosed = true
			break
		}

		newContent.WriteByte(ch)
	}

	// 更新 ContentStart 位置
	rs.ContentStart = len(raw)

	// 累积到缓冲区
	rs.BufferContent += newContent.String()

	// 检查是否需要发送
	events := make([]PublicStreamEvent, 0, 2)

	// 首次检测到 response 字段时，发送 replan 事件
	if !rs.ReplanSent && len(rs.BufferContent) > 0 {
		rs.ReplanSent = true
		state.ReplanIteration++
		state.RunPhase = "executing"
		events = append(events, PublicStreamEvent{
			Event: "replan",
			Data: map[string]any{
				"steps":     []string{},
				"completed": state.TotalPlanSteps,
				"iteration": state.ReplanIteration,
				"is_final":  true,
			},
		})
	}

	// 缓冲区达到阈值或 response 结束时发送
	if len(rs.BufferContent) >= ResponseMinChunkSize || len(rs.BufferContent) >= ResponseMaxWaitChars || rs.QuoteClosed {
		if len(rs.BufferContent) > 0 {
			events = append(events, PublicStreamEvent{
				Event: "delta",
				Data: map[string]any{
					"content": rs.BufferContent,
					"agent":   agent,
				},
			})
			rs.BufferContent = ""
		}
	}

	return events
}

// truncateString 截断字符串到指定长度。
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// buildPersistedPlanFromSteps 从步骤字符串数组构建 PersistedPlan。
//
// 参数:
//   - steps: 步骤标题数组
//   - completedCount: 已完成的步骤数量
func buildPersistedPlanFromSteps(steps []string, completedCount int) *PersistedPlan {
	result := &PersistedPlan{}
	for i, title := range steps {
		status := "pending"
		if i < completedCount {
			status = "done"
		} else if i == completedCount {
			status = "active"
		}
		result.Steps = append(result.Steps, PersistedStep{
			ID:     fmt.Sprintf("plan-step-%d", i),
			Title:  title,
			Status: status,
		})
	}
	if completedCount < len(steps) {
		result.ActiveStepIndex = completedCount
	}
	return result
}
