## ADDED Requirements

### Requirement: AI runtime SHALL emit callback-based thoughtChain observability hooks
The system MUST emit callback hooks for thoughtChain and node lifecycle so runtime telemetry can be collected independently from frontend rendering.

#### Scenario: chain lifecycle triggers callbacks
- **WHEN** an assistant thoughtChain starts, pauses, resumes, completes, or fails
- **THEN** the runtime MUST emit corresponding callback hooks with stable `trace_id`, `chain_id`, `session_id`, and status metadata

#### Scenario: node lifecycle triggers callbacks
- **WHEN** a thoughtChain node opens, updates, closes, or resolves approval
- **THEN** the runtime MUST emit callbacks containing `node_id`, `node_kind`, and relevant execution context such as `tool`, `scene`, or approval outcome

### Requirement: AI runtime SHALL export thoughtChain metrics to Prometheus
The system MUST translate thoughtChain observability callbacks into Prometheus-compatible metrics through the existing monitoring integration.

#### Scenario: chain metrics are recorded
- **WHEN** a thoughtChain completes or fails
- **THEN** the system MUST record counters and duration metrics for the chain outcome
- **AND** exported labels MUST include enough context to distinguish scene and terminal status

#### Scenario: approval and replan metrics are recorded
- **WHEN** approval waits or replans occur in a chain
- **THEN** the system MUST record approval counts, approval wait duration, and replan counts
- **AND** the exported metrics MUST preserve labels for node kind and approval outcome when applicable
