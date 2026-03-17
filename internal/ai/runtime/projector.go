package runtime

import "github.com/cloudwego/eino/adk"

type StreamProjector struct {
	state ProjectionState
}

func NewStreamProjector() *StreamProjector {
	return &StreamProjector{}
}

func (p *StreamProjector) Consume(event *adk.AgentEvent) []PublicStreamEvent {
	normalized := NormalizeAgentEvent(event)
	return projectNormalizedEvents(normalized, &p.state)
}

func (p *StreamProjector) Finish(runID string) PublicStreamEvent {
	return doneEvent(runID, p.state.ReplanIteration)
}

func (p *StreamProjector) Fail(runID string, err error) PublicStreamEvent {
	p.state.RunPhase = "failed"
	return errorEvent(runID, err)
}
