## ADDED Requirements

### Requirement: AI messages MUST be normalized into extensible render blocks
The system MUST normalize assistant messages into typed render blocks before rich rendering so that markdown, code, thinking, recommendations, and future structured content can evolve independently.

#### Scenario: Normalize assistant response into blocks
- **WHEN** the AI assistant receives a response payload containing rich content
- **THEN** the frontend normalizes the content into one or more typed render blocks before rendering
- **AND** each block carries enough data to render or safely degrade without re-parsing the entire message tree

### Requirement: Rich render blocks MUST fail independently
The system MUST isolate each rich render block so that a renderer failure affects only the failed block and not the surrounding message list or assistant panel.

#### Scenario: Code renderer fails
- **WHEN** a code block renderer throws during render
- **THEN** the failed block is replaced with a safe fallback representation
- **AND** the rest of the message continues rendering
- **AND** the assistant panel remains usable

### Requirement: Rich render blocks MUST provide safe fallback output
The system MUST provide a textual or otherwise safe fallback for each supported rich block type when advanced rendering cannot complete.

#### Scenario: Unsupported future block type
- **WHEN** the frontend encounters a block type without an available rich renderer
- **THEN** the system renders a safe fallback output for that block
- **AND** the user can still understand the message at a coarse level

### Requirement: AI message rendering MUST support additive block types
The system MUST support adding new block types without requiring unrelated changes to existing block renderers.

#### Scenario: Add recommendation block support
- **WHEN** the assistant message includes a recommendations block
- **THEN** the recommendations block is rendered through a dedicated renderer
- **AND** existing markdown, code, and thinking block renderers do not require behavioral changes to keep working
