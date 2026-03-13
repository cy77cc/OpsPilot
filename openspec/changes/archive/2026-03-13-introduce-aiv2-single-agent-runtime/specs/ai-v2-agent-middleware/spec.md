## ADDED Requirements

### Requirement: AIV2 agent SHALL apply middleware for cross-cutting runtime concerns

The `aiv2` runtime SHALL apply `ChatModelAgentMiddleware` to handle cross-cutting runtime concerns around the single-agent execution path. Middleware MUST be the primary extension point for agent context injection, approval policy enforcement, streaming event projection, and runtime observability.

#### Scenario: middleware chain is applied to agent runs
- **WHEN** the `aiv2` runtime creates or resumes a chat run
- **THEN** the main `ChatModelAgent` MUST run with a configured middleware chain
- **AND** the middleware chain MUST execute consistently for both initial runs and resumed runs

### Requirement: Context injection middleware SHALL provide stable operational context

The `aiv2` runtime SHALL inject scene, route, selected resources, project context, and other request-scoped metadata through middleware rather than ad-hoc prompt assembly inside handlers.

#### Scenario: request context is injected before agent reasoning
- **WHEN** a user sends a request to the `aiv2` runtime
- **THEN** middleware MUST augment the agent context with scene-aware runtime metadata
- **AND** the same context shape MUST remain available when the run is resumed after approval

### Requirement: Approval policy middleware SHALL coordinate tool gating with interrupts

Approval policy middleware SHALL coordinate approval gating for mutating tools and MUST produce consistent interrupt metadata for downstream resume flows.

#### Scenario: middleware recognizes a gated tool invocation
- **WHEN** the agent selects a tool whose policy requires approval
- **THEN** middleware MUST identify the invocation as gated before the tool mutates the target system
- **AND** the runtime MUST emit approval metadata sufficient for the frontend approval UI and resume bridge

### Requirement: Streaming middleware SHALL project agent events to the existing frontend contract

Streaming middleware SHALL project single-agent runtime events into the existing frontend-compatible event model so the chat UI can be reused without a full rewrite.

#### Scenario: middleware maps agent activity to SSE events
- **WHEN** the single agent produces thinking output, tool activity, approval interruptions, and final answer content
- **THEN** middleware MUST emit frontend-compatible `thinking_delta`, `tool_call`, `tool_result`, `approval_required`, `delta`, `done`, and `error` semantics
- **AND** the projected events MUST remain bound to the active assistant turn identity

### Requirement: Observability middleware SHALL expose single-agent runtime costs

Observability middleware SHALL record single-agent runtime costs so operators can compare the new runtime against the legacy multi-stage runtime.

#### Scenario: middleware records runtime metrics
- **WHEN** a request is processed by `aiv2`
- **THEN** middleware MUST capture model latency, tool latency, interrupt count, and resume outcomes
- **AND** the resulting telemetry MUST distinguish `aiv2` runs from legacy `internal/ai` runs
