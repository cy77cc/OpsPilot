package runtime

import (
	"maps"
	"time"
)

type PublicStreamEvent = StreamEvent

type EventEnvelope struct {
	Version   string
	Type      string
	Timestamp time.Time
	RunID     string
	AgentName string
}

type a2uiProjectionState struct {
	iterations int
}

func NewRunStateEvent(status string, payload map[string]any) PublicStreamEvent {
	data := map[string]any{
		"status": status,
	}
	maps.Copy(data, payload)
	return PublicStreamEvent{
		Event: "run_state",
		Data:  data,
	}
}

func NewToolResultEvent(callID, toolName, content, status, agentName string) PublicStreamEvent {
	return PublicStreamEvent{
		Event: "tool_result",
		Data: map[string]any{
			"call_id":   callID,
			"tool_name": toolName,
			"content":   content,
			"status":    status,
			"agent":     agentName,
		},
	}
}

// mapAgentNameToIntentType 将 Agent 名称映射为意图类型。
//
// DeepAgents 架构下的 Agent 名称：
//   - QAAgent: 知识问答
//   - K8sAgent: K8s 操作
//   - HostAgent: 主机操作
//   - MonitorAgent: 监控查询
//   - ChangeAgent: 变更操作
func mapAgentNameToIntentType(agentName string) string {
	switch agentName {
	case "QAAgent":
		return "qa"
	case "K8sAgent":
		return "k8s"
	case "HostAgent":
		return "host"
	case "MonitorAgent":
		return "monitor"
	case "ChangeAgent":
		return "change"
	default:
		return "unknown"
	}
}
