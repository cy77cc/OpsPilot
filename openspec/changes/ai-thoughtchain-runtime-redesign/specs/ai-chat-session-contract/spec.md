## MODIFIED Requirements

### Requirement: AI session APIs MUST expose structured turn replay

The system MUST define AI session replay responses that expose structured assistant replay data as native ThoughtChain state so live streaming and restored sessions can use the same rendering model.

#### Scenario: session detail includes chain replay
- **WHEN** a client requests AI session detail
- **THEN** the response MUST include ordered assistant replay data containing chain identity, node identity, node kind, lifecycle status, timestamps, and final answer relationship sufficient for UI reconstruction
- **AND** the replay data MUST allow the frontend to render the same ThoughtChain-oriented assistant view used for live streaming

#### Scenario: canonical replay contains final answer markdown
- **WHEN** a completed assistant turn is persisted for later replay
- **THEN** the canonical replay payload MUST include final-answer markdown content and enough runtime state to reproduce its reveal relationship to process nodes
- **AND** the system MUST NOT rely on empty or missing compatibility blocks as the only persisted assistant output

### Requirement: AI session APIs MUST preserve legacy message-compatible fields during rollout

The system MUST preserve only the compatibility fields required during the migration window, while making ThoughtChain replay the canonical assistant runtime representation.

#### Scenario: canonical replay representation is chain-based
- **WHEN** backend and frontend exchange AI session replay data during the redesign rollout
- **THEN** the canonical assistant replay model MUST be the ThoughtChain structure
- **AND** any temporary compatibility message fields MUST NOT redefine the primary replay semantics

#### Scenario: restore prefers runtime-first replay data
- **WHEN** a client restores assistant history during or after the rollout
- **THEN** the client and server contract MUST prefer persisted ThoughtChain runtime data and final-answer content before compatibility `blocks`
- **AND** narrow compatibility fallbacks MUST only apply when canonical runtime replay data is unavailable

#### Scenario: completed assistant turn cannot persist as empty replay
- **WHEN** an assistant turn reaches a completed terminal state
- **THEN** the persisted session contract MUST contain recoverable runtime or final-answer replay data for that turn
- **AND** a completed assistant turn with effectively empty replay content MUST be treated as invalid

#### Scenario: compatibility fields can be removed after migration
- **WHEN** all primary chat consumers have migrated to the ThoughtChain replay contract
- **THEN** the system MAY remove temporary message-compatible fields
- **AND** the removal MUST NOT require a second replay model to remain authoritative
