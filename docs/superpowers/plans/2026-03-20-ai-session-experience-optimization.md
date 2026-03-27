# AI Session Experience Optimization Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rework the AI copilot drawer so session data loads only when the drawer opens, multi-session switching is reliable, historical assistant steps are lazy and collapsible, and scroll follow behavior matches the approved UX.

**Architecture:** Keep the backend contract unchanged and refactor the frontend into three explicit layers: session list state, session detail state, and history runtime state. `CopilotSurface` owns session loading and the follow/detach scroll state machine, `AssistantReply` owns history-detail disclosure and message-local error isolation, and `historyProjection` remains the projection/content adapter for historical runs.

**Tech Stack:** React 18, TypeScript, Ant Design, `@ant-design/x`, `antd-style`, Vitest, Testing Library

---

## File Map

- Modify: `web/src/components/AI/CopilotSurface.tsx`
  - Remove `defaultMessages`-driven history loading, add explicit session list/detail state, add request cancellation and request-id guards, and split `initial-scroll` from `stream-follow`.
- Modify: `web/src/components/AI/AssistantReply.tsx`
  - Add history-only runtime disclosure, step-level collapse state, and a local fallback for malformed history details.
- Modify: `web/src/components/AI/historyProjection.ts`
  - Keep projection parsing but expose helpers suitable for message-local lazy loading and cache cleanup.
- Modify: `web/src/components/AI/types.ts`
  - Add any message/runtime typing needed for explicit history loading and disclosure state.
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
  - Cover open-time loading, multi-session switching, abort/ignore stale responses, cache cleanup, and scroll behavior.
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
  - Cover history runtime lazy load, step-title-first rendering, local fallback rendering, and no eager step content.
- Modify: `web/src/components/AI/historyProjection.test.ts`
  - Cover runtime cache cleanup hooks and any helper changes made for lazy history loading.
- Optional create: `web/src/components/AI/AssistantReplyHistoryBoundary.tsx`
  - If `AssistantReply.tsx` becomes too dense, isolate the history detail `ErrorBoundary` in a focused helper component.

## Implementation Order

Implement in this order only:

1. Lock failing tests for session loading and switching.
2. Refactor `CopilotSurface` session list/detail state and cancellation rules.
3. Lock failing tests for history runtime lazy disclosure and local failure isolation.
4. Implement history-only disclosure in `AssistantReply` and cache cleanup in `historyProjection`.
5. Lock failing tests for follow/detach scroll behavior after the state refactor.
6. Implement the scroll state machine and verify the combined drawer behavior.
7. Run targeted and broad verification, then update the plan status.

## Chunk 1: Session State Refactor

### Task 1: Add failing tests for drawer-open loading behavior

**Files:**
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Modify: `web/src/components/AI/CopilotSurface.tsx`

- [ ] **Step 1: Add a test that closed drawers do not fetch sessions**

Add a test that renders:

```tsx
<CopilotSurface open={false} onClose={() => undefined} />
```

and asserts:

```tsx
expect(aiApi.getSessions).not.toHaveBeenCalled();
expect(aiApi.getSession).not.toHaveBeenCalled();
```

- [ ] **Step 2: Add a test that opening the drawer loads only the latest session detail**

Mock:

```ts
vi.mocked(aiApi.getSessions).mockResolvedValue({
  data: [
    { id: 'sess-2', title: 'Latest', scene: 'cluster', updated_at: '2026-03-20T10:00:00Z' },
    { id: 'sess-1', title: 'Older', scene: 'cluster', updated_at: '2026-03-20T09:00:00Z' },
  ],
} as any);
```

Assert that opening the drawer calls:

```tsx
expect(aiApi.getSessions).toHaveBeenCalledTimes(1);
expect(aiApi.getSession).toHaveBeenCalledWith('sess-2');
expect(aiApi.getSession).toHaveBeenCalledTimes(1);
```

- [ ] **Step 3: Add a test that switching sessions does not leave later sessions stuck in loading**

Model one active session as ready and switch to another. Assert that:

- the second session triggers its own detail fetch
- the second session content renders once resolved
- loading indicators are scoped to the selected session only

- [ ] **Step 4: Add a test for stale-response protection**

Simulate two rapid switches:

