package runtime

import "strings"

type ProjectionState struct {
	TotalPlanSteps         int
	ReplanIteration        int
	RunPhase               string
	PendingPlannerJSON     string
	PendingReplannerJSON   string
	ReplannerResponseState *ResponseExtractState // 用于增量提取 response 字段
}

// ResponseExtractState 跟踪 response 字段的增量提取状态。
type ResponseExtractState struct {
	InResponse    bool   // 是否在 response 字段值内
	ContentStart  int    // response 内容开始位置
	EscapeNext    bool   // 下一个字符是否被转义
	QuoteClosed   bool   // response 值的引号是否已关闭
	PendingBuffer string // 未提取的缓冲内容
}

func projectNormalizedEvents(events []NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	projected := make([]PublicStreamEvent, 0, len(events))
	for _, event := range events {
		projected = append(projected, projectNormalizedEvent(event, state)...)
	}
	return projected
}

func projectNormalizedEvent(event NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	switch event.Kind {
	case NormalizedKindHandoff:
		if event.Handoff == nil {
			return nil
		}
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

	if trimmedAgent == "planner" {
		raw := appendAgentJSONBuffer(state.PendingPlannerJSON, event.Message.Content)
		if steps, ok := decodeStepsEnvelope(strings.TrimSpace(raw)); ok {
			state.PendingPlannerJSON = ""
			state.TotalPlanSteps = len(steps)
			state.ReplanIteration = 0
			state.RunPhase = "planning"
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
		if extracted, hasReplan := extractResponseStreaming(state, raw, prevLen); len(extracted) > 0 {
			events := make([]PublicStreamEvent, 0, 2)

			// 首次检测到 response 字段时，发送 replan 事件
			if hasReplan {
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

			events = append(events, PublicStreamEvent{
				Event: "delta",
				Data: map[string]any{
					"content": extracted,
					"agent":   trimmedAgent,
				},
			})
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
// 返回值：
//   - string: 新提取的内容（可发送 delta）
//   - bool: 是否首次检测到 response 字段（需要发送 replan 事件）
func extractResponseStreaming(state *ProjectionState, raw string, prevLen int) (string, bool) {
	// 初始化状态
	if state.ReplannerResponseState == nil {
		state.ReplannerResponseState = &ResponseExtractState{}
	}

	rs := state.ReplannerResponseState

	// 如果 response 值的引号已关闭，不再提取
	if rs.QuoteClosed {
		return "", false
	}

	justEntered := false
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
			return "", false
		}
		// 找到 response 字段开始
		rs.InResponse = true
		rs.ContentStart = idx + keyLen
		rs.EscapeNext = false
		justEntered = true
	}

	// 从上次位置开始提取新内容
	newContent := strings.Builder{}
	startPos := max(rs.ContentStart, prevLen)

	for i := startPos; i < len(raw); i++ {
		ch := raw[i]

		if rs.EscapeNext {
			// 处理转义字符
			switch ch {
			case '"':
				newContent.WriteByte('"')
			case '\\':
				newContent.WriteByte('\\')
			case 'n':
				newContent.WriteByte('\n')
			case 't':
				newContent.WriteByte('\t')
			default:
				// 其他转义，原样输出
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

		if ch == '"' {
			// response 值结束
			rs.QuoteClosed = true
			break
		}

		newContent.WriteByte(ch)
	}

	// 更新 ContentStart 位置（用于下一次提取）
	rs.ContentStart = len(raw)

	result := newContent.String()
	// 首次进入 response 字段且有内容时，返回 hasReplan=true
	hasReplan := justEntered && len(result) > 0

	return result, hasReplan
}
