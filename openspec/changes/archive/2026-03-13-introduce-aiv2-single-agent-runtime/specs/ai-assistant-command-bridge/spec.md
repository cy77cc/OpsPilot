## MODIFIED Requirements

### Requirement: AI assistant gateway MUST preserve a stable command bridge

The AI assistant gateway MUST preserve a stable bridge between gateway chat requests and AI-core orchestration entrypoints. The bridge SHALL allow the rollout of new orchestration implementations without changing existing request or SSE transport contracts at the gateway boundary, while permitting additive turn/block streaming endpoints.

#### Scenario: gateway keeps the same bridge while orchestration evolves
- **WHEN** maintainers switch the AI runtime from the legacy multi-stage path to the `aiv2` single-agent path
- **THEN** the gateway-facing command bridge MUST remain stable
- **AND** chat requests MUST still delegate through the existing AI gateway module
- **AND** existing `/api/v1/ai/chat` and `/api/v1/ai/resume/step` compatibility semantics MUST remain available
- **AND** additive turn/block streaming endpoints MAY be introduced without removing those existing gateway contracts

## MODIFIED Requirements

### Requirement: command bridge rollout MUST be gated by configuration

The command bridge MUST support selecting the multi-stage orchestration path through `ai.use_multi_domain_arch`. The bridge MUST also support selecting turn/block streaming rollout through `ai.use_turn_block_streaming`. The bridge MUST additionally support selecting the `aiv2` single-agent runtime through a dedicated runtime selector or rollout flag. Rollout controls SHALL preserve a stable fallback to the existing runtime when `aiv2` is disabled.

#### Scenario: aiv2 runtime path is enabled
- **WHEN** operators enable the `aiv2` single-agent runtime
- **THEN** AI chat and streaming resume requests MUST enter the `internal/aiv2` runtime path
- **AND** disabling the selector MUST restore the current `internal/ai` control-plane path
- **AND** the gateway contract MUST remain the same for clients

#### Scenario: turn-block streaming rollout remains compatible
- **WHEN** operators set `ai.use_turn_block_streaming=true`
- **THEN** the gateway MUST allow turn/block-native streaming and persisted replay behavior for compatible consumers
- **AND** compatibility SSE events and legacy message-compatible APIs MUST remain available during rollout
- **AND** this behavior MUST remain true whether the active backend runtime is `internal/ai` or `internal/aiv2`

## MODIFIED Requirements

### Requirement: gateway MUST provide a dedicated streaming resume bridge

The gateway MUST preserve `/api/v1/ai/resume/step` as a compatibility JSON control endpoint and MUST provide `/api/v1/ai/resume/step/stream` as the dedicated streaming resume endpoint for turn/block-native clients.

#### Scenario: streaming resume uses a dedicated endpoint
- **WHEN** a client needs resumed execution to continue streaming on the active assistant turn
- **THEN** the client MUST be able to call `/api/v1/ai/resume/step/stream`
- **AND** the system MUST continue the interrupted turn through SSE from that runtime point
- **AND** the existing `/api/v1/ai/resume/step` endpoint MUST remain available for compatibility JSON resume flows
- **AND** the bridge MUST allow the active backend runtime to map the resume request into either legacy execution identity or `aiv2` checkpoint-backed identity without changing the public endpoint
