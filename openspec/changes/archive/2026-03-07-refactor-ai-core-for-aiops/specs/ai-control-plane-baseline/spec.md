## MODIFIED Requirements

### Requirement: AI Chat SHALL Support SSE Streaming Contract
AI chat capability SHALL be specified with an SSE event contract that includes message streaming and completion signaling, while preserving a separation between gateway transport responsibilities and AI core event semantics.

#### Scenario: SSE event family is defined
- **WHEN** reviewers inspect the AI baseline
- **THEN** the spec SHALL include `meta`, `delta`, `thinking_delta`, `tool_call`, `tool_result`, `approval_required`, `done`, and `error` as baseline stream events
- **AND** the gateway SHALL remain responsible for transport framing and delivery compatibility
- **AND** the AI core SHALL remain responsible for the semantic meaning of streamed execution and interrupt events

## ADDED Requirements

### Requirement: AI gateway and AI core MUST have separate ownership boundaries
The baseline MUST define `internal/service/ai` as the gateway surface for AI APIs and streaming, and `internal/ai` as the owner of AI orchestration and control-plane semantics.

#### Scenario: ownership boundary is documented
- **WHEN** maintainers inspect the AI control-plane baseline
- **THEN** the baseline MUST describe `internal/service/ai` as handling routes, auth-aware request mapping, and transport delivery
- **AND** the baseline MUST describe `internal/ai` as handling orchestration, execution semantics, interrupt-aware flow, and AI platform behavior

### Requirement: control-plane capabilities MUST remain consumable by the AI core
The baseline MUST require that approvals, preview flows, execution records, and session-oriented AI control-plane capabilities remain available to the AI core as dependencies during backend refactoring.

#### Scenario: existing control-plane surfaces remain available through AI core refactor
- **WHEN** the backend is refactored toward an AI-core-owned orchestration model
- **THEN** approval, execution, preview, and session capabilities MUST remain available
- **AND** the refactor MUST not require immediate removal of existing `/api/v1/ai` control-plane endpoints
