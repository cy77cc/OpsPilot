## ADDED Requirements

### Requirement: AI runtime MUST emit native ThoughtChain lifecycle events

The system MUST emit a native event family for user-visible ThoughtChain lifecycle rather than requiring the frontend to infer chain semantics from generic phase or tool events.

#### Scenario: chain nodes are opened and closed by the runtime
- **WHEN** a chat turn enters planning, execution, replanning, or approval waiting
- **THEN** the runtime MUST emit `chain_node_open`, `chain_node_patch`, and `chain_node_close` events with stable node identity
- **AND** the frontend MUST be able to render the user-visible chain directly from those events without reconstructing stage ownership from unrelated event families

#### Scenario: tool result updates do not create extra narrative nodes
- **WHEN** a tool call produces progress, result, or summary data
- **THEN** the runtime MUST update the active tool node details
- **AND** the runtime MUST NOT create a separate user-visible `tool_result` chain node by default

### Requirement: process chain MUST complete before final answer starts

The system MUST keep the live UI focused on the ThoughtChain while execution is in progress, and it MUST only start final-answer streaming after the process chain has reached a collapsed completed state.

#### Scenario: final answer starts only after chain collapse
- **WHEN** the runtime finishes all process-chain work for a turn
- **THEN** it MUST emit `chain_collapsed` before `final_answer_started`
- **AND** final answer content MUST be delivered only through `final_answer_delta`
- **AND** process-chain content MUST NOT continue streaming as normal final-answer prose

### Requirement: approval waits MUST behave as normal chain nodes

Approval-required states MUST be represented as first-class ThoughtChain nodes rather than detached side panels or out-of-band flow interruptions.

#### Scenario: approval node pauses chain progression
- **WHEN** execution reaches a step that requires approval
- **THEN** the runtime MUST open an `approval` chain node
- **AND** the UI MUST render approval interaction within that node's detail area
- **AND** approval acceptance or rejection MUST close or patch the same node before the chain proceeds or terminates

### Requirement: session replay MUST preserve chain and final-answer relationship

The persisted chat session model MUST preserve enough lifecycle state to reconstruct the same user-visible relationship between the collapsed ThoughtChain and the final answer during history replay.

#### Scenario: completed session replays collapsed chain and answer
- **WHEN** a client restores a completed AI session
- **THEN** the session detail response MUST allow the client to render a collapsed completed ThoughtChain and the final answer separately
- **AND** planner JSON, tool arguments, or replanning notes MUST NOT be replayed as ordinary final-answer prose
