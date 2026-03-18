# AI Assistant Reply Design

Date: 2026-03-18

## Goal

Land the Figma Make assistant reply experience into the existing AI copilot surface, but only for the assistant response area.

The current product color system stays unchanged. The redesign focuses on how assistant replies are organized and rendered so that structured information feels like one continuous answer instead of a stack of visible cards.

## Scope

In scope:

- Assistant reply presentation inside the existing copilot drawer
- Real-time streaming rendering for assistant replies
- Mapping backend SSE events into a unified assistant reply runtime model
- Rendering structured report content without obvious card boundaries
- Backward compatibility for existing plain markdown replies and historical sessions

Out of scope:

- Global AI surface redesign
- User message styling changes
- New backend event types
- Major session/history API redesign
- Full Figma parity for the entire assistant shell

## Product Direction

The selected direction is a "segmented integrated flow".

The assistant response should read like one composed answer with internal rhythm:

1. Lightweight assistant identity and active phase
2. Brief stage summary
3. Embedded process updates
4. Structured result sections merged into the body
5. Final recommendations and completion state

The UI should avoid the feeling of separate dashboard widgets inside chat. Structure still exists, but it should be expressed through spacing, typography, subtle separators, and local tonal shifts instead of strong card borders.

## Current State

The current frontend chat flow treats assistant output primarily as a streaming markdown string.

- `web/src/components/AI/providers/PlatformChatProvider.ts` receives SSE events and currently uses them to emit placeholder status text plus appended visible content.
- `web/src/components/AI/CopilotSurface.tsx` renders assistant replies through `Bubble.List` and `XMarkdown`.
- `web/src/api/modules/ai.ts` already exposes event handlers for `meta`, `agent_handoff`, `plan`, `replan`, `delta`, `tool_call`, `tool_approval`, `tool_result`, `done`, and `error`.

This means the backend already provides enough process information, but the UI does not yet organize that information into a coherent assistant reply layout.

## Proposed Architecture

### 1. Assistant Runtime Model

Extend the assistant message model with an optional runtime payload on the frontend.

The runtime payload should describe the visible reply state instead of mirroring raw transport events. It should include:

- Current phase
- Human-readable activity feed
- Structured report segments
- Completion or interruption status

This runtime state is owned by the frontend. The backend remains event-driven; the frontend is responsible for translating event noise into user-readable presentation.

### 2. Event-to-Reply Mapping

Backend SSE events map into four presentation groups.

#### Phase

Source events:

- `meta`
- `agent_handoff`
- `plan`
- `replan`

Purpose:

- Show the current stage of work
- Update the assistant header and stage summary

Rendering:

- Small status line at the top of the assistant reply
- Replaceable and progressive, not a separate card

#### Activity

Source events:

- `tool_call`
- `tool_approval`
- `tool_result`
- recoverable `error`

Purpose:

- Reveal meaningful work as the run progresses
- Keep users oriented without exposing raw execution logs

Rendering:

- Inline activity rows with low visual weight
- Only user-relevant actions should be shown
- Avoid dumping raw tool arguments or verbose result payloads

#### Report

Source events:

- `delta`
- final accumulated markdown content
- selectively extracted structure from reply content when possible

Purpose:

- Hold the primary answer, findings, table-like result areas, and recommendations

Rendering:

- Continuous body content
- Structured sections embedded directly in the answer flow
- Visual grouping through spacing, local labels, soft dividers, and typography
- No hard card boundaries for each report section

#### Status

Source events:

- `done`
- non-recoverable `error`
- hard timeout

Purpose:

- Mark final completion or interruption

Rendering:

- Lightweight footer status at the end of the reply
- Must not dominate the answer

## Rendering Design

### Assistant Reply Composition

Assistant replies should render as a single composition with the following optional pieces:

1. Reply chrome
2. Phase summary
3. Activity feed
4. Main markdown/report body
5. Inline structured summary blocks
6. Completion footer

These pieces belong to one assistant message and should visually read as one reply.

### Visual Rules

- Keep the current product palette
- Do not restyle the entire drawer to match the Figma Make shell
- Do not introduce thick borders around each section
- Prefer background tint, spacing, monospace accents, and subtle separators
- Preserve readability for long markdown, code blocks, and tables

### Structured Content Strategy

Structured result areas should feel embedded into the answer flow. For example:

- A short section label before a host or cluster summary
- A soft list or grid for compact status summaries
- A table rendered inline with restrained chrome
- Recommendations rendered as the natural continuation of the same answer

The design target is "composed report in chat", not "chat plus mini-dashboard".

## Implementation Plan

### Data Layer

Files:

- `web/src/components/AI/types.ts`
- `web/src/components/AI/providers/PlatformChatProvider.ts`

Changes:

- Extend frontend assistant message typing to support optional runtime metadata
- Build event reducers/helpers that translate SSE events into presentation-oriented runtime state
- Continue producing plain content for compatibility

### Presentation Layer

Files:

- `web/src/components/AI/CopilotSurface.tsx`
- possibly a new assistant-reply-focused render helper/component under `web/src/components/AI/`

Changes:

- Replace assistant markdown-only rendering with a unified assistant reply renderer
- Render runtime state above, within, and below the markdown body as one integrated flow
- Keep user message rendering unchanged

### Compatibility Layer

Requirements:

- Historical messages without runtime metadata must still render correctly
- If a session replay only contains plain `content`, fallback to current markdown rendering
- Streaming behavior must remain smooth during partial updates

## Data Shape Guidance

The assistant runtime type should be intentionally narrow. Recommended categories:

- `phase`
- `activities`
- `summary`
- `status`

Avoid exposing raw backend transport payloads directly to the renderer. Convert them into stable, user-facing primitives first.

## Error Handling

- Recoverable delays such as slow tool execution should appear as lightweight status hints, not terminal failures
- Hard failures should preserve any already-rendered reply content and append a restrained terminal error state
- Timeout handling must not erase previously streamed content

## Testing Strategy

Add focused tests for:

- Event mapping from SSE handlers into assistant runtime state
- Rendering with runtime-enhanced assistant messages
- Fallback rendering for plain markdown-only assistant messages
- Error and timeout cases preserving partial content

Regression coverage should include existing assistant markdown streaming behavior.

## Risks

### Message Model Constraints

The current `@ant-design/x-sdk` message shape is minimal. The implementation may need a local adapter layer so runtime metadata can exist without breaking current message flow.

### Event Payload Stability

`tool_result.content` may not always be structured enough for deterministic parsing. The UI should only extract gentle structure where the payload is stable and otherwise keep content inline as text.

### Replay vs Live Divergence

Live streaming replies may contain richer runtime state than historical session fetches. The renderer must tolerate both without inconsistent layout failures.

## Success Criteria

- Assistant replies feel like one integrated response instead of stacked cards
- Existing color styling remains aligned with the current product
- Backend events become visible in a useful, non-loggy way
- Historical messages still render correctly
- No regression in streaming markdown rendering

## Open Questions

The current implementation can proceed without blocking questions, but these may affect polish:

- Whether session replay should eventually persist frontend-friendly runtime blocks
- Whether certain tool results should gain domain-specific visual formatting later
- Whether the activity feed should collapse automatically after completion