1. click `sess-1`
2. before it resolves, click `sess-2`
3. resolve `sess-1` last

Assert that the UI still shows `sess-2` messages and the late `sess-1` result is ignored.

- [ ] **Step 5: Run the targeted test file to confirm failure**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx
```

Expected:

- FAIL on the new loading/switching assertions because the drawer still uses implicit `defaultMessages` loading

- [ ] **Step 6: Commit the failing test scaffold**

```bash
git add web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "test(ai): cover session drawer loading boundaries"
```

### Task 2: Implement explicit session list/detail state in `CopilotSurface`

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Modify: `web/src/components/AI/types.ts`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`

- [ ] **Step 1: Replace `defaultMessages` history loading with explicit session state**

Introduce local state with a shape similar to:

```ts
type SessionDetailState = {
  status: 'idle' | 'loading' | 'ready' | 'error';
  messages: XChatMessage[];
  error?: string;
  loadedAt?: number;
  requestId?: string;
  abortController?: AbortController;
};
```

Use one state bucket for:

- session list
- active session key
- detail-by-session cache

Do not leave `useXChat` responsible for historical `getSession()` reads.

- [ ] **Step 2: Load sessions only when `open === true`**

Refactor the scene-loading effect so it:

- returns immediately when the drawer is closed
- fetches `getSessions(scene)` once per open/scene change
- sorts or trusts sorted results and picks the latest session
- falls back to `NEW_SESSION_KEY` when the list is empty

- [ ] **Step 3: Add request cancellation and stale-result guards**

For each detail fetch:

- create a fresh `AbortController`
- create a fresh `requestId`
- store both on the target session detail entry
- abort the previous in-flight detail request when switching away
- abort any still-active detail request during effect cleanup when the drawer closes or the component unmounts
- only commit results if `requestId` still matches the latest request for that session

Keep the request context effect-local so React 18 Strict Mode cleanup cannot cancel a later request by accident.

- [ ] **Step 4: Hydrate session messages explicitly**

Convert `aiApi.getSession()` results into the message shape used by the surface. Preserve:

- user messages as plain text
- assistant messages with summary body visible immediately
- `run_id` metadata needed for later history-runtime disclosure

Do not eagerly call `getRunProjection()` while hydrating the session detail.

- [ ] **Step 5: Connect current-turn sending to the explicit session cache**

Ensure:

- sending on `NEW_SESSION_KEY` still creates a session first
- the newly created session becomes active
- the current-turn stream can append/update messages for the active session
- completed turns reconcile back into that session's detail cache

- [ ] **Step 6: Run the targeted test file**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx
```

Expected:

- PASS for the new session loading and switching tests

- [ ] **Step 7: Commit the state refactor**

```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/types.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "feat(ai): refactor copilot session loading state"
```

## Chunk 2: Historical Runtime Disclosure

### Task 3: Add failing tests for history-only lazy runtime disclosure

**Files:**
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/historyProjection.ts`

- [ ] **Step 1: Add a test that history messages show summary before steps**

Render:

```tsx
<AssistantReply
  content="Final answer"
  status="success"
  messageId="msg-1"
  hasRuntime
  onLoadRuntime={mockLoadRuntime}
/>
```

Assert that:

- `Final answer` is visible immediately
- step body content is not visible yet
- a disclosure affordance such as `查看执行过程` is present

- [ ] **Step 2: Add a test that runtime loads only after expanding history detail**

Click the disclosure button and assert:

```tsx
expect(mockLoadRuntime).toHaveBeenCalledWith('msg-1');
```

Then resolve runtime with multiple steps and assert:

- only step titles render initially
- step body content is still hidden

- [ ] **Step 3: Add a test that clicking a step reveals only that step content**

Use runtime:

```ts
plan: {
  steps: [
    { id: 's1', title: '检查节点', status: 'done', content: '节点详情' },
    { id: 's2', title: '汇总结果', status: 'done', content: '汇总详情' },
  ],
}
```

Assert that:

- `节点详情` appears only after clicking `检查节点`
- `汇总详情` remains hidden until its own title is clicked

- [ ] **Step 4: Add a test for malformed history runtime fallback**

Make `onLoadRuntime` reject or throw during render and assert:

