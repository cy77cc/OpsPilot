## MODIFIED Requirements

### Requirement: AI runtime MUST emit native ThoughtChain lifecycle events

The system MUST emit `thoughtChain` lifecycle as the only primary runtime protocol for user-visible AI chat execution and MUST NOT require the frontend to infer chain semantics from legacy phase, block, or detached approval event families on the primary path.

#### Scenario: chain nodes are opened and closed by the runtime
- **WHEN** a chat turn enters planning, execution, replanning, approval waiting, or final answer generation
- **THEN** the runtime MUST emit native chain lifecycle events with stable chain and node identity
- **AND** the frontend MUST be able to render the user-visible chain directly from those events without reconstructing stage ownership from unrelated event families

#### Scenario: primary chat flow does not depend on legacy runtime families
- **WHEN** the primary AI chat runtime streams a live assistant response
- **THEN** the runtime MUST NOT require legacy `turn/block`, `phase/step`, or detached approval event families to describe the user-visible process chain
- **AND** any temporary migration shim MUST NOT remain part of the steady-state primary path

### Requirement: process chain MUST complete before final answer starts

The system MUST keep the live UI focused on the ThoughtChain while execution is in progress, and it MUST only stream final-answer content after the process chain reaches a completed state within the same runtime model.

#### Scenario: final answer starts only after process chain completion
- **WHEN** the runtime finishes all process-chain work for a turn
- **THEN** it MUST mark the chain process nodes complete before starting the final answer node
- **AND** final answer content MUST be delivered through the dedicated answer node stream
- **AND** process-chain content MUST NOT continue streaming as ordinary final-answer prose

### Requirement: approval waits MUST behave as normal chain nodes

Approval-required states MUST be represented as first-class ThoughtChain nodes within the same runtime, pause the chain in-place, and resume or terminate the same chain after a user decision.

#### Scenario: approval node pauses chain progression
- **WHEN** execution reaches a step that requires approval
- **THEN** the runtime MUST open an `approval` chain node in `waiting` state
- **AND** the UI MUST render approval interaction within that node's detail area
- **AND** approval acceptance or rejection MUST update and close the same node before the chain proceeds or terminates

### Requirement: session replay MUST preserve chain and final-answer relationship

The persisted chat session model MUST preserve enough lifecycle state to reconstruct the same user-visible thoughtChain and final-answer relationship during history replay, using the same runtime model as live rendering.

#### Scenario: completed session replays chain and answer from one model
- **WHEN** a client restores a completed AI session
- **THEN** the session detail response MUST allow the client to reconstruct the same ordered thoughtChain nodes and final answer separately
- **AND** planner JSON, tool arguments, or replanning notes MUST NOT be replayed as ordinary final-answer prose
- **AND** restored sessions MUST use the same rendering model as live assistant responses
