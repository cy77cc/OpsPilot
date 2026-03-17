## ADDED Requirements

### Requirement: Complex-agent runs SHALL render a collapsible thought chain
The AI copilot MUST render a default-collapsed thought-process card for complex agent runs so users can inspect execution details without overwhelming the main answer surface.

#### Scenario: Show deep thinking card for plan-execute-replan agent
- **WHEN** an assistant turn enters a complex multi-stage agent flow such as planner, executor, and replanner
- **THEN** the copilot MUST show a `Deep thinking` card in loading state using Ant Design X `Think`
- **AND** that card MUST remain collapsible while the turn is still running
- **AND** the executor action chain inside that card MUST be rendered with Ant Design X `ThoughtChain`

#### Scenario: Skip thought chain for simple QA turn
- **WHEN** an assistant turn only transfers to a simple QA-style agent and returns a direct answer
- **THEN** the copilot MUST be allowed to omit the complex thought chain UI
- **AND** the user-visible response MUST still render normally as a standard assistant message

### Requirement: `run_path` SHALL define thought-chain hierarchy
The system SHALL use `run_path` as the primary source for organizing complex-agent stages, and executor tool calls MUST appear as child items beneath the corresponding executor stage.

#### Scenario: Build hierarchy from run path
- **WHEN** the stream includes planner, executor, and replanner events with nested `run_path` values
- **THEN** the copilot MUST assign those events to the corresponding stage in the thought chain using `run_path`
- **AND** repeated executor or replanner rounds MUST remain distinguishable as separate progress segments in the same turn

#### Scenario: Attach tool call under executor
- **WHEN** an executor stage emits one or more tool calls
- **THEN** each tool call MUST appear as a thought-chain child item under the active executor stage
- **AND** related tool status or result information MUST update the same child item rather than creating unrelated top-level stages

### Requirement: Official Ant Design X thinking components SHALL be the default UI primitives
The copilot thought-process surface MUST use Ant Design X `Think` and `ThoughtChain` as its default UI primitives instead of introducing a parallel custom thought-card or timeline pattern when the official components satisfy the interaction.

#### Scenario: Build thought-process UI with official components
- **WHEN** the copilot renders a complex-agent thought process
- **THEN** the outer collapsible container MUST use Ant Design X `Think`
- **AND** the nested action chain MUST use Ant Design X `ThoughtChain`
- **AND** implementation-specific styling or wrappers MUST NOT replace those official components with an unrelated custom structure

### Requirement: Planner and replanner narration SHALL stay outside the thought chain
Planner and replanner textual narration MUST be shown as user-visible summary content outside the thought chain so the action chain remains focused on what the system did.

#### Scenario: Stream planner output as summary text
- **WHEN** the planner emits visible textual planning output
- **THEN** the copilot MUST stream that content in the assistant summary area outside the thought chain
- **AND** the planner output MUST NOT be forced into the executor tool-call chain

#### Scenario: Stream replanner step updates as summary text
- **WHEN** the replanner emits step-oriented progress or explanation text before the final response exists
- **THEN** the copilot MUST continue streaming that content in the summary area outside the thought chain
- **AND** the thought chain MUST remain focused on executor actions and nested tool activity

### Requirement: Final response SHALL take over once replanner response appears
When the replanner produces a final `response`, the copilot MUST switch the main visible answer area from transient process summaries to the final response while retaining earlier process context in collapsed form.

#### Scenario: Handoff from transient summary to final response
- **WHEN** the stream first yields a replanner `response`
- **THEN** the copilot MUST begin streaming that response as the primary visible assistant answer
- **AND** previously streamed planner or replanner summaries MUST collapse by default instead of continuing as the main answer

#### Scenario: Preserve process context after response handoff
- **WHEN** the main answer has switched to the replanner response
- **THEN** the user MUST still be able to expand the thought-process area to inspect prior executor tool calls and collapsed planning context

### Requirement: Replanner completion SHALL close the thought chain
For complex plan-execute-replan agents, the thought chain MUST remain loading until the final replanner completion signal indicates the run has converged.

#### Scenario: End thought chain on replanner completion
- **WHEN** the final replanner stage emits its completion signal after the final response phase
- **THEN** the `Deep thinking` card MUST transition from loading to completed
- **AND** the thought chain for that assistant turn MUST stop accepting further stage progression updates
