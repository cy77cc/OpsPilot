# AI Session Interaction Repair Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Repair AI assistant session interactions so active-session steps re-expand correctly, historical sessions always show lazy-loadable steps, and sending a new message from a detached scroll position forces bottom-follow recovery.

**Architecture:** Keep the existing AI surface structure and repair behavior in place rather than refactoring the UI shell. `historyProjection.ts` becomes the source of truth for stable historical step lookup metadata, `AssistantReply.tsx` owns message-local disclosure/cache state, and `CopilotSurface.tsx` owns the send-reset plus stream-follow scroll transitions.

**Tech Stack:** React, TypeScript, Ant Design, Vitest, Testing Library

---

## File Map

- Modify: `web/src/components/AI/historyProjection.ts`
  - Add stable historical step lookup metadata, backward-compatible fallback mapping, and helper behavior for lazy step resolution.
- Modify: `web/src/components/AI/types.ts`
  - Extend step/runtime typing for stable lookup metadata without changing visual concerns.
- Modify: `web/src/components/AI/AssistantReply.tsx`
  - Repair controlled collapse state, preserve message-local lazy content cache, and keep historical step failures local.
- Modify: `web/src/components/AI/CopilotSurface.tsx`
  - Add send-time follow reset, post-commit bottom alignment scheduling, and maintain stream-follow behavior.
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
  - Lock active-step re-expand behavior, historical step visibility, stable lazy lookup, and local failure/retry behavior.
- Modify: `web/src/components/AI/historyProjection.test.ts`
  - Lock projection adapter mapping and backward-compatible fallback behavior.
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
  - Lock follow-state transitions, send-time recovery scheduling, and mocked scroll invocation contracts.
- Optional Modify or Create: browser E2E test under the repo's existing E2E location if one already covers AI surface interactions
  - Verify real scroll recovery in a browser if the project already has a compatible harness.

## Chunk 1: Historical Step Mapping

### Task 1: Add failing projection mapping tests

**Files:**
- Modify: `web/src/components/AI/historyProjection.test.ts`
- Modify: `web/src/components/AI/historyProjection.ts`

- [ ] **Step 1: Add a test that historical steps keep visible titles even when step content cannot yet be resolved**

```ts
it('keeps historical step titles visible before lazy content loads', async () => {
  const hydrated = await hydrateAssistantHistoryFromProjection(messageWithPlanAndExecutorBlocks);
  expect(hydrated.runtime?.plan?.steps.map((step) => step.title)).toEqual([
    '收集上下文',
    '执行检查',
  ]);
});
```

- [ ] **Step 2: Add a test that historical lazy step loading uses stable mapping instead of rendered position**

```ts
it('resolves historical step content through stable mapping metadata', async () => {
  const hydrated = await hydrateAssistantHistoryFromProjection(messageWithReplanProjection);
  const runtime = hydrated.runtime!;
  expect(runtime.plan?.steps[1]).toMatchObject({
    title: '执行检查',
  });
  expect(runtime.plan?.steps[1].sourceBlockIndex).toBeDefined();
});
```

- [ ] **Step 3: Add a backward-compatibility test for projections that lack stable mapping metadata**

```ts
it('falls back safely for older historical projections without stable mapping metadata', async () => {
  const hydrated = await hydrateAssistantHistoryFromProjection(legacyProjectionMessage);
  expect(hydrated.runtime?.plan?.steps).toHaveLength(1);
  expect(hydrated.runtime?.plan?.steps[0].title).toBe('检查节点');
});
```

