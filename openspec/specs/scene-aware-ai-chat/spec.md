# Scene-Aware AI Chat

## Purpose

Enable the AI chat system to operate within contextual boundaries defined by the active application scene, providing scene-specific session management, hidden context augmentation, and relevant quick prompts without polluting the user-visible conversation transcript.

## Requirements

### Requirement: Chat requests SHALL carry scene identity
The AI chat system SHALL accept an explicit scene identifier for each chat request and MUST associate new or continued sessions with that scene instead of defaulting all conversations to a generic global AI bucket.

#### Scenario: Start chat from scene-specific page
- **WHEN** a user opens the copilot from a scene-specific page such as host, cluster, service, or Kubernetes operations
- **THEN** the chat request MUST include the resolved scene identifier
- **AND** any new session created from that request MUST be persisted under the same scene

#### Scenario: Filter sessions by scene
- **WHEN** the client requests AI sessions for a specific scene
- **THEN** the system MUST return only sessions associated with that scene
- **AND** the copilot UI MUST be able to distinguish scene-specific history from generic AI history

### Requirement: Scene-aware chat SHALL distinguish scene identity from scene context
The system SHALL model scene identity separately from scene context so that the chat pipeline can use both domain-level routing and entity-level grounding.

#### Scenario: Include entity context with scene
- **WHEN** the user invokes copilot while viewing a specific resource
- **THEN** the chat pipeline MUST be able to include structured scene context such as route, resource type, resource identifier, or resource name alongside the scene identifier
- **AND** the system MUST preserve the user-visible message text separately from that hidden context payload

### Requirement: Scene prompts and tool hints SHALL be injected as hidden augmentation
Scene prompts, tool hints, and scene configuration constraints SHALL be applied as hidden augmentation inputs to the chat pipeline and MUST NOT be injected into the user-visible message transcript as if the user typed them.

#### Scenario: Use scene prompts without polluting transcript
- **WHEN** a scene defines prompt templates or quick actions relevant to the current page
- **THEN** the system MUST be able to use those prompts to augment the chat request or provider context
- **AND** the visible user message in the conversation transcript MUST remain the user's original input

#### Scenario: Apply scene tool constraints
- **WHEN** a scene defines allowed tools, blocked tools, or other execution constraints
- **THEN** the chat pipeline MUST make those constraints available to backend routing or prompt construction
- **AND** the system MUST use them to improve tool relevance without exposing internal constraint text as assistant or user chat content

### Requirement: Scene-aware quick prompts SHALL reflect the active page domain
The copilot UI SHALL expose scene-relevant quick prompts or recommendations derived from the active scene so that users can start common tasks without manually restating the current domain.

#### Scenario: Show scene-specific quick prompts
- **WHEN** the copilot drawer opens on a page with a resolved active scene
- **THEN** the system MUST display quick prompts, suggestions, or starter actions scoped to that scene
- **AND** selecting one of those prompts MUST start a chat request using the same scene-aware pipeline as typed input
