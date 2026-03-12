## ADDED Requirements

### Requirement: AI control plane MUST gate mutating execution before executor start

The system MUST treat approval for mutating or high-risk AI steps as a pre-execution control-plane gate rather than an executor-internal interruption point.

#### Scenario: plan enters approval gate before execution
- **WHEN** planner produces a plan containing a mutating or high-risk step that requires approval
- **THEN** the control plane MUST create an approval-required runtime state before starting executor work for that step
- **AND** the step MUST NOT enter actual expert execution before approval is granted
- **AND** the user-visible lifecycle MUST reflect that the task is waiting for confirmation rather than already executing

### Requirement: approval resume MUST continue execute then summary on the same assistant turn

The system MUST treat approval acceptance as permission to begin or resume execution and MUST continue into summarization on the same assistant turn after execution completes.

#### Scenario: approved gate continues full post-approval flow
- **WHEN** a user approves a gated AI step
- **THEN** the control plane MUST continue execution for the gated `session_id + plan_id + step_id`
- **AND** execution events MUST continue on the original assistant `turn_id`
- **AND** once execution completes, the control plane MUST continue into summary generation before emitting terminal completion

#### Scenario: rejected gate terminates without execution
- **WHEN** a user rejects a gated AI step
- **THEN** the control plane MUST NOT start expert execution for that step
- **AND** the assistant turn MUST move to a terminal cancelled or rejected outcome
- **AND** the user-visible output MUST describe that execution did not proceed
