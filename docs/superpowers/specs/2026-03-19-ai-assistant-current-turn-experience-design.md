# AI Assistant Current Turn Experience Design

Date: 2026-03-19

## Goal

Optimize the current-turn AI assistant experience in the copilot drawer.

This design only covers the actively streaming assistant turn. It does not solve historical conversation replay consistency in this change. The design should still preserve component and event boundaries so the same rendering model can later be extended to history.

## Scope

In scope:

- Current-turn auto-follow scrolling behavior
- User-controlled scroll detachment and automatic recovery
- Current-turn `tool_call` inline rendering behavior
- Current-turn `tool_call` loading animation and non-breaking layout rules
- Frontend and backend event contract requirements needed for the above behavior

Out of scope:

- Historical conversation replay consistency
- Session persistence redesign
- New backend event categories
- Global drawer or shell redesign

## Product Direction

The selected direction is a unified rendering target with current-turn-first delivery.

This means:

- The UI behavior for the active assistant turn should be designed as the canonical model.
- The implementation in this change may land only on the current turn.
- The design must avoid introducing new assumptions that make history replay harder later.

## Problems

### 1. Auto-follow is too weak

The current drawer mostly behaves like a generic "scroll to bottom" chat container. That does not match the intended interaction:

- After the user sends a question, that question should immediately move to the top of the viewport.
- The viewport should then follow the assistant output as it grows.
- If the user manually scrolls upward, follow mode must stop immediately.
- If the user later returns to the bottom area, follow mode should recover automatically.

### 2. Current-turn tool rendering breaks reading flow

`tool_call` output currently appears with unwanted line breaks. The tool name does not feel embedded in the assistant answer and instead reads like a detached activity row or block boundary.

### 3. Loading feedback is too mechanical

The current executing state is not aligned with the intended visual language. The desired behavior is:

- show the tool name as soon as `tool_call` arrives
- keep it inline with the active step body
- avoid leading and trailing line breaks
- render a left-to-right text color gradient loading effect while the tool is waiting for its result

## Approach

### 1. Scroll ownership as a small state machine

Scrolling should be modeled as explicit ownership, not a loose set of bottom checks.

Two states are sufficient:

- `following`: the system owns scroll position for the active turn
- `detached`: the user owns scroll position

The state machine is driven by current-turn anchors rather than generic container-bottom behavior.

### 2. Tool references as inline content segments

`tool_call` should be rendered as part of the active step content flow, not as an external activity block.

The rendering target is an inline code-like marker appended in the same reading stream as the active assistant text.

### 3. Current-turn-first implementation boundary

This change should land in the current-turn path only, but with interfaces that remain reusable for later history replay.

That means:

- `CopilotSurface` owns scroll behavior only
- `AssistantReply` owns active-turn content composition
- `ToolReference` owns a single inline tool token

## Design

### Section 1: Scroll State Machine

The drawer should explicitly model scroll-follow behavior with two states.

#### States

- `following`
  - The system controls the viewport for the active turn.
  - New user input and assistant streaming updates can move the viewport.

- `detached`
  - The user controls the viewport.
  - New assistant updates do not force the viewport to move.

#### Events

`user_message_committed`

- Trigger: the new user message has been inserted into the message list
- Effect:
  - set state to `following`
  - scroll the viewport so the new user message top aligns with the container top

`assistant_stream_updated`

- Trigger: the active assistant message gains visible content or visible runtime updates
- Condition: only when the state is `following`
- Effect:
  - keep the active assistant message bottom within the follow zone
  - do not use a blind `scrollToBottom` strategy

`user_scrolled_up`

- Trigger: a user-originated scroll moves the viewport out of the bottom follow zone
- Effect:
  - switch state to `detached`

`user_returned_to_bottom`

- Trigger: the user scrolls back into the bottom threshold zone or clicks the "scroll to bottom" button
- Effect:
  - switch state back to `following`

#### Thresholds

- Recover `following` when the distance to bottom is less than or equal to `24px`
- Show the "scroll to bottom" affordance when the distance to bottom is greater than `120px`

#### Anchor model

The implementation should maintain stable DOM anchors for:

- the current user message
- the current assistant message

Behavior by phase:

- turn start: align the user message anchor to the top of the viewport
- streaming: keep the assistant message anchor in the follow zone

