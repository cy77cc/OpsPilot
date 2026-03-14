## ADDED Requirements

### Requirement: Prometheus integration MUST ingest AI thoughtChain runtime metrics

The system MUST expose AI thoughtChain runtime metrics through the existing Prometheus integration so operators can observe chain throughput, latency, approvals, replans, and terminal outcomes.

#### Scenario: chain metrics are exported
- **WHEN** AI thoughtChain callbacks record chain lifecycle activity
- **THEN** Prometheus-accessible metrics MUST include chain totals, completion or failure counts, and duration histograms
- **AND** metric labels MUST include scene and terminal status when available

#### Scenario: node-level approval and tool metrics are exported
- **WHEN** AI thoughtChain callbacks record tool execution, approval wait, or replan activity
- **THEN** Prometheus-accessible metrics MUST include node-level durations or counters for those events
- **AND** labels MUST preserve node kind, tool name, and approval outcome when applicable
