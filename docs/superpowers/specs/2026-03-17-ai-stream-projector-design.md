# AI Stream Projector Design

## Goal

Design a backend-only stream projection layer that converts `adk.AgentEvent` runtime events into a stable, frontend-friendly SSE protocol without coupling the frontend directly to ADK internals.

This design keeps the current public SSE event family, but introduces an internal normalized event model so approval workflows, resume flows, and future protocol upgrades can be added without rewriting chat execution logic.

## Scope

In scope:

- define a two-stage projection pipeline from ADK events to public SSE events
- keep the current public SSE event family compatible with the existing frontend path
- add an explicit public `run_state` event for frontend state handling
- model approval interrupts as first-class projected events
- reduce coupling between `logic.go` and event projection details
- define test boundaries for normalization, projection, and chat integration

Out of scope for this design:

- frontend rendering changes
- SSE reconnect/replay redesign
- generic UI block tree modeling
- approval persistence or approval execution implementation
- replacing the current SSE transport format

## Current Problems

The current implementation already projects ADK events into frontend SSE events, but the logic is spread across `internal/service/ai/logic/logic.go` and `internal/ai/runtime/streamer.go`.

This creates several problems:

- runtime semantics and product semantics are mixed together
- `logic.go` understands too much about planner, replanner, approval, and stream variants
- planner and replanner parsing is tightly coupled to current event names and agent names
- approval is only partially represented as `tool_approval`, with no first-class run state transition
- future additions like approval resume or richer stream states would likely add more ad hoc branching

## Design Summary

Introduce a two-layer event projection model:

1. `adk.AgentEvent -> NormalizedEvent`
2. `NormalizedEvent -> PublicStreamEvent`

`Logic.Chat()` remains the runtime driver, but no longer directly interprets ADK event structure. Instead, it delegates all event understanding to a `StreamProjector`.

The public SSE contract remains event-name-based for compatibility:

- `meta`
- `agent_handoff`
- `plan`
- `replan`
- `delta`
- `tool_call`
- `tool_result`
- `tool_approval`
- `run_state`
- `done`
- `error`

Internally, all projected events may also carry a consistent envelope so the backend can later support a single-envelope protocol without redesigning the internal model.

## Architecture

### 1. StreamProjector

`StreamProjector` is the public backend component used by chat execution.

Responsibilities:

- consume raw `adk.AgentEvent`
- maintain projection state needed for plan and run-state projection
- emit one or more public stream events for each runtime event
- provide consistent `done` and `error` event generation

It does not:

- persist run data
- update database state
- own SSE writing
- own approval business logic

Proposed shape:

```go
type StreamProjector struct {
	state ProjectionState
}

func (p *StreamProjector) Consume(event *adk.AgentEvent) []PublicStreamEvent
func (p *StreamProjector) Finish(runID string) PublicStreamEvent
func (p *StreamProjector) Fail(runID string, err error) PublicStreamEvent
```

### 2. Normalizer

The normalizer translates ADK runtime events into backend-stable event objects.

Responsibilities:

- understand `AgentEvent`, `AgentAction`, `MessageVariant`, streaming vs non-streaming output
- preserve runtime detail needed by later projection
- avoid product-specific assumptions such as `plan` or `replan`

The normalizer should emit events close to runtime semantics, for example:

- `message`
- `tool_call`
- `tool_result`
- `handoff`
- `interrupt`
- `error`

Important rule:

Planner and replanner are not distinct normalized event kinds. They are message events with different `AgentName` values. Product-specific meanings like `plan` and `replan` are decided later by the projector.

### 3. Projector

The projector converts normalized runtime events into frontend-facing public stream events.

Responsibilities:

- map runtime events to current A2UI/SSE event family
- emit `run_state` transitions
- decode planner and replanner JSON envelopes
- generate approval-facing payloads

This layer is where product semantics live.

## Core Data Structures

### NormalizedEvent

```go
type NormalizedEvent struct {
	Kind      NormalizedKind
	AgentName string
	RunPath   []string

	Message   *NormalizedMessage
	Tool      *NormalizedTool
	Handoff   *NormalizedHandoff
	Interrupt *NormalizedInterrupt
	Err       error

	Raw *adk.AgentEvent
}
```

Suggested related types:

```go
type NormalizedMessage struct {
	Role        string
	Content     string
	IsStreaming bool
}

type NormalizedTool struct {
	CallID    string
	ToolName  string
	Arguments map[string]any
	Content   string
	Phase     string // call | result
}

type NormalizedHandoff struct {
	From string
	To   string
}

type NormalizedInterrupt struct {
	Type           string // approval
	ApprovalID     string
	CallID         string
	ToolName       string
	Preview        map[string]any
	TimeoutSeconds int
}
```

### ProjectionState

`ProjectionState` should stay intentionally small and only contain data required to translate runtime events into public protocol events.

```go
type ProjectionState struct {
	TotalPlanSteps  int
	ReplanIteration int
	RunPhase        string
}
```

### PublicStreamEvent

The current external API remains event-name-based, but the backend should use a structured event object internally.