- the summary body remains visible
- a local fallback such as `该详情无法渲染` or retry affordance appears
- no global drawer crash is required for the test to pass

- [ ] **Step 5: Run the targeted test file to confirm failure**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx
```

Expected:

- FAIL because history replies currently render runtime immediately or expose the older disclosure behavior

- [ ] **Step 6: Commit the failing history tests**

```bash
git add web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "test(ai): cover lazy history runtime disclosure"
```

### Task 4: Implement history runtime lazy disclosure and cache cleanup

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/historyProjection.ts`
- Optional create: `web/src/components/AI/AssistantReplyHistoryBoundary.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/historyProjection.test.ts`

- [ ] **Step 1: Separate current-turn and history rendering paths**

In `AssistantReply`, keep the existing direct-runtime path for current-turn messages. Only enable the new two-level disclosure when:

- `status` is not streaming/updating
- `messageId`, `hasRuntime`, and `onLoadRuntime` are present

- [ ] **Step 2: Implement history disclosure state**

Track:

```ts
const [historyExpanded, setHistoryExpanded] = useState(false);
const [expandedSteps, setExpandedSteps] = useState<Record<string, boolean>>({});
```

Behavior:

- first click expands the execution-process section and triggers `onLoadRuntime`
- once runtime is ready, render step titles only
- clicking a step toggles only that step body

- [ ] **Step 3: Add message-local error isolation**

Wrap the history-detail section in a local error boundary. If projection-derived rendering throws:

- keep the summary body visible
- show a local fallback
- do not let the exception escape to the full `CopilotSurface`

- [ ] **Step 4: Expose history cache cleanup helpers**

Extend `historyProjection.ts` with focused helpers, for example:

```ts
export function clearHistoryRuntimeCacheForSession(runIds: string[]): void {}
export function clearHistoryRuntimeCacheExcept(keepRunIds: string[]): void {}
```

Keep the implementation minimal. The lowest requirement is to clear non-active-session runtime cache when the drawer closes.

- [ ] **Step 5: Add/adjust tests for cache cleanup**

Update `web/src/components/AI/historyProjection.test.ts` so a cleared run requires a fresh `getRunProjection()` call on the next access.

- [ ] **Step 6: Run the targeted tests**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx
npm run test:run -- web/src/components/AI/historyProjection.test.ts
```

Expected:

- PASS for the new lazy history disclosure and cache cleanup cases

- [ ] **Step 7: Commit the history runtime work**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/historyProjection.ts web/src/components/AI/__tests__/AssistantReply.test.tsx web/src/components/AI/historyProjection.test.ts web/src/components/AI/AssistantReplyHistoryBoundary.tsx
git commit -m "feat(ai): lazy load historical runtime details"
```

If no helper file was created, omit it from `git add`.

## Chunk 3: Scroll Follow State Machine

### Task 5: Add failing tests for post-refactor scroll behavior

**Files:**
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Modify: `web/src/components/AI/CopilotSurface.tsx`

- [ ] **Step 1: Add a test that opening a ready session performs one `initial-scroll`**

Assert that after the active session messages render, the container scrolls to the bottom once with:

```tsx
expect(scrollToMock).toHaveBeenCalledWith(
  expect.objectContaining({ top: expect.any(Number), behavior: 'auto' }),
);
```

- [ ] **Step 2: Add a test that user upward scroll enters detached mode**

Simulate container metrics so distance-to-bottom exceeds `120`, fire a scroll event, and assert:

- the `快速回到底部` button is visible
- later assistant updates do not trigger another auto-scroll

- [ ] **Step 3: Add a test that history-step expansion does not trigger follow**

With the surface detached, expand a historical step and assert that:

```tsx
expect(scrollToMock).not.toHaveBeenCalledWith(
  expect.objectContaining({ behavior: 'auto' }),
);
```

for that expansion path.

- [ ] **Step 4: Add a test that interrupted streams preserve follow state**

Render an updating assistant message, then rerender it as errored/retry-ready. Assert:

- no unexpected scroll reset occurs
- if the container was detached before the interruption, it remains detached after rerender

