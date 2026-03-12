## MODIFIED Requirements

### Requirement: existing approval and resume flows MUST remain part of the control plane

The control plane MUST keep approval, confirmation, preview, and resume flows as formal orchestration dependencies, and approval for mutating AI steps MUST be modeled as a pre-execution gate between planning and execution rather than as an executor-internal pause after work has started.

#### Scenario: mutating execution pauses for control-plane review
- **WHEN** a planned step is mutating, high-risk, or otherwise requires approval
- **THEN** the control plane MUST use approval, confirmation, preview, and resume flows as orchestration dependencies before expert execution begins for that step
- **AND** the task lifecycle MUST remain resumable after interruption
- **AND** approval MUST authorize the transition into executor work rather than resume a step that has already entered actual tool execution
