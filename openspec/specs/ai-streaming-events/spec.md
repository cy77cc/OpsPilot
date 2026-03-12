## MODIFIED Requirements

### Requirement: AI Chat Stream Events

The system SHALL emit Server-Sent Events (SSE) that preserve model-stage output as faithfully as possible during AI chat streaming, while also exposing turn-and-block-native lifecycle events for richer UI consumers.

Required compatibility event categories:

- `meta`: session metadata at stream start
- `stage_delta`: incremental model-stage output for rewrite, plan, approval-gate messaging, execute, and summary stages
- `delta`: incremental final answer content
- `thinking_delta`: model reasoning/thinking content when available
- `tool_call`: tool invocation with name and arguments
- `tool_result`: tool execution completion with result
- `approval_required`: approval gate requiring user confirmation before execution starts
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

#### Scenario: successful chat stream preserves model-stage output
- **WHEN** user sends a message that triggers rewrite, planning, approval gating, execution, and summarization
- **THEN** the system MUST emit `stage_delta` events as stage content becomes available
- **AND** the system MUST emit `delta` events for the final answer before `done`
- **AND** the system MUST NOT wait until the entire request finishes before exposing all model-stage content

#### Scenario: stream exposes turn-and-block lifecycle
- **WHEN** an assistant turn is created for a streaming request
- **THEN** the system MUST emit `turn_started`
- **AND** user-visible status, text, tool, approval, evidence, and error surfaces MUST be represented through block lifecycle events
- **AND** block updates MUST identify the owning `turn_id` and `block_id`

#### Scenario: unavailable stage is reported explicitly
- **WHEN** a model-backed stage becomes unavailable during streaming
- **THEN** the system MUST emit an explicit user-visible failure or unavailability signal for that stage
- **AND** the stream MUST NOT substitute code-generated semantic content pretending the stage completed successfully

#### Scenario: approval gate appears before execution
- **WHEN** AI identifies a planned step that requires approval
- **THEN** the stream MUST emit `approval_required` before the gated step produces `tool_call` or `tool_result`
- **AND** the system MUST also surface a block-native approval update bound to the active turn
- **AND** the stream MUST pause in a resumable gate state with runtime step identity for later resume

#### Scenario: heartbeat maintains connection
- **WHEN** streaming session lasts longer than 10 seconds
- **THEN** system emits `heartbeat` events every 10 seconds with timestamp

## MODIFIED Requirements

### Requirement: Checkpoint-based Resume

The system SHALL support resuming interrupted sessions by stable runtime identity rather than model-specific checkpoint semantics, and resumed execution MUST continue the original assistant turn lifecycle through execution and summary.

Resume inputs MUST be based on:

- `session_id`
- `plan_id`
- `step_id`
- approval decision

Compatibility aliases MAY exist for legacy clients, but runtime semantics MUST be plan-step based.

#### Scenario: resume after approval
- **WHEN** user submits approval for an interrupted session
- **THEN** system resumes execution for the specific `session_id + plan_id + step_id`
- **AND** continues emitting SSE events from that runtime point
- **AND** resumed events MUST continue on the previously active assistant `turn_id`
- **AND** if execution completes successfully the resumed stream MUST continue into summary before terminal completion

#### Scenario: resume after rejection
- **WHEN** user rejects approval for an interrupted session
- **THEN** system returns a rejection message
- **AND** does not execute the interrupted step
- **AND** the interrupted assistant turn MUST move into a terminal cancelled or rejected state

#### Scenario: legacy resume compatibility does not redefine runtime identity
- **WHEN** a legacy client sends a checkpoint-style resume request
- **THEN** the system MAY translate that request into current runtime identifiers
- **AND** the canonical runtime identity MUST remain `session_id + plan_id + step_id`
