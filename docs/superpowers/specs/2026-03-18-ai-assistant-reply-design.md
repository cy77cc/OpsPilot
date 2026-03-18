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

### 1.1 Message Contract

The implementation must define an explicit adapter contract at the seams between streaming, persistence, and rendering.

Recommended frontend shape:

```ts
type AssistantReplyPhase =
  | 'preparing'
  | 'identifying'
  | 'planning'
  | 'executing'
  | 'summarizing'
  | 'completed'
  | 'interrupted';

type AssistantReplyActivityKind =
  | 'agent_handoff'
  | 'plan'
  | 'replan'
  | 'tool_call'
  | 'tool_approval'
  | 'tool_result'
  | 'hint'
  | 'error';

interface AssistantReplyActivity {
  id: string;
  kind: AssistantReplyActivityKind;
  label: string;
  detail?: string;
  status?: 'pending' | 'active' | 'done' | 'error';
  createdAt?: string;
}

interface AssistantReplyRuntime {
  phase?: AssistantReplyPhase;
  phaseLabel?: string;
  activities: AssistantReplyActivity[];
  summary?: {
    title?: string;
    items?: Array<{ label: string; value: string; tone?: 'default' | 'success' | 'warning' | 'danger' }>;
  };
  status?: {
    kind: 'streaming' | 'completed' | 'soft-timeout' | 'error';
    label: string;
  };
}

interface XChatMessage {
  role: 'user' | 'assistant';
  content: string;
  runtime?: AssistantReplyRuntime;
}
```

Contract rules:

- The provider is responsible for producing `content` plus optional `runtime` for live assistant messages.
- Persisted history may contain richer backend fields such as `runtime`, `thoughtChain`, `traces`, `recommendations`, `turns`, and `blocks`.
- The surface renderer must consume `XChatMessage.runtime` when present, but must continue to render correctly when only `content` exists.
- `Bubble.List` item construction must preserve the full assistant message object, not just `content`, so assistant rendering can use both.

### 1.2 History and Replay Precedence

The renderer must use these precedence rules:

1. If a persisted assistant message already contains a compatible `runtime`, prefer it.
2. Else, if replay `turns` or `blocks` can be deterministically mapped into the new runtime model, synthesize runtime from them.
3. Else, if only plain `content` exists, render markdown-only fallback.

For history loading:

- `messages[]` remains the primary source for initial compatibility.
- `turns[]` and `blocks[]` are an enrichment source, not a replacement, unless implementation explicitly migrates to replay-first rendering.
- Historical `status` values must map into footer state:
  - `done` -> `completed`
  - incomplete or loading -> `streaming`
  - known error states -> `error`

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
- Only one phase is visible at a time
- Newer phase updates replace older phase text

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
- Activities are append-only once shown, except when explicitly coalesced
- Duplicate consecutive tool events for the same `call_id` should collapse into one evolving activity row
- Activities remain visible after the first `delta`, but their weight should reduce once the main body starts streaming

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

Reducer rules:

- `delta` content remains the source of truth for the primary markdown body
- The first visible `delta` replaces placeholder-only assistant content such as `[准备中]`
- Any structured summary extracted from events or final content must not duplicate text already shown verbatim in the markdown body
- Best-effort extraction is allowed only for stable patterns. If parsing confidence is low, keep content in markdown only

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

Reducer rules:

- `done` marks the reply as `completed` even if no additional markdown arrives
- Non-recoverable `error` marks the reply as `error` and preserves already streamed content
- `tool_timeout_soft` maps to a temporary `soft-timeout` hint and should disappear after the next successful progress event
- `tool_timeout_hard` maps to terminal `error` unless the request later resumes, which the current implementation does not expect

### 2.1 Reducer Semantics

The implementation should use a reducer-like mapping layer for live events.

Required semantics:

- `meta`: set phase to `preparing`; no activity row required
- `agent_handoff`: update phase and append or coalesce an `agent_handoff` activity row
- `plan`: replace any previous active planning summary and append a single planning activity
- `replan`: supersede the latest planning activity instead of appending an unbounded list
- `tool_call`: append or update an active tool activity keyed by `call_id`
- `tool_approval`: update the matching tool activity to approval-needed or approval-resolved state
- `tool_result`: complete the matching tool activity keyed by `call_id`
- `delta`: append visible content to markdown body and retain previously collected runtime state
- `done`: finalize footer status and freeze activity list
- recoverable `error`: append transient hint state without wiping markdown
- terminal `error`: finalize footer status as interrupted or failed without wiping markdown

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

### Rendering Boundary

Structured runtime UI must be rendered by React outside the markdown AST, not by mutating markdown syntax during streaming.

Rationale:

- Current markdown streaming animation already relies on a single text body
- Runtime fragments need stable keys and stateful updates
- Mixing AST transforms into partial streaming text would increase remount and scroll instability risk

The supported composition is:

- React-rendered phase and activity sections above markdown
- Optional React-rendered embedded summary section between activity and markdown, or immediately after the markdown intro
- Markdown body rendered by `XMarkdown`
- React-rendered footer status below markdown

This keeps one visual composition while preserving predictable streaming behavior.

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
- Preserve `runtime` when creating `Bubble.List` items so assistant renderers can access the full message object

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
- If persisted `message.runtime` exists, do not discard it during `defaultMessages()` transformation
- If `turns` or `blocks` are used, their synthesis path must be deterministic and tested against the same render contract as live runtime

## Data Shape Guidance

The assistant runtime type should be intentionally narrow. Recommended categories:

- `phase`
- `activities`
- `summary`
- `status`

Avoid exposing raw backend transport payloads directly to the renderer. Convert them into stable, user-facing primitives first.

Explicit non-goals:

- No raw tool argument dump in UI
- No free-form parsing of arbitrary markdown tables into custom cards
- No one-off domain renderer unless the source payload shape is stable

## Error Handling

- Recoverable delays such as slow tool execution should appear as lightweight status hints, not terminal failures
- Hard failures should preserve any already-rendered reply content and append a restrained terminal error state
- Timeout handling must not erase previously streamed content
- Soft timeout is rendered as a temporary hint in the activity or footer layer and is cleared on the next successful event
- Existing partial markdown must remain mounted when timeout or terminal error state changes

## Testing Strategy

Add focused tests for:

- Event mapping from SSE handlers into assistant runtime state
- Rendering with runtime-enhanced assistant messages
- Fallback rendering for plain markdown-only assistant messages
- Error and timeout cases preserving partial content

Regression coverage should include existing assistant markdown streaming behavior.

Required high-risk coverage:

- Reducer tests for `replan`, `tool_call`, `tool_approval`, `tool_result`, and `done`
- Duplicate `tool_call` coalescing by `call_id`
- First `delta` replacing placeholder-only content
- Live-stream runtime plus markdown rendering without duplicate content
- Replay hydration parity between stored `runtime` and synthesized runtime
- History loading preserving existing `message.runtime` when available
- Footer status transitions for `completed`, `soft-timeout`, and `error`

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
