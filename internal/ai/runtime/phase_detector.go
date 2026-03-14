package runtime

import (
	"strings"

	"github.com/cloudwego/eino/adk"
)

// PhaseDetector 从 ADK 事件中做最小阶段推断。
// 当前实现只依赖稳定可见的 AgentName 和文本特征，不引入复杂状态机。
type PhaseDetector struct {
	currentPhase PhaseName
	stepCounter  int
}

func NewPhaseDetector() *PhaseDetector {
	return &PhaseDetector{currentPhase: PhasePlanning}
}

func (d *PhaseDetector) Detect(event *adk.AgentEvent) string {
	if d == nil {
		return string(PhasePlanning)
	}

	phase := d.currentPhase
	agentName := strings.ToLower(strings.TrimSpace(eventAgentName(event)))
	text := strings.ToLower(strings.TrimSpace(eventMessageText(event)))

	switch {
	case strings.Contains(agentName, "replan"), strings.Contains(agentName, "replanner"):
		phase = PhaseReplanning
	case strings.Contains(agentName, "plan"), strings.Contains(agentName, "planner"):
		phase = PhasePlanning
	case strings.Contains(agentName, "execute"), strings.Contains(agentName, "executor"):
		phase = PhaseExecuting
	case strings.Contains(text, "重新规划"), strings.Contains(text, "调整计划"), strings.Contains(text, "replan"):
		phase = PhaseReplanning
	case strings.Contains(text, "计划"), strings.Contains(text, "步骤"), strings.Contains(text, "```json"):
		phase = PhasePlanning
	case strings.Contains(text, "执行"), strings.Contains(text, "调用工具"), strings.Contains(text, "approval"):
		phase = PhaseExecuting
	}

	d.currentPhase = phase
	return string(phase)
}

func (d *PhaseDetector) NextStepID() string {
	if d == nil {
		return "step-1"
	}
	d.stepCounter++
	return "step-" + itoa(d.stepCounter)
}

func eventAgentName(event *adk.AgentEvent) string {
	if event == nil {
		return ""
	}
	return event.AgentName
}

func eventMessageText(event *adk.AgentEvent) string {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil || event.Output.MessageOutput.Message == nil {
		return ""
	}
	return event.Output.MessageOutput.Message.Content
}
