## ADDED Requirements

### Requirement: AI core MUST host the primary orchestration boundary

The system MUST host the primary AIOps orchestration boundary inside `internal/ai`, including orchestration ownership for AI task coordination, interrupt-aware execution flow, and platform-level AI semantics.

#### Scenario: backend orchestration host is defined in the AI core
- **WHEN** maintainers inspect the AI backend architecture
- **THEN** the orchestration host for AI task coordination MUST be implemented under `internal/ai`
- **AND** `internal/service/ai` MUST not be the architectural owner of AI orchestration behavior

### Requirement: AI core MUST provide a stable orchestration entrypoint for gateway use

The AI core MUST expose a stable orchestration entrypoint that gateway handlers can call for chat, interrupt, and resume flows without re-implementing orchestration semantics in handler code.

#### Scenario: gateway delegates chat execution to the AI core
- **WHEN** an `/api/v1/ai` chat request is handled
- **THEN** the gateway MUST delegate orchestration to an entrypoint hosted in `internal/ai`
- **AND** the gateway MUST remain responsible only for request mapping, auth/session shell concerns, and transport delivery

### Requirement: AI core MUST be shaped for plan-execute-replan evolution

The orchestration core MUST be organized so that planning, execution, and replanning responsibilities can evolve inside `internal/ai` without another handler-centered rewrite.

#### Scenario: future orchestration roles have a defined host
- **WHEN** maintainers evolve the AI backend toward ADK `plan-execute-replan`
- **THEN** planner, executor, and replanner responsibilities MUST have a defined home inside `internal/ai`
- **AND** the migration MUST not depend on moving orchestration ownership back into gateway packages

### Requirement: AI core MUST own platform event semantics

The AI core MUST own the semantic meaning of AI platform events, including execution progress, interrupt meaning, and future AIOps operational events, even if transport framing remains in the gateway.

#### Scenario: execution semantics are emitted from the AI core
- **WHEN** tool execution, interrupt, or orchestration progress is surfaced to clients
- **THEN** the meaning of those events MUST originate from the AI core
- **AND** gateway code MUST only frame or serialize those events for transport compatibility
