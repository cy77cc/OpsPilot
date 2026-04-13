package runtime

import airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"

type PublicStreamEvent = airuntime.PublicStreamEvent
type ProjectionBlock = airuntime.ProjectionBlock
type RunProjection = airuntime.RunProjection
type NormalizedEvent = airuntime.NormalizedEvent
type NormalizedKind = airuntime.NormalizedKind

const (
	EventTypeToolCall       = airuntime.EventTypeToolCall
	EventTypeDelta          = airuntime.EventTypeDelta
	EventTypeError          = airuntime.EventTypeError
	EventTypeToolApproval   = airuntime.EventTypeToolApproval
	EventTypeRunState       = airuntime.EventTypeRunState
	EventTypeOpsPlanUpdated = airuntime.EventTypeOpsPlanUpdated
	EventTypeToolResult     = airuntime.EventTypeToolResult
	NormalizedKindInterrupt = airuntime.NormalizedKindInterrupt
)

type ToolCallPayload = airuntime.ToolCallPayload
type DeltaPayload = airuntime.DeltaPayload
type ToolApprovalPayload = airuntime.ToolApprovalPayload
type RunStatePayload = airuntime.RunStatePayload
type OpsPlanUpdatedPayload = airuntime.OpsPlanUpdatedPayload

var (
	NewStreamProjector    = airuntime.NewStreamProjector
	MarshalEventPayload   = airuntime.MarshalEventPayload
	NormalizeAgentEvent   = airuntime.NormalizeAgentEvent
	UnmarshalEventPayload = airuntime.UnmarshalEventPayload
)
