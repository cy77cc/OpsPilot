# AI Assistant Current Turn Experience Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve the active AI assistant turn so the drawer follows the current answer correctly and inline tool calls render without block-style line breaks.

**Architecture:** Keep the change inside the existing AI drawer pipeline. `CopilotSurface` owns the follow/detach scroll state machine, `AssistantReply` owns active-step content flow, and `ToolReference` owns the inline tool token visuals. Current-turn behavior is the delivery target, but the interfaces should remain reusable for later history alignment.

**Tech Stack:** React, TypeScript, Ant Design, `@ant-design/x`, `antd-style`, Vitest, Testing Library

---

## File Map

- Modify: `web/src/components/AI/CopilotSurface.tsx`
  - Add current-turn scroll state machine, anchor wiring, and follow recovery behavior.
- Modify: `web/src/components/AI/AssistantReply.tsx`
  - Tighten active-step segment rendering so tool references remain in the same content flow.
- Modify: `web/src/components/AI/ToolReference.tsx`
  - Replace the current icon/spinner treatment with a compact inline code-style token and gradient loading motion.
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
  - Add follow/detach tests for the drawer scroll behavior.
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
  - Add coverage for inline tool placement and no-block-break behavior.

## Chunk 1: Scroll Follow State Machine

### Task 1: Add failing scroll-follow tests in `CopilotSurface`

**Files:**
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Modify: `web/src/components/AI/CopilotSurface.tsx`

- [ ] **Step 1: Add a test for turn-start top anchoring**

Add a test case that renders one user message and one updating assistant message, stubs the scroll container dimensions, triggers the effect path, and asserts that the container scroll call aligns to the active user turn rather than blindly using bottom-only behavior.

Suggested assertions:

```tsx
expect(scrollToMock).toHaveBeenCalledWith(
  expect.objectContaining({ behavior: 'auto' }),
);
```

- [ ] **Step 2: Add a test for detach on user upward scroll**

Add a test that:

- mounts the drawer with active messages
- simulates a user-originated scroll away from the bottom zone
- verifies the "快速回到底部" button appears
- verifies later runtime updates do not auto-scroll while detached

- [ ] **Step 3: Add a test for automatic recovery when returning to bottom**

Add a test that:

- starts detached
- updates the mocked container `scrollTop` so the distance to bottom is `<= 24`
- fires a scroll event
- verifies follow mode is restored and later updates scroll again

- [ ] **Step 4: Run the targeted test file to confirm failures**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx
```

Expected:

- FAIL on the newly added assertions because follow/detach state is not implemented yet

- [ ] **Step 5: Commit the failing test scaffold**

```bash
git add web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "test(ai): cover copilot follow mode behavior"
```

### Task 2: Implement follow/detach behavior in `CopilotSurface`

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`

- [ ] **Step 1: Add explicit scroll state refs**

Introduce focused refs/state in `CopilotSurface`:

```ts
const followStateRef = React.useRef<'following' | 'detached'>('following');
const programmaticScrollRef = React.useRef(false);
const activeTurnRef = React.useRef<{ userId?: string; assistantId?: string }>({});
```

Keep these local to the surface. Do not move message rendering logic into the scroll code.

- [ ] **Step 2: Derive current-turn anchors from the rendered message list**

Track the newest user message and newest assistant message IDs from `messages`. Pass DOM markers into the rendered `Bubble.List` output using wrapper elements or stable `data-*` attributes that can be queried from `contentRef.current`.

Example target shape:

```tsx
<div data-message-id={item.id} data-message-role={item.message.role}>
  ...
</div>
```

- [ ] **Step 3: Implement turn-start top alignment**

When a new user message is committed:

- set follow mode to `following`
- schedule a programmatic scroll that aligns the active user message top with the scroll container top

Keep this separate from the streaming-follow effect.

- [ ] **Step 4: Implement assistant follow updates**

When the current assistant message changes and follow mode is active:

- compute the active assistant element
- scroll so its lower edge remains in view
- keep `behavior: 'auto'` for frequent streaming updates

Avoid calling raw `scrollTo({ top: el.scrollHeight })` on every update.

- [ ] **Step 5: Implement detach and auto-recovery thresholds**

Inside the container scroll listener:

- ignore scrolls during `programmaticScrollRef`
- when user movement takes the viewport beyond the follow zone, set `detached`
- when the distance to bottom is `<= 24`, restore `following`
- continue to drive the "快速回到底部" button from the existing `> 120px` threshold

- [ ] **Step 6: Wire the button to restore follow mode**

Update `handleScrollToBottom` so it:

- sets follow mode back to `following`
- performs a smooth scroll

- [ ] **Step 7: Run the targeted test file**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx
```

Expected:

- PASS for the new and existing `CopilotSurface` tests

- [ ] **Step 8: Commit the scroll behavior**

```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "feat(ai): add current turn follow mode"
```

## Chunk 2: Inline Tool Reference Rendering

### Task 3: Add failing inline tool rendering tests in `AssistantReply`

**Files:**
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/ToolReference.tsx`

