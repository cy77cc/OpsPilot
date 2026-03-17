## MODIFIED Requirements

### Requirement: Copilot drawer SHALL support session-oriented chat workflow
The copilot drawer SHALL support session history, session switching, new session creation, and streaming assistant responses by integrating with the existing AI session and chat APIs. For complex agent runs, the drawer MUST separate process visibility from the main final-answer area instead of treating all assistant output as one flat markdown stream.

#### Scenario: Stream assistant output for simple response
- **WHEN** the assistant returns a normal streaming response from the AI chat endpoint without complex-agent thought-chain signals
- **THEN** the copilot drawer MUST render incremental assistant output in the active conversation
- **AND** the final message state MUST settle to success, abort, or error according to the stream outcome

#### Scenario: Stream complex agent output with split process and answer areas
- **WHEN** the assistant turn produces complex-agent planning, execution, or replanning signals
- **THEN** the copilot drawer MUST render the thought-process card separately from the main visible answer area
- **AND** that thought-process card MUST use Ant Design X `Think` with Ant Design X `ThoughtChain` inside for action-chain rendering
- **AND** the main visible answer area MUST be able to switch from transient process summary text to the final replanner response when that response becomes available