- [ ] **Step 4: Run the focused projection tests to confirm the new cases fail**

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts`
Expected: FAIL on new mapping assertions or missing metadata fields.

- [ ] **Step 5: Commit the failing tests**

```bash
git add web/src/components/AI/historyProjection.test.ts
git commit -m "test(ai): cover historical step mapping repair"
```

### Task 2: Implement stable historical mapping with backward-compatible fallback

**Files:**
- Modify: `web/src/components/AI/historyProjection.ts`
- Modify: `web/src/components/AI/types.ts`
- Test: `web/src/components/AI/historyProjection.test.ts`

- [ ] **Step 1: Extend the step typing for stable historical lookup metadata**

```ts
export interface AssistantReplyPlanStep {
  id: string;
  title: string;
  status: 'pending' | 'active' | 'done';
  content?: string;
  segments?: AssistantReplySegment[];
  loaded?: boolean;
  sourceBlockIndex?: number;
  sourceStepIndex?: number;
}
```

- [ ] **Step 2: Update `projectionToLazyRuntime()` to emit stable metadata and stable fallback identifiers**

```ts
steps.push({
  id: `historical-step-${planIndex}-${stepIndex}`,
  title,
  status: 'done',
  loaded: false,
  sourceBlockIndex: resolvedBlockIndex,
  sourceStepIndex: stepIndex,
});
```

- [ ] **Step 3: Keep backward compatibility by deriving the safest fallback mapping when stable block metadata cannot be inferred**

```ts
const fallbackBlockIndex = resolvedBlockIndex ?? legacyExecutorIndexMap.get(stepOrdinal) ?? undefined;
```

- [ ] **Step 4: Ensure unresolved historical steps still render titles and can surface local disclosure failure later**

```ts
if (fallbackBlockIndex === undefined) {
  return { ...step, unresolved: true };
}
```

- [ ] **Step 5: Run the focused projection tests to confirm they pass**

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts`
Expected: PASS for the new historical mapping and fallback assertions.

- [ ] **Step 6: Commit the mapping implementation**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/historyProjection.ts web/src/components/AI/historyProjection.test.ts
git commit -m "feat(ai): stabilize historical step mapping"
```

## Chunk 2: Assistant Reply Disclosure Repair

### Task 3: Add failing reply interaction tests

**Files:**
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Modify: `web/src/components/AI/AssistantReply.tsx`

- [ ] **Step 1: Add a test that completed active-session steps can collapse and re-expand**

```tsx
it('re-expands a completed active-session step after collapse', async () => {
  render(<AssistantReply {...activeReplyProps} />);
  await user.click(screen.getByText('获取服务器列表'));
  await user.click(screen.getByText('获取服务器列表'));
  expect(screen.getByText('已找到 5 台服务器')).toBeInTheDocument();
});
```

- [ ] **Step 2: Add a test that historical replies show step titles before step content loads**

```tsx
it('shows historical step titles before lazy content loads', () => {
  render(<AssistantReply {...historicalReplyProps} />);
  expect(screen.getByText('执行检查')).toBeInTheDocument();
});
```

- [ ] **Step 3: Add a test that expanding a historical step uses stable lookup metadata instead of rendered position**

```tsx
it('loads historical step content using stable mapping metadata', async () => {
  const onLoadStepContent = vi.fn().mockResolvedValue({
    content: '执行完成',
    segments: [{ type: 'text', text: '执行完成' }],
    activities: [],
  });
  render(<AssistantReply {...historicalReplyPropsWithStableMapping} onLoadStepContent={onLoadStepContent} />);
  await user.click(screen.getByText('执行检查'));
  expect(onLoadStepContent).toHaveBeenCalledWith('historical-step-1', 4);
});
```

- [ ] **Step 4: Add a test that a historical step load failure stays local and retry actually recovers**

```tsx
it('keeps historical step load failure local and recovers on retry', async () => {
  const onLoadStepContent = vi.fn()
    .mockRejectedValueOnce(new Error('boom'))
    .mockResolvedValueOnce({
      content: '恢复后的内容',
      segments: [{ type: 'text', text: '恢复后的内容' }],
      activities: [],
    });
  render(<AssistantReply {...historicalReplyProps} onLoadStepContent={onLoadStepContent} />);
  await user.click(screen.getByText('执行检查'));
  expect(screen.getByText('加载失败')).toBeInTheDocument();
  expect(screen.getByText('结论')).toBeInTheDocument();
  await user.click(screen.getByText('重试'));
  expect(onLoadStepContent).toHaveBeenCalledTimes(2);
  expect(screen.getByText('恢复后的内容')).toBeInTheDocument();
});
```

- [ ] **Step 5: Run the focused reply tests to confirm the new cases fail**

Run: `npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: FAIL on collapse state drift, missing historical-step rendering, or retry behavior.

