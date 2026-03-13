## MODIFIED Requirements

### Requirement: AI session APIs MUST expose structured turn replay

The system MUST define AI session replay responses that expose structured assistant turn and block history for new consumers.

#### Scenario: session detail includes turns
- **WHEN** a client requests AI session detail
- **THEN** the response MUST include ordered replay data for turns and their blocks
- **AND** each assistant turn MUST expose stable identifiers, lifecycle status, timestamps, and ordered blocks sufficient for UI reconstruction
- **AND** the contract MUST remain capable of replaying both legacy `internal/ai` turns and `aiv2` single-agent turns

## MODIFIED Requirements

### Requirement: AI session APIs MUST preserve legacy message-compatible fields during rollout

The system MUST preserve message-oriented session compatibility for existing clients while introducing structured turn replay fields.

#### Scenario: compatibility client continues reading messages
- **WHEN** a legacy client requests session history
- **THEN** the response MUST still include message-compatible fields used by the current frontend
- **AND** the addition of structured turn replay fields MUST NOT remove required existing compatibility fields during rollout
- **AND** `aiv2` session storage and replay MUST continue to populate those compatibility fields until the frontend no longer depends on them

## MODIFIED Requirements

### Requirement: AI API contract types MUST be declared in api/ai/v1

The system MUST define the request and response contract for structured AI session replay in `api/ai/v1/ai.go` so frontend and backend use one stable contract source.

#### Scenario: AI replay contract lives in api package
- **WHEN** developers add or update structured AI session replay fields for `aiv2`
- **THEN** those request and response structs MUST be declared in `api/ai/v1/ai.go`
- **AND** handler-local response-only structs MUST NOT become the canonical replay contract
- **AND** the contract MUST explicitly permit runtime-source metadata so replay consumers can distinguish legacy and `aiv2` sessions when needed
