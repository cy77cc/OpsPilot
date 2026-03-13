## ADDED Requirements

### Requirement: AIV2 runtime SHALL execute AI conversations through a single ChatModelAgent

The system SHALL provide an `aiv2` runtime that executes AI conversations through a single `ChatModelAgent + Runner` path rather than the legacy multi-stage `rewrite -> planner -> executor -> summarizer` path. The `aiv2` runtime MUST support tool-calling, streaming output, final-answer generation, and human-in-the-loop resume inside the same runtime.

#### Scenario: single agent handles a readonly request
- **WHEN** a request is routed to the `aiv2` runtime for a readonly operational question
- **THEN** the system MUST execute the request through one primary `ChatModelAgent`
- **AND** the agent MUST be able to call registered readonly tools directly
- **AND** the runtime MUST stream tool activity and final answer without invoking expert-as-tool sub-agents

#### Scenario: single agent handles a mutating request with approval
- **WHEN** a request is routed to the `aiv2` runtime for a mutating operation
- **THEN** the system MUST keep the conversation inside the same `ChatModelAgent + Runner` lifecycle
- **AND** the runtime MUST interrupt before executing the gated tool call
- **AND** after approval the runtime MUST resume the interrupted run rather than start a new planning flow

### Requirement: AIV2 runtime SHALL reuse the existing operational tool inventory

The `aiv2` runtime SHALL reuse the existing operational tools and dependency injection model as the primary capability source, while removing the `expert agent as tool` indirection from the execution path.

#### Scenario: existing host and platform tools are reused
- **WHEN** maintainers build the `aiv2` runtime
- **THEN** they MUST be able to register existing host, kubernetes, service, delivery, and observability tools into a unified tool registry
- **AND** the runtime MUST NOT require re-implementing those tools as new business functions before first use
- **AND** the runtime MUST NOT insert an additional model-backed expert wrapper between the main agent and the tool execution path

### Requirement: AIV2 HITL SHALL gate a concrete pending tool invocation

Human-in-the-loop approval in `aiv2` MUST apply to a specific pending tool invocation, not to an abstract future step or a new model turn. The system MUST persist enough information to resume that exact invocation after approval.

#### Scenario: approval stores exact tool invocation state
- **WHEN** the agent attempts to invoke a mutating tool that requires approval
- **THEN** the runtime MUST persist the pending tool name, tool arguments, approval summary, turn identity, and checkpoint identity
- **AND** the approval response MUST refer to that persisted pending invocation

#### Scenario: approved resume executes the stored invocation
- **WHEN** the user approves a pending `aiv2` action
- **THEN** the runtime MUST resume the interrupted run from checkpoint
- **AND** the resumed run MUST execute the exact stored pending tool invocation
- **AND** the runtime MUST continue from that observation to produce the final answer

#### Scenario: rejected resume does not execute the stored invocation
- **WHEN** the user rejects a pending `aiv2` action
- **THEN** the runtime MUST NOT execute the stored tool invocation
- **AND** the assistant turn MUST end with a user-visible cancellation result