- [ ] **Step 6: Commit the failing tests**

```bash
git add web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "test(ai): cover reply interaction repair"
```

### Task 4: Repair controlled collapse state and message-local lazy cache

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Rebuild the expanded-step map on every collapse change instead of only handling newly opened keys**

```ts
const nextExpandedMap = Object.fromEntries(nextKeys.map((key) => [String(key), true]));
setStepExpandStates(nextExpandedMap);
```

- [ ] **Step 2: Trigger lazy loading only for keys entering the expanded set for the first time**

```ts
const newlyExpanded = nextKeys.filter((key) => !prevExpandedMap[String(key)]);
```

- [ ] **Step 3: Keep lazy content and activities in `AssistantReply` instance-local cache so re-expansion is immediate**

```ts
const [stepContentCache, setStepContentCache] = useState<Record<string, LoadedStepContent | null>>({});
```

- [ ] **Step 4: Resolve historical step content through stable step metadata rather than completed-step render index**

```ts
const lookupIndex = step.sourceBlockIndex;
if (lookupIndex === undefined || step.unresolved) {
  setStepLoadStates((prev) => ({ ...prev, [step.id]: 'error' }));
  return;
}
await onLoadStepContent(step.id, lookupIndex);
```

- [ ] **Step 5: Guard the async lazy-loader against collapse/unmount races before writing cache**

```ts
const requestToken = Symbol(step.id);
inflightRequestsRef.current[step.id] = requestToken;
const result = await onLoadStepContent(step.id, lookupIndex);
if (inflightRequestsRef.current[step.id] !== requestToken) return;
```

- [ ] **Step 6: Preserve local failure and retry behavior without affecting sibling steps or summary content**

```ts
if (loadState === 'error') {
  return <RetryPanel onRetry={() => handleRetry(step.id, lookupIndex)} />;
}
```

- [ ] **Step 7: Run the focused reply tests to confirm they pass**

Run: `npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS for re-expand, historical-step visibility, and local retry assertions.

- [ ] **Step 8: Commit the reply repair**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "fix(ai): repair reply step interactions"
```

## Chunk 3: Send-Time Follow Recovery

### Task 5: Add failing scroll-follow tests

**Files:**
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Modify: `web/src/components/AI/CopilotSurface.tsx`

- [ ] **Step 1: Preserve or add explicit coverage that opening an existing conversation performs one initial bottom alignment**

```tsx
it('performs one initial bottom alignment when opening an existing conversation', async () => {
  render(<CopilotSurface open onClose={() => undefined} />);
  expect(scrollToMock).toHaveBeenCalledWith(expect.objectContaining({ behavior: 'auto' }));
});
```

- [ ] **Step 2: Add a test that sending a new message while detached forces one post-commit bottom alignment**

```tsx
it('forces bottom alignment when sending from detached mode', async () => {
  render(<CopilotSurface open onClose={() => undefined} />);
  simulateDetachedScroll(screen.getByTestId('copilot-scroll-container'));
  await user.type(screen.getByPlaceholderText('请输入问题或输入 / 查看命令'), '检查集群');
  await user.keyboard('{Enter}');
  expect(scrollToMock).toHaveBeenCalledWith(expect.objectContaining({ top: 1600 }));
});
```

- [ ] **Step 3: Add a test that follow mode remains active for streamed updates after send reset**

