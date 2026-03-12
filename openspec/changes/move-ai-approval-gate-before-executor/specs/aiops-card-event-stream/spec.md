## MODIFIED Requirements

### Requirement: card event stream MUST cover plan, execution, and outcome phases

The platform event family MUST represent planning, approval gating, step execution, evidence, user interaction, and final outcome states, and MUST distinguish lightweight status updates from durable user-visible content blocks.

#### Scenario: event family spans the task lifecycle
- **WHEN** reviewers inspect the event model
- **THEN** the event family MUST cover plan creation, approval-required states, step status, tool activity, evidence, ask-user interactions, replanning, summaries, next actions, completion, and errors
- **AND** the event family MUST define which events are projected as status, plan, approval, tool, evidence, text, or error blocks
- **AND** approval-required states MUST appear before gated execution begins

## ADDED Requirements

### Requirement: approval blocks MUST behave as pre-execution gate surfaces

Approval-oriented card events MUST represent an execution gate, not an already-running tool step.

#### Scenario: approval card precedes execute cards
- **WHEN** a planned AI step requires approval
- **THEN** the backend MUST project an approval block before any tool execution block for that gated step
- **AND** once approval is granted, the approval block MAY collapse into lightweight status feedback while execution and summary blocks take over the turn
