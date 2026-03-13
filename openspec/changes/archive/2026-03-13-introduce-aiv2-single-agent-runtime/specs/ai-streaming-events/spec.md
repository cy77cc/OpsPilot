## MODIFIED Requirements

### Requirement: AI Chat Stream Events

The system SHALL emit Server-Sent Events (SSE) that preserve model-stage output as faithfully as possible during AI chat streaming, while also exposing turn-and-block-native lifecycle events for richer UI consumers.

Required compatibility event categories:

- `meta`: session metadata at stream start
- `stage_delta`: incremental model-stage output for request understanding, tool-chain progress, approval-gate messaging, and final-answer summarization
- `delta`: incremental final answer content
- `thinking_delta`: model reasoning/thinking content when available
- `tool_call`: tool invocation with name and arguments
- `tool_result`: tool execution completion with result
- `approval_required`: approval gate requiring user confirmation before a gated tool executes
- `heartbeat`: periodic keep-alive event
- `done`: stream completion
- `error`: error notification

Required turn/block-native event categories:

- `turn_started`: assistant turn lifecycle start
- `block_open`: a renderable block is created
- `block_delta`: incremental content or state update for a block
- `block_replace`: block payload replacement for structured cards
- `block_close`: block is finalized
- `turn_state`: assistant turn status or phase transition
- `turn_done`: assistant turn completion

The system MUST treat `stage_delta` and `delta` as model-facing content channels rather than code-generated template placeholders, and the turn/block-native events MUST be the semantic source of truth for new UI consumers.

#### Scenario: successful aiv2 chat stream exposes thought, tools, and final answer
- **WHEN** a request is processed by the `aiv2` runtime
- **THEN** the system MUST emit `thinking_delta` as the main agent reasons
- **AND** the system MUST emit `tool_call` and `tool_result` as tools are invoked
- **AND** the system MUST emit `delta` for the final answer before `done`
- **AND** the system MUST NOT require a separate rewrite/planner/summarizer model chain to expose streaming content

#### Scenario: approval gate interrupts before mutating tool execution
- **WHEN** the `aiv2` runtime encounters a mutating tool call that requires approval
- **THEN** the stream MUST emit `approval_required` before the tool mutates the target system
- **AND** the stream MUST pause in a resumable gate state tied to checkpoint identity
- **AND** no `tool_result` for the gated mutation may be emitted before approval

#### Scenario: turn and block lifecycle remains available for single-agent runtime
- **WHEN** `aiv2` produces a streaming assistant turn
- **THEN** the system MUST continue emitting `turn_started`, block lifecycle events, `turn_state`, and `turn_done`
- **AND** tool, approval, text, and thinking surfaces MUST still be representable through block events

## MODIFIED Requirements

### Requirement: Checkpoint-based Resume

The system SHALL support resuming interrupted sessions by stable runtime identity rather than model-specific checkpoint semantics, and resumed execution MUST continue the original assistant turn lifecycle through the rest of the tool loop and final answer generation.

Resume inputs MUST be based on:

- public request identity at the gateway boundary
- runtime-specific interrupt/checkpoint identity inside the selected backend runtime
- approval decision

Compatibility aliases MAY exist for legacy clients, but resumed behavior MUST continue the original assistant turn rather than starting a new turn.

#### Scenario: aiv2 resume after approval
- **WHEN** the user approves an interrupted `aiv2` request
- **THEN** the system MUST resume the stored interrupted run from checkpoint
- **AND** the resumed stream MUST continue on the previously active assistant `turn_id`
- **AND** the resumed run MUST execute the stored pending tool invocation
- **AND** after tool execution the resumed stream MUST continue into the final answer before terminal completion

#### Scenario: aiv2 resume after rejection
- **WHEN** the user rejects an interrupted `aiv2` request
- **THEN** the system MUST not execute the stored pending tool invocation
- **AND** the interrupted assistant turn MUST move into a terminal cancelled or rejected state
- **AND** the stream MUST still produce a user-visible cancellation result before completion