This avoids conflating "start of turn positioning" with "streaming follow behavior".

### Section 2: Tool Reference Rendering

`tool_call` should be treated as inline content in the active step, not as a separate block or global activity row.

#### Placement rules

- Render the tool reference as soon as `tool_call` arrives and `tool_name` is known
- Insert it into the active step content flow
- Prefer inline placement immediately after the active step text
- Do not inject explicit line breaks before or after the tool reference
- If the active step has no text yet, fall back to a single standalone line directly under the step header

#### Update rules

- `tool_call + active` renders the loading state
- `tool_result + done` updates the same visual node in place
- `tool_result + error` updates the same visual node in place
- The UI must not create a second separate visible node for the result state

#### Interaction rules

- Loading state is not clickable
- Done or error states may open the existing result detail UI

### Section 3: Non-breaking Inline Layout

The "no extra line breaks" requirement is a rendering strategy constraint, not just a CSS preference.

#### Layout requirements

- The tool reference root element must be inline-level
- It must not have top or bottom margins
- It must keep `white-space: nowrap`
- It should align with the text baseline
- It may use a small inline start gap from the preceding text

#### Content composition rules

- The active step renderer must keep text segments and `tool_ref` segments in one content flow
- A block-level markdown container must not force the tool token into a new paragraph
- Natural wrapping is acceptable when the current line runs out of width
- Forced block separation is not acceptable

#### Acceptance criteria

- If the final line of assistant text has enough space, the tool name appears on that same visual line
- If the line does not have enough width, wrapping may occur naturally
- The tool token must not jump between loading and done/error states

### Section 4: Loading Motion

The loading state should use a left-to-right foreground gradient motion rather than a spinner.

#### Motion rules

- Apply the animation only while the tool is in the active waiting state
- Animate text foreground color or text mask from left to right
- Keep the background mostly stable so motion stays low-noise
- Remove the loading animation immediately when the result state arrives

#### Visual direction

- Use an inline code-style token
- Keep the shape light and narrow
- The text should remain readable during animation
- The effect should suggest "running" rather than "attention alert"

### Section 5: Responsibilities

#### `CopilotSurface`

Owns:

- scroll state machine
- current-turn anchors
- bottom affordance visibility

Does not own:

- tool reference rendering
- step content assembly

#### `AssistantReply`

Owns:

- active-step content composition
- inline `tool_ref` placement
- fallback placement when the step body has no text

Does not own:

- drawer scroll behavior

#### `ToolReference`

Owns:

- a single inline tool token
- loading / done / error visual states
- result-detail trigger in non-loading states

Does not own:

- active step layout decisions

## Minimal Contract Requirements

This design only needs a minimal current-turn contract.

Required from the event stream:

- `tool_call.call_id`
- `tool_call.tool_name`
- active-step association for the current rendering context
- `tool_result.call_id`
- `tool_result.status`

Contract rules:

- `tool_call` and `tool_result` must refer to the same visual token through the same `call_id`
- the frontend must be able to render the loading state from `tool_call` alone
- the frontend must update the existing inline token in place when `tool_result` arrives

## Testing Strategy

### Component tests

- active-turn tool reference stays inline with step content when text exists
- active-turn tool reference falls back to a single line below the step header when text is empty
- loading state transitions to done or error without creating duplicate nodes

### Scroll interaction tests

- sending a user message moves that message to the top of the viewport
- assistant streaming updates keep the current assistant answer in view while `following`
- manual upward scrolling switches to `detached`
- returning within the bottom threshold restores `following`

### Visual regression checks

- tool loading animation remains readable on light backgrounds
- no extra top or bottom spacing is introduced around inline tool tokens

## Risks

### Markdown composition edge cases

Block-level markdown output may still create unexpected spacing if the renderer boundary is not handled carefully.

Mitigation:

- keep inline tool placement attached to the active step content flow
- verify paragraph-ending and code-block-ending edge cases

### Scroll intent misclassification

Programmatic scrolling may be mistaken for user scrolling.

Mitigation:

- explicitly track programmatic scroll windows
- only detach on user-originated upward movement

## Deferred Work

The following work is intentionally excluded from this design:

- historical conversation replay consistency
- session persistence redesign
- unified current-turn and history hydration

These remain compatible follow-up areas, but they are not part of this design change.