```tsx
it('keeps following streamed updates after send reset', async () => {
  // start detached, send message, then emit assistant updates
  expect(scrollToMock).toHaveBeenCalledTimes(2);
});
```

- [ ] **Step 4: Keep the existing detach test so send recovery does not break upward-scroll behavior**

```tsx
it('stays detached until user sends or manually returns to bottom', () => {
  expect(showScrollBottomButton()).toBe(true);
});
```

- [ ] **Step 5: Run the focused CopilotSurface tests to confirm the new cases fail**

Run: `npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: FAIL on missing send-reset scheduling or incorrect follow-state transitions.

- [ ] **Step 6: Commit the failing tests**

```bash
git add web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "test(ai): cover send-time follow recovery"
```

### Task 6: Implement send reset plus stream follow separation

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`

- [ ] **Step 1: Add an explicit send-reset trigger that restores `following` when a new user request is queued**

```ts
followStateRef.current = 'following';
pendingSendScrollRef.current = true;
```

- [ ] **Step 2: Execute the send reset only after the new user turn is committed**

```ts
useLayoutEffect(() => {
  if (!pendingSendScrollRef.current) return;
  pendingSendScrollRef.current = false;
  requestAnimationFrame(() => {
    scrollToBottom('auto');
  });
}, [messages.length]);
```

Implementation note:
- Preserve the `useLayoutEffect` + `requestAnimationFrame` combination.
- `useLayoutEffect` guarantees the effect runs after the DOM commit for the new turn.
- `requestAnimationFrame` yields one frame so the browser can finalize the painted layout before measuring `scrollHeight`.
- Do not "simplify" this to a same-tick synchronous scroll unless verification proves the message height is already final in this codepath.

- [ ] **Step 3: Preserve `ResizeObserver` as the stream-follow correction path for later asynchronous height changes**

```ts
const resizeObserver = new ResizeObserver(() => {
  if (followStateRef.current === 'following') {
    scrollToBottom('auto');
  }
});
```

- [ ] **Step 4: Keep programmatic scroll guarding so the send reset does not immediately re-detach itself**

```ts
withProgrammaticScroll(() => {
  el.scrollTo({ top: el.scrollHeight, behavior });
});
```

- [ ] **Step 5: Run the focused CopilotSurface tests to confirm they pass**

Run: `npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: PASS for detach preservation, send recovery, and follow-after-send assertions.

- [ ] **Step 6: Commit the scroll repair**

```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "fix(ai): restore send-time follow recovery"
```

## Chunk 4: Final Verification

### Task 7: Run combined verification and capture residual risk

**Files:**
- Test: `web/src/components/AI/historyProjection.test.ts`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`

- [ ] **Step 1: Run the focused AI component test suite**

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx web/src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: PASS for all targeted AI interaction tests.

- [ ] **Step 2: Detect whether the repo already has a compatible browser E2E harness**

Run: `rg -n "playwright|cypress|e2e" package.json web .`
Expected: Identify the existing browser-level test entrypoint or confirm that none exists.

- [ ] **Step 3: If a compatible browser E2E harness exists, add or run one required AI scroll-recovery scenario**

Run: `npm run test:run -- <existing-e2e-target>`
Expected: PASS for a browser-level scenario covering detach -> send -> follow recovery.

- [ ] **Step 4: If no browser E2E harness exists, manually verify the AI drawer in the browser and record that the browser-level requirement remains manual**

Checklist:
- Open a historical session and confirm step titles are visible before expansion.
- Expand one historical step and verify only that step loads.
- Collapse and re-expand a completed active-session step.
- Scroll upward, send a new message, and confirm the viewport snaps to the latest turn and follows the streamed reply.

- [ ] **Step 5: Commit the final verification or E2E additions**

```bash
git add <any-e2e-files-if-added>
git commit -m "test(ai): verify session interaction repair"
```
