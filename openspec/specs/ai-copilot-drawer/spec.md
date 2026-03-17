# AI Copilot Drawer

## Purpose

Provide a global AI assistant entry point accessible from anywhere in the authenticated application shell, implemented as a drawer-style surface that maintains page context while enabling session-oriented AI interactions.

## Requirements

### Requirement: Global copilot drawer entry
The system SHALL provide a global AI copilot entry in the authenticated application shell and MUST open the assistant inside a drawer-style surface without forcing navigation away from the current page.

#### Scenario: Open copilot from application shell
- **WHEN** an authenticated user clicks the AI copilot entry from the main layout header
- **THEN** the system MUST open the AI assistant as a drawer or equivalent side surface within the current shell
- **AND** the current route and page state MUST remain intact behind the assistant surface

#### Scenario: Open copilot with keyboard shortcut
- **WHEN** an authenticated user triggers the configured global AI shortcut
- **THEN** the system MUST open the same drawer-based copilot surface instead of navigating to a disconnected standalone page

### Requirement: Copilot drawer SHALL support session-oriented chat workflow
The copilot drawer SHALL support session history, session switching, new session creation, and streaming assistant responses by integrating with the existing AI session and chat APIs.

#### Scenario: Resume an existing session
- **WHEN** the user selects an existing conversation in the copilot drawer
- **THEN** the system MUST load that session's persisted messages
- **AND** subsequent prompts MUST continue within the same session unless the user starts a new conversation

#### Scenario: Start a new session
- **WHEN** the user starts a new conversation from the copilot drawer
- **THEN** the system MUST create or initialize a new chat session context
- **AND** the new session MUST appear in the conversation list with a generated title after the first user message is sent

#### Scenario: Stream assistant output
- **WHEN** the assistant returns a streaming response from the AI chat endpoint
- **THEN** the copilot drawer MUST render incremental assistant output in the active conversation
- **AND** the final message state MUST settle to success, abort, or error according to the stream outcome

### Requirement: Assistant markdown rendering SHALL use Ant Design X Markdown
The assistant message body SHALL be rendered through `@ant-design/x-markdown` as the primary markdown renderer, and the UI MUST NOT rely on ad hoc newline-to-HTML conversion or a parallel markdown rendering stack for the main assistant content.

#### Scenario: Render markdown response with rich structure
- **WHEN** the assistant returns markdown containing headings, lists, code fences, or tables
- **THEN** the system MUST render that content through `@ant-design/x-markdown`
- **AND** the rendered output MUST preserve markdown semantics rather than flattening the response into plain text or injected `<br/>` fragments

#### Scenario: Extend markdown rendering for AI-specific tags
- **WHEN** the assistant output includes supported AI-specific markdown extensions such as think-style blocks
- **THEN** the system MUST extend `@ant-design/x-markdown` via supported component customization points
- **AND** the implementation MUST avoid replacing the primary renderer with a custom markdown pipeline

### Requirement: Copilot drawer SHALL degrade safely when AI surface fails
The AI copilot surface MUST isolate its initialization and runtime failures so that main application navigation and page content remain usable even if the drawer cannot initialize.

#### Scenario: AI surface initialization fails
- **WHEN** the copilot drawer encounters an initialization or provider failure
- **THEN** the system MUST contain the failure within the AI surface boundary
- **AND** the main application shell MUST continue rendering and remain interactive