- [ ] **Step 1: Add a test for inline tool token placement after markdown text**

Add a case where the active step contains:

```ts
segments: [
  { type: 'text', text: 'Checking cluster state ' },
  { type: 'tool_ref', callId: 'call-1' },
]
```

Assert that:

- the tool label is rendered
- it is not duplicated in the activity list
- the surrounding step still renders as one active-step body

- [ ] **Step 2: Add a fallback test for tool-only steps**

Add a case where the active step has no text, only a `tool_ref`. Assert that the tool token still appears under the active step header and no markdown body is required.

- [ ] **Step 3: Add a test that loading and result states reuse the same tool node**

Render one loading tool and rerender it as done/error using the same `call_id`. Assert that only one visible tool label instance remains present.

- [ ] **Step 4: Run the targeted test file to verify failures**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx
```

Expected:

- FAIL if the current rendering still creates layout or duplication mismatches against the new assertions

- [ ] **Step 5: Commit the failing test scaffold**

```bash
git add web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "test(ai): cover inline tool reference rendering"
```

### Task 4: Tighten `AssistantReply` content flow

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Refactor `StepContentRenderer` to preserve one content flow**

Keep the existing segment-based approach, but ensure the renderer differentiates between:

- inline tool references after text
- tool-only fallback rows when no text has appeared

Use a shape like:

```tsx
<span className={styles.inlineToolFlow}>
  <XMarkdown ... />
  <ToolReference ... />
</span>
```

only where it preserves inline behavior. Do not move tool rendering back into the activity list.

- [ ] **Step 2: Add focused styles for inline flow and fallback rows**

Add style slots such as:

```ts
inlineToolFlow
inlineToolFallback
```

with these constraints:

- no top/bottom margins
- inline-level layout for the normal path
- `white-space` behavior that allows natural wrapping but avoids forced paragraph breaks

- [ ] **Step 3: Keep non-tool activities out of the content flow**

Preserve the current split:

- `tool_call/tool_result` belong in the step body content flow
- other activity kinds remain separate low-weight activity rows

- [ ] **Step 4: Run the targeted reply tests**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx
```

Expected:

- PASS for all existing and newly added `AssistantReply` tests

- [ ] **Step 5: Commit the active-step rendering changes**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai): keep tool references inline in active steps"
```

## Chunk 3: ToolReference Visual States

### Task 5: Add visual-state coverage for `ToolReference`

**Files:**
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Modify: `web/src/components/AI/ToolReference.tsx`

- [ ] **Step 1: Extend reply-level tests to assert loading vs done labels**

Add assertions that verify:

- loading tool state renders immediately from `tool_call`
- done and error states remain clickable
- loading state is not clickable

If direct style assertions are too brittle, assert DOM semantics such as `role="button"` only appearing for done/error states.

- [ ] **Step 2: Run the reply tests to confirm the new assertions fail**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx
```

Expected:

- FAIL until the `ToolReference` state behavior is updated

- [ ] **Step 3: Commit the failing assertions**

```bash
git add web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "test(ai): cover tool reference visual states"
```

### Task 6: Implement compact code-style tokens and loading motion

**Files:**
- Modify: `web/src/components/AI/ToolReference.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Replace the current badge-like styling with compact inline token styling**

Revise `ToolReference` styles so the root behaves like an inline code token:

- `display: inline-flex`
- `vertical-align: baseline`
- no block margins
- lighter padding
- narrower radius
- monospace font preserved

- [ ] **Step 2: Replace the spinner with left-to-right text gradient motion**

Implement a loading class that animates text foreground rather than rotating an icon. Keep the text readable and the background stable.

Suggested direction:

```ts
backgroundImage: 'linear-gradient(...)';
backgroundSize: '200% 100%';
WebkitBackgroundClip: 'text';
animation: 'toolRefSweep 1.2s linear infinite';
```

Remove the rotating spinner treatment.

- [ ] **Step 3: Keep clickability limited to done/error states**

Preserve the existing result card behavior, but ensure:

- loading token has no `role="button"`
- done/error tokens keep `role="button"` and keyboard support

- [ ] **Step 4: Run the targeted reply tests**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx
```

Expected:

- PASS for all tool visual-state coverage

- [ ] **Step 5: Commit the tool token visual polish**

```bash
git add web/src/components/AI/ToolReference.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai): refine inline tool reference states"
```

## Chunk 4: Final Verification

### Task 7: Run focused verification for the complete change

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/ToolReference.tsx`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Run both focused test files together**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
```

Expected:

- PASS for both files

- [ ] **Step 2: Inspect the final diff**

Run:

```bash
git diff --stat HEAD~4..HEAD
```

Expected:

- only the planned AI surface files and tests changed

- [ ] **Step 3: Create the final integration commit**

```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/AssistantReply.tsx web/src/components/AI/ToolReference.tsx web/src/components/AI/__tests__/CopilotSurface.test.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai): improve current turn experience"
```
