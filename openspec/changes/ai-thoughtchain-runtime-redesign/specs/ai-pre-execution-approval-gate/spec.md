## MODIFIED Requirements

### Requirement: AI control plane MUST gate mutating execution before executor start

The system MUST treat approval for mutating or high-risk AI steps as a ThoughtChain-native control-plane gate that pauses user-visible execution with a first-class `approval` node before the gated operation starts.

All mutating or high-risk tools exposed to the runtime MUST still be classified into approval policy evaluation before execution starts. Tool registration, runtime wrapping, and fallback classification MUST fail closed for unresolved mutating tools rather than silently bypassing approval.

#### Scenario: gated step opens approval node before execution
- **WHEN** planner or executor reaches a mutating or high-risk step that requires approval
- **THEN** the control plane MUST open an `approval` node on the active ThoughtChain before starting the gated operation
- **AND** the user-visible lifecycle MUST reflect that the chain is waiting for confirmation rather than already executing the gated action
- **AND** the gated operation MUST NOT begin before approval is granted

#### Scenario: mutating tool registration cannot bypass approval
- **WHEN** a runtime tool is classified as mutating or high-risk by registry metadata or safe fallback inference
- **THEN** the tool MUST be wrapped by approval gating before it becomes invokable
- **AND** a registry lookup miss or tool-name mismatch MUST NOT silently downgrade the tool to unapproved execution
- **AND** the system MUST record enough diagnostic metadata to explain why approval was required or skipped

### Requirement: approval resume MUST continue execute then summary on the same assistant turn

The system MUST treat approval acceptance as permission to resume the same ThoughtChain and MUST continue execution and answer generation on that chain after the approval node resolves.

#### Scenario: approved gate resumes the same chain
- **WHEN** a user approves a waiting ThoughtChain approval node
- **THEN** the control plane MUST resume execution for that chain and node identity
- **AND** subsequent execution and answer events MUST continue on the same chain
- **AND** once execution completes, the runtime MUST continue into answer generation before terminal completion

#### Scenario: rejected gate terminates without execution
- **WHEN** a user rejects a waiting ThoughtChain approval node
- **THEN** the control plane MUST NOT start the gated operation
- **AND** the chain MUST move to a terminal rejected or cancelled outcome
- **AND** the user-visible output MUST describe that execution did not proceed