- [ ] **Step 5: Run the targeted test file to confirm failure**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx
```

Expected:

- FAIL on the new follow/detach expectations after the state refactor

- [ ] **Step 6: Commit the failing scroll tests**

```bash
git add web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "test(ai): cover session follow state machine"
```

### Task 6: Implement `initial-scroll` and `stream-follow`

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`

- [ ] **Step 1: Preserve an explicit follow/detach ref**

Use focused refs such as:

```ts
const followStateRef = React.useRef<'following' | 'detached'>('following');
const programmaticScrollRef = React.useRef(false);
```

Do not let history disclosure state mutate this ref directly.

- [ ] **Step 2: Implement `initial-scroll` for open/switch transitions**

When:

- the drawer opens with a ready active session, or
- the active session changes and its detail becomes ready

schedule one bottom alignment pass after the DOM is committed.

Use `requestAnimationFrame` or `useLayoutEffect` so measurements reflect the actual rendered height.

- [ ] **Step 3: Implement `stream-follow` only for the live tail message**

Only auto-follow when all of the following are true:

- the drawer is open
- follow state is `following`
- the active session is unchanged
- the last assistant message for the active session gained streaming content

Ignore:

- history projection disclosure
- step expansion
- stale session updates

For high-frequency streaming updates:

- prefer native scroll anchoring where possible
- if a manual scroll correction is still needed, schedule it with `requestAnimationFrame`
- avoid issuing a programmatic scroll on every token-sized content delta if it would cause visible scroll thrashing

- [ ] **Step 4: Keep detach/recovery thresholds explicit**

Inside the scroll listener:

- ignore scroll events during programmatic movement
- set `detached` when user movement leaves the bottom zone
- restore `following` when distance-to-bottom is `<= 24`
- keep the scroll button visible when distance-to-bottom is `> 120`

- [ ] **Step 5: Preserve follow state across interrupted streams**

When a streaming assistant message transitions to error/retry UI:

- do not reset follow state
- do not perform automatic bottom recovery
- allow a later retry to keep following only if the user never detached

- [ ] **Step 6: Run the targeted test file**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx
```

Expected:

- PASS for the new scroll tests and the earlier loading/switching coverage

- [ ] **Step 7: Commit the scroll behavior**

```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "feat(ai): stabilize copilot follow behavior"
```

## Chunk 4: Verification And Integration

### Task 7: Run focused verification for the AI drawer stack

**Files:**
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/historyProjection.test.ts`

- [ ] **Step 1: Run the focused AI component tests**

Run:

```bash
npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx
npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx
npm run test:run -- web/src/components/AI/historyProjection.test.ts
```

Expected:

- PASS for all targeted AI drawer tests

- [ ] **Step 2: Run the broader AI/frontend regression set**

Run:

```bash
npm run test:run -- web/src/api/modules/ai.test.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/replyRuntime.test.ts
```

Expected:

- PASS with no regressions from the session/drawer refactor

- [ ] **Step 3: Review the final diff for scope discipline**

Check:

```bash
git diff --stat HEAD~3..HEAD
```

Confirm the work stayed inside:

- `CopilotSurface`
- `AssistantReply`
- `historyProjection`
- related AI drawer tests

- [ ] **Step 4: Commit any final polish**

```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/AssistantReply.tsx web/src/components/AI/historyProjection.ts web/src/components/AI/types.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx web/src/components/AI/historyProjection.test.ts
git commit -m "test(ai): verify session experience optimization"
```

Only create this commit if verification required final non-functional fixes.

- [ ] **Step 5: Update the spec and plan status if needed**

If implementation deviates from the approved design, update:

- `docs/superpowers/specs/2026-03-20-ai-session-experience-optimization-design.md`
- this plan file

before handing off execution results.

## Notes For Execution

- Follow TDD strictly. Do not implement ahead of the failing tests.
- Keep current-turn behavior working while moving historical loading out of `defaultMessages`.
- Favor small helper functions over large new abstractions inside `CopilotSurface.tsx`.
- If `AssistantReply.tsx` becomes difficult to reason about, split only the history disclosure/error-boundary portion into a focused helper component.
- Do not add backend APIs in this plan. The approved scope is frontend-only.
- Test commands in this plan assume the existing `web/package.json` script `test:run = vitest run`; run them from the repo root as written.
