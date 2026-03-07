## ADDED Requirements

### Requirement: AI assistant drawer MUST remain backend-compatible during AI core refactor

The system MUST preserve the current assistant drawer's ability to consume backend AI streaming responses while orchestration ownership moves from gateway-centric execution toward an AI-core-owned backend architecture.

#### Scenario: assistant drawer remains compatible during backend ownership migration
- **WHEN** the backend AI implementation is refactored so orchestration is hosted in `internal/ai`
- **THEN** the assistant drawer MUST continue to operate against the existing `/api/v1/ai` surfaces in this phase
- **AND** the refactor MUST not require a simultaneous drawer protocol rewrite
