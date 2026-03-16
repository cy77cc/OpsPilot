package ai

// StreamEventName is the Phase 1 public SSE event name.
type StreamEventName string

const (
	StreamEventInit        StreamEventName = "init"
	StreamEventIntent      StreamEventName = "intent"
	StreamEventStatus      StreamEventName = "status"
	StreamEventDelta       StreamEventName = "delta"
	StreamEventProgress    StreamEventName = "progress"
	StreamEventReportReady StreamEventName = "report_ready"
	StreamEventError       StreamEventName = "error"
	StreamEventDone        StreamEventName = "done"
)

// StreamEvent is the public Phase 1 SSE envelope.
type StreamEvent struct {
	Event StreamEventName `json:"event"`
	Data  any             `json:"data,omitempty"`
}