```go
type PublicStreamEvent struct {
	Event    string
	Data     any
	Envelope *EventEnvelope
}

type EventEnvelope struct {
	Version   string
	Type      string
	Timestamp time.Time
	RunID     string
	AgentName string
}
```

The SSE transport can continue to write `Event` and `Data` exactly as it does now. The envelope exists to stabilize internal processing and leave room for later protocol upgrades.

## Public Event Contract

### Existing Events To Preserve

- `meta`
- `agent_handoff`
- `plan`
- `replan`
- `delta`
- `tool_call`
- `tool_result`
- `tool_approval`
- `done`
- `error`

### New Event

- `run_state`

Suggested `run_state.status` values:

- `routing`
- `planning`
- `executing`
- `waiting_approval`
- `completed`
- `failed`

This event exists to reduce frontend state reconstruction logic. The frontend should not be forced to infer high-level run status only from low-level event combinations.

## Approval Projection

Approval should be projected through the same two-stage pipeline as all other runtime behavior.

Source:

- `event.Action.Interrupted`

Normalized output:

- `NormalizedEvent{Kind: interrupt}`

Public projection:

- `tool_approval`
- `run_state{status: "waiting_approval"}`

This ensures approval becomes a first-class protocol concept without exposing raw ADK interrupt payloads to the frontend.

The design intentionally avoids putting approval persistence or resume semantics inside the projector. Those remain outside this component.

## Planner And Replanner Handling

Planner and replanner outputs remain special only in the projection layer.

Rules:

- if `AgentName == "planner"` and the assistant content decodes into plan steps, emit `plan`
- if `AgentName == "replanner"` and the assistant content decodes into plan steps, emit `replan`
- if `AgentName == "replanner"` and the assistant content decodes into a final response envelope, emit:
  - `replan` with `is_final=true`
  - `delta` with final assistant text

This preserves current frontend behavior while keeping planner-specific parsing out of the normalizer.

## Chat Runtime Integration

`Logic.Chat()` should become the owner of execution flow, not event semantics.

Responsibilities kept in `Logic.Chat()`:

- create session, messages, and run records
- create the ADK runner
- iterate over raw runtime events
- feed events into the projector
- emit projected SSE events
- accumulate final assistant content from projected `delta` events
- persist final run and assistant message state

Responsibilities removed from `Logic.Chat()`:

- direct interpretation of `AgentEvent.Action`
- direct planner/replanner parsing
- direct approval payload extraction
- direct conversion of message variants into product events

## File Layout

The current `internal/ai/runtime/streamer.go` mixes too many responsibilities. Split it into focused files:

- `internal/ai/runtime/projector.go`
  - public entrypoint
  - `StreamProjector`
  - `ProjectionState`

- `internal/ai/runtime/normalize.go`
  - `AgentEvent -> NormalizedEvent`

- `internal/ai/runtime/project.go`
  - `NormalizedEvent -> PublicStreamEvent`

- `internal/ai/runtime/events.go`
  - public event structs
  - envelopes
  - helper constructors for `done`, `error`, `run_state`

- `internal/ai/runtime/plan_decode.go`
  - planner and replanner JSON envelope parsing

This keeps protocol evolution and runtime parsing isolated from chat business logic.

## Testing Strategy

### 1. Normalizer Tests

Focus on ADK-facing translation:

- assistant message event
- tool call extraction
- tool result extraction
- transfer-to-agent action
- interrupt action
- streaming and non-streaming variants

### 2. Projector Tests

Focus on product-facing projection:

- planner output to `plan`
- replanner output to `replan`
- replanner final response to `replan + delta`
- interrupt to `tool_approval + run_state(waiting_approval)`
- handoff to `agent_handoff`
- state transitions to `run_state`

### 3. Chat Integration Tests

Keep a small number of integration tests in `logic`:

- projected delta accumulation updates assistant message content
- `done` marks run completed
- projected error marks run failed
- approval event is forwarded to emitter

## Migration Plan

Implement this in place without changing the external SSE API in the same step.

Recommended order:

1. introduce normalized event types and projector types
2. move current projection helpers behind the new projector interface
3. update `Logic.Chat()` to call projector methods instead of scattered helpers
4. add `run_state` emission
5. add tests for normalization and projection
6. remove obsolete helper functions or merge them into the new file layout

## Risks And Guardrails

### Risk: over-designing the internal protocol

Guardrail:

- keep normalized event kinds close to runtime semantics
- do not introduce UI block trees in this layer

### Risk: coupling projection to current agent names forever

Guardrail:

- keep planner/replanner recognition isolated in one projection file
- treat this as a protocol adapter, not a global runtime invariant

### Risk: making projector responsible for business workflow

Guardrail:

- projector emits state
- logic owns persistence and lifecycle actions

## Recommendation

Adopt a two-layer projection architecture:

- `adk.AgentEvent -> NormalizedEvent -> PublicStreamEvent`

Keep the current public SSE event family for compatibility, add `run_state`, and model approval as both:

- `tool_approval`
- `run_state(waiting_approval)`

This is the smallest design that cleans up current coupling, preserves the frontend contract, and leaves a clear path for approval and resume expansion.
