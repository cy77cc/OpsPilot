# AI Session Projection Contract Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make assistant history render from `run projection` only, remove duplicated conclusion rendering, and collapse tool call/result display into a single inline tool reference model.

**Architecture:** Keep `ai_chat_messages` as the session skeleton, but stop treating assistant `content` as the persisted history truth. The backend should expose `run_id` on assistant session messages and persist final answer text only in `ai_run_projections.summary.content`; the frontend should hydrate assistant history from projection, render only one final markdown body, and treat a tool call/result pair as one in-place tool reference with explicit terminal states.

**Tech Stack:** Go, Gin, GORM, SQLite test DB, React 19, TypeScript, Vitest, Testing Library, Ant Design, Ant Design X Markdown.

---

## Execution Notes

- Use @superpowers:executing-plans if implementing in the current session.
- Follow @superpowers:test-driven-development for each task: write the failing test first, then the minimum code, then rerun the narrow test, then rerun the broader suite.
- Use @superpowers:verification-before-completion before claiming the feature is done.
- Work in a dedicated git worktree before executing this plan. This plan was written from `/root/project/k8s-manage`, but execution should happen in an isolated worktree.
- Suggested worktree bootstrap:

```bash
git worktree add ../k8s-manage-ai-session-projection -b feat/ai-session-projection-contract
```

- Current baseline test reality:
  - `go test ./internal/service/ai/handler ./internal/service/ai/logic ./internal/ai/runtime` passes from repo root.
  - `npm run test:run -- src/components/AI/historyProjection.test.ts src/components/AI/replyRuntime.test.ts src/components/AI/__tests__/AssistantReply.test.tsx` from `/root/project/k8s-manage/web` currently fails in `replyRuntime.test.ts` and `AssistantReply.test.tsx`. Execution tasks below intentionally rewrite those expectations as part of the new contract.

## File Map

### Backend files

- Modify: `internal/service/ai/logic/logic.go:450-503`
  Responsibility: persist projection/content/run status without rewriting assistant final text into session message content.
- Modify: `internal/service/ai/logic/logic.go:855-919`
  Responsibility: keep projection rebuild behavior as the short-window backend fallback when stored projection is missing or incomplete.
- Modify: `internal/service/ai/handler/session.go:32-99`
  Responsibility: emit different session-message payloads for user vs assistant rows; assistant rows should expose `run_id` and status, but not final `content`.
- Modify: `internal/model/ai.go:22-35`
  Responsibility: confirm model comments match the new contract; do not change schema unless implementation proves it is necessary.
- Test: `internal/service/ai/handler/session_test.go:208-269`
  Responsibility: verify assistant session messages include `run_id` and omit `content`.
- Test: `internal/service/ai/logic/logic_test.go:546-615`
  Responsibility: verify projection summary is persisted and assistant rows no longer need persisted final text.

### Frontend files

- Modify: `web/src/api/modules/ai.ts:5-20`
  Responsibility: update session message typing to reflect that assistant `content` may be absent from server payloads.
- Modify: `web/src/components/AI/types.ts:25-98`
  Responsibility: collapse tool activity semantics into a single tool reference object with explicit terminal states.
- Modify: `web/src/components/AI/historyProjection.ts:39-198`
  Responsibility: hydrate assistant history from projection only, map projection summary into the single markdown body, and stop duplicating the summary card text.
- Modify: `web/src/components/AI/replyRuntime.ts:80-360`
  Responsibility: update stream-time runtime state so one `call_id` is one tool entity, support interrupted/incomplete terminal states, and preserve step segment ordering.
- Modify: `web/src/components/AI/AssistantReply.tsx:243-552`
  Responsibility: render one final markdown body, remove duplicated summary text rendering, keep tool refs inline inside step content, and stabilize step/body layout during streaming transitions.
- Modify: `web/src/components/AI/ToolReference.tsx:80-132`
  Responsibility: show a single tool entity in `active/done/error/interrupted` states and only make terminal states interactive.
- Modify: `web/src/components/AI/CopilotSurface.tsx:620-668`
  Responsibility: wire the copy button to the final markdown body only and keep scroll anchoring stable when runtime updates swap from stream state to projection state.
- Test: `web/src/components/AI/historyProjection.test.ts:39-129`
  Responsibility: verify assistant history content comes from projection summary and projection failures no longer fall back to old assistant message content.
- Test: `web/src/components/AI/replyRuntime.test.ts:98-260`
  Responsibility: verify tool call/result coalesce into one entity and interrupted states are explicit.
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx:20-320`
  Responsibility: verify no duplicated conclusion body, inline tool refs remain inline, terminal tool states are interactive, and no extra tool activity rows appear.

## Chunk 1: Backend Contract

### Task 1: Lock the Session API Contract with Tests

**Files:**
- Modify: `internal/service/ai/handler/session_test.go:208-269`
- Test: `internal/service/ai/handler/session_test.go`

- [ ] **Step 1: Add a failing test that asserts assistant session messages omit `content`**

```go
func TestGetSession_AssistantMessageOmitsContentButKeepsRunID(t *testing.T) {
    // seed one user message and one assistant message
    // seed one run that points at the assistant message
    // call GetSession
    // assert user message has content
    // assert assistant message has run_id
    // assert assistant message does not contain the "content" key
}
```

- [ ] **Step 2: Run only the handler test and verify it fails for the right reason**

Run: `go test ./internal/service/ai/handler -run TestGetSession_AssistantMessageOmitsContentButKeepsRunID -v`
Expected: FAIL because the handler still includes assistant `content`.

- [ ] **Step 3: Add the same contract assertion for `ListSessions`**

```go
func TestListSessions_AssistantMessagesOmitContentButKeepRunID(t *testing.T) {
    // same seed shape, but call ListSessions
    // assert the embedded assistant message map omits "content"
}
```

- [ ] **Step 4: Run the second narrow test and verify it fails**

Run: `go test ./internal/service/ai/handler -run TestListSessions_AssistantMessagesOmitContentButKeepRunID -v`
Expected: FAIL because `ListSessions` still serializes `message.Content` for assistant rows.

- [ ] **Step 5: Commit the red tests**

```bash
git add internal/service/ai/handler/session_test.go
git commit -m "test(ai): define assistant session message contract"
```

### Task 2: Implement Assistant Session Skeleton Semantics

**Files:**
- Modify: `internal/service/ai/handler/session.go:43-58`
- Modify: `internal/service/ai/handler/session.go:76-90`
- Modify: `internal/service/ai/logic/logic.go:469-503`
- Modify: `internal/service/ai/logic/logic.go:581-590`
- Test: `internal/service/ai/handler/session_test.go`
- Test: `internal/service/ai/logic/logic_test.go:555-615`

- [ ] **Step 1: Write a failing logic test that assistant messages do not need final persisted content**

```go
func TestChat_PersistsProjectionSummaryWithoutAssistantMessageContent(t *testing.T) {
    // run a one-shot assistant completion
    // load the assistant AIChatMessage row
    // assert assistant.Content == "" (or remains empty placeholder)
    // assert projection summary content is present
}
```

- [ ] **Step 2: Run the narrow logic test and verify it fails**

Run: `go test ./internal/service/ai/logic -run TestChat_PersistsProjectionSummaryWithoutAssistantMessageContent -v`
Expected: FAIL because `persistRunArtifacts` still updates `content` with `assistantContent`.

- [ ] **Step 3: Update session serialization so user rows include `content`, assistant rows do not**

```go
item := gin.H{
    "id":         message.ID,
    "role":       message.Role,
    "status":     message.Status,
    "created_at": formatTime(message.CreatedAt),
}
if message.Role != "assistant" {
    item["content"] = message.Content
}
if runID != "" {
    item["run_id"] = runID
}
```

- [ ] **Step 4: Stop rewriting assistant final text into the session row in `persistRunArtifacts`**

```go
updates := map[string]any{
    "status": assistantStatusFromRunStatus(status),
}
if err := chatDAO.UpdateMessage(ctx, assistantMessageID, updates); err != nil {
    return err
}
```

- [ ] **Step 5: Preserve the synchronous projection/content upsert in the same transaction**

Do not let the implementation accidentally reduce `persistRunArtifacts` to only updating `ai_runs`.

The transaction must still durably write:

```go
for _, content := range contents {
    if err := contentDAO.Create(ctx, content); err != nil {
        return err
    }
}

if err := projectionDAO.Upsert(ctx, &model.AIRunProjection{
    ID:             uuid.NewString(),
    RunID:          runID,
    SessionID:      sessionID,
    Version:        projection.Version,
    Status:         projection.Status,
    ProjectionJSON: string(projectionJSON),
}); err != nil {
    return err
}
```

Requirement: `projection.summary.content` must be written before the run is treated as complete from the API perspective.

- [ ] **Step 6: Keep the run-level summary field as secondary metadata only**

```go
return runDAO.UpdateRunStatus(ctx, runID, aidao.AIRunStatusUpdate{
    Status:          status,
    ProgressSummary: truncateString(assistantContent, 500),
})
```

- [ ] **Step 7: Run the three narrow backend tests**

Run: `go test ./internal/service/ai/handler -run 'Test(GetSession|ListSessions)_.*Assistant.*' -v`
Expected: PASS.

Run: `go test ./internal/service/ai/logic -run TestChat_PersistsProjectionSummaryWithoutAssistantMessageContent -v`
Expected: PASS.

- [ ] **Step 8: Run the broader backend AI suites**

Run: `go test ./internal/service/ai/handler ./internal/service/ai/logic ./internal/ai/runtime`
Expected: PASS.

- [ ] **Step 9: Commit the backend contract change**

```bash
git add internal/service/ai/handler/session.go internal/service/ai/handler/session_test.go internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "feat(ai): serve assistant history from projection contract"
```

## Chunk 2: Frontend History Hydration and Reply Rendering

### Task 3: Redefine the History Hydration Contract

**Files:**
- Modify: `web/src/components/AI/historyProjection.test.ts:39-129`
- Modify: `web/src/api/modules/ai.ts:5-20`
- Modify: `web/src/components/AI/historyProjection.ts:39-198`

- [ ] **Step 1: Replace the old fallback test with a failing projection-only contract test**

```ts
it('hydrates assistant history body from projection summary only', async () => {
  (aiApi.getRunProjection as any).mockResolvedValue({
    data: {
      version: 1,
      run_id: 'run-1',
      session_id: 'sess-1',
      status: 'completed',
      summary: { content: 'projection body' },
      blocks: [],
    },
  });

  const hydrated = await hydrateAssistantHistoryFromProjection({
    id: 'msg-1',
    role: 'assistant',
    content: 'legacy body',
    run_id: 'run-1',
    timestamp: '',
  } as any);

  expect(hydrated.content).toBe('projection body');
});
```

- [ ] **Step 2: Add a failing test for missing projection summary**

```ts
it('returns an error-style placeholder when projection summary is unavailable', async () => {
  (aiApi.getRunProjection as any).mockResolvedValue({
    data: { version: 1, run_id: 'run-1', session_id: 'sess-1', status: 'completed', blocks: [] },
  });

  const hydrated = await hydrateAssistantHistoryFromProjection({
    id: 'msg-1',
    role: 'assistant',
    run_id: 'run-1',
    timestamp: '',
  } as any);

  expect(hydrated.content).toContain('不可恢复');
  expect(hydrated.runtime?.status?.kind).toBe('error');
});
```

- [ ] **Step 3: Run the historyProjection tests and verify they fail**

Run: `npm run test:run -- src/components/AI/historyProjection.test.ts`
Workdir: `/root/project/k8s-manage/web`
Expected: FAIL because assistant history still falls back to `message.content` and still duplicates summary text into `runtime.summary`.

- [ ] **Step 4: Before changing hydration, grep for any client-side persisted chat cache**

Run: `rg -n "localStorage|sessionStorage|indexedDB|persist|zustand|redux-persist" src/components/AI src`
Workdir: `/root/project/k8s-manage/web`
Expected: either no AI-history persistence, or a short list of places that must be purged or migrated.

If any AI-history persistence exists, add a tiny purge helper in the same task that removes assistant entries without `run_id` before hydration starts.

- [ ] **Step 5: Update the API typing to allow assistant `content` to be absent**

```ts
export interface AIMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  run_id?: string;
  content?: string;
  // ...
}
```

- [ ] **Step 6: Change history hydration to treat projection summary as the sole assistant body**

```ts
if (!projection?.summary?.content?.trim()) {
  return {
    id: message.id,
    role: 'assistant',
    content: '回答内容不可恢复',
    runtime: {
      activities: [],
      status: { kind: 'error', label: 'projection missing summary' },
    },
  };
}

return {
  id: message.id,
  role: 'assistant',
  content: projection.summary.content,
  runtime,
};
```

- [ ] **Step 7: Stop injecting full conclusion text into `runtime.summary`**

```ts
return {
  activities,
  plan: steps.length > 0 ? { steps } : undefined,
  status: {
    kind: projection.status === 'failed_runtime' ? 'error' : 'completed',
    label: projection.status,
  },
};
```

- [ ] **Step 8: Re-run the historyProjection tests**

Run: `npm run test:run -- src/components/AI/historyProjection.test.ts`
Workdir: `/root/project/k8s-manage/web`
Expected: PASS.

- [ ] **Step 9: Commit the hydration contract change**

```bash
git add web/src/api/modules/ai.ts web/src/components/AI/historyProjection.ts web/src/components/AI/historyProjection.test.ts
git commit -m "feat(ai): hydrate assistant history from projection only"
```

### Task 4: Collapse Tool Call and Tool Result into One Runtime Entity

**Files:**
- Modify: `web/src/components/AI/types.ts:25-98`
- Modify: `web/src/components/AI/replyRuntime.test.ts:98-260`
- Modify: `web/src/components/AI/replyRuntime.ts:80-360`
- Modify: `web/src/components/AI/historyProjection.ts:77-182`

- [ ] **Step 1: Rewrite the runtime tests around one tool entity per `call_id`**

```ts
it('coalesces tool call and result into one tool reference entity', () => {
  let runtime = applyToolCall(createEmptyAssistantRuntime(), {
    call_id: 'call-1',
    tool_name: 'kubectl_describe',
    arguments: {},
  });
  runtime = applyToolResult(runtime, {
    call_id: 'call-1',
    tool_name: 'kubectl_describe',
    content: 'ok',
  });

  expect(runtime.activities).toEqual([
    expect.objectContaining({
      id: 'call-1',
      label: 'kubectl_describe',
      status: 'done',
      rawContent: 'ok',
    }),
  ]);
});
```

- [ ] **Step 2: Add a failing runtime test for interrupted/incomplete tools**

```ts
it('marks orphaned tool refs as interrupted when the run finishes without a result', () => {
  let runtime = applyPlan(createEmptyAssistantRuntime(), { steps: ['检查'], iteration: 0 });
  runtime = applyToolCall(runtime, { call_id: 'call-1', tool_name: 'kubectl_describe', arguments: {} });
  runtime = applyDone(runtime);

  expect(runtime.activities[0]).toEqual(
    expect.objectContaining({ id: 'call-1', status: 'error', detail: expect.stringContaining('未完成') }),
  );
});
```

- [ ] **Step 3: Run the runtime tests and verify they fail**

Run: `npm run test:run -- src/components/AI/replyRuntime.test.ts`
Workdir: `/root/project/k8s-manage/web`
Expected: FAIL because `applyToolResult` still appends a `call-1:result` row and `applyDone` does not settle active tools.

- [ ] **Step 4: Simplify the runtime activity type to one tool entity**

```ts
export type AssistantReplyActivityKind =
  | 'agent_handoff'
  | 'plan'
  | 'replan'
  | 'tool'
  | 'tool_approval'
  | 'hint'
  | 'error';
```

- [ ] **Step 5: Update `applyToolCall`, `applyToolResult`, and `applyDone`**

```ts
runtime = upsertActivity(runtime, {
  id: payload.call_id,
  kind: 'tool',
  label: payload.tool_name,
  status: 'active',
  arguments: payload.arguments,
  stepIndex: activeStepIndex,
}, item => item.id === payload.call_id);
```

```ts
return upsertActivity(runtime, {
  id: payload.call_id,
  kind: 'tool',
  label: payload.tool_name,
  status: payload.status === 'error' ? 'error' : 'done',
  detail: detailContent,
  rawContent: payload.content,
  stepIndex: existing?.stepIndex ?? runtime.plan?.activeStepIndex,
  arguments: existing?.arguments,
}, item => item.id === payload.call_id);
```

```ts
const activities = runtime.activities.map((item) =>
  item.kind === 'tool' && item.status === 'active'
    ? { ...item, status: 'error', detail: item.detail || '执行未完成' }
    : item,
);
```

- [ ] **Step 6: Make history projection emit the same single-tool runtime shape**

```ts
activities.push({
  id: item.tool_call_id,
  kind: 'tool',
  label: item.tool_name,
  status: item.result?.status === 'done' ? 'done' : item.result?.status === 'error' ? 'error' : 'active',
  detail: item.result?.preview || undefined,
  rawContent: resultContent?.body_text || item.result?.preview,
  arguments: argumentContent?.body_json ? JSON.parse(argumentContent.body_json) : undefined,
  stepIndex: steps.length,
});
```

- [ ] **Step 7: Re-run the runtime tests**

Run: `npm run test:run -- src/components/AI/replyRuntime.test.ts`
Workdir: `/root/project/k8s-manage/web`
Expected: PASS.

- [ ] **Step 8: Commit the single-tool runtime model**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/replyRuntime.test.ts web/src/components/AI/historyProjection.ts
git commit -m "feat(ai): collapse tool events into single runtime references"
```

### Task 5: Rewrite Assistant Reply Rendering and Copy Semantics

**Files:**
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx:20-320`
- Modify: `web/src/components/AI/AssistantReply.tsx:243-552`
- Modify: `web/src/components/AI/ToolReference.tsx:80-132`
- Modify: `web/src/components/AI/CopilotSurface.tsx:623-668`

- [ ] **Step 1: Rewrite the reply tests around the new rendering contract**

Add or update tests to assert all of the following:

```tsx
it('renders only one final markdown body when projection-backed content is present', () => {
  render(<AssistantReply content="最终正文" runtime={{ activities: [], status: { kind: 'completed', label: '已生成' } }} />);
  expect(screen.getAllByText('最终正文')).toHaveLength(1);
  expect(screen.queryByText('结论')).not.toBeInTheDocument();
});

it('renders one inline tool reference instead of separate tool call/result rows', () => {
  // step segments: text -> tool_ref -> text
  // activities: one tool entity with status done
});
```

- [ ] **Step 2: Add a failing test for copy semantics**

```tsx
it('copy action uses final markdown body only', async () => {
  const writeText = vi.fn();
  Object.assign(navigator, { clipboard: { writeText } });

  render(/* CopilotSurface assistant footer context with runtime + content */);
  fireEvent.click(screen.getByLabelText('复制回复'));

  expect(writeText).toHaveBeenCalledWith('最终正文');
});
```

- [ ] **Step 3: Run the reply tests and verify they fail**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx`
Workdir: `/root/project/k8s-manage/web`
Expected: FAIL because `AssistantReply` still renders `runtime.summary`, still looks up `call-1:result`, and the copy button does not yet copy the normalized final body.

- [ ] **Step 4: Remove duplicated summary-body rendering from `AssistantReply`**

```tsx
{runtime.summary?.items?.length ? (
  <div className={styles.summary}>
    {/* short structured stats only */}
  </div>
) : null}

{content ? <SimpleMarkdownContent content={content} styles={styles} isStreaming={isStreaming} /> : null}
```

Constraint: do not render a summary card whose only purpose is to repeat the final conclusion paragraph.

- [ ] **Step 5: Make step rendering resolve a tool by `call_id` only**

```tsx
const activity = activityMap.get(segment.callId);
if (activity?.kind === 'tool') {
  elements.push(<ToolReference key={`tool-${segment.callId}`} activity={activity} />);
}
```

- [ ] **Step 6: Update `ToolReference` to use status-based rendering rather than `tool_call`/`tool_result` kind switching**

```tsx
const isLoading = status === 'active';
const isSuccess = status === 'done';
const isError = status === 'error';
```

Also add the explicit interrupted/incomplete label copy if `detail` contains `"未完成"` or `"异常中断"`.

- [ ] **Step 7: Wire the copy button to final markdown content only**

```tsx
<Button
  type="text"
  size="small"
  icon={<CopyOutlined />}
  aria-label="复制回复"
  onClick={() => navigator.clipboard.writeText(item.message.content || '')}
/>
```

If the surrounding component structure makes this awkward, extract a small helper that receives the normalized message body and keep the helper colocated in `CopilotSurface.tsx`.

- [ ] **Step 8: Re-run the reply-focused tests**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx`
Workdir: `/root/project/k8s-manage/web`
Expected: PASS.

- [ ] **Step 9: Commit the rendering and copy semantics**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/ToolReference.tsx web/src/components/AI/CopilotSurface.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai): render projection-backed replies without duplication"
```

## Chunk 3: Integration, Transition, and Verification

### Task 6: Stabilize Streaming-to-History Handoff

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx:520-700`
- Modify: `web/src/components/AI/historyProjection.ts:13-37`
- Test: `web/src/components/AI/historyProjection.test.ts`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Add a failing integration-style test for projection handoff without body flicker**

```tsx
it('keeps the streamed body visible until projection-backed content is ready', async () => {
  // seed a streaming message with runtime in memory
  // mock projection fetch to resolve after one tick
  // assert the streamed content remains visible during the fetch
  // assert the final projection body replaces it once available
});
```

- [ ] **Step 2: Run the narrow test and verify it fails**

Run: `npm run test:run -- src/components/AI/historyProjection.test.ts src/components/AI/__tests__/AssistantReply.test.tsx`
Workdir: `/root/project/k8s-manage/web`
Expected: FAIL because current code has no explicit “retain in-memory body until projection is ready” handoff logic.

- [ ] **Step 3: Implement the transition guard**

Use a small helper or state flag so that:

```ts
// streaming: keep message.content from applyDelta
// done: request projection silently
// projection unavailable within retry window: keep current content, mark retry state
// projection loaded: replace content source with projection.summary.content
```

Implementation constraint: do not temporarily clear the rendered body while waiting for projection. Keep the last streamed text in a ref such as `lastStreamedContentRef`, and only perform one atomic replacement after `projection.summary.content` is resolved and non-empty.

Do not introduce any new long-lived fallback to legacy session assistant `content`.

- [ ] **Step 4: Re-run the targeted frontend suites**

Run: `npm run test:run -- src/components/AI/historyProjection.test.ts src/components/AI/replyRuntime.test.ts src/components/AI/__tests__/AssistantReply.test.tsx`
Workdir: `/root/project/k8s-manage/web`
Expected: PASS.

- [ ] **Step 5: Commit the handoff stabilization**

```bash
git add web/src/components/AI/historyProjection.ts web/src/components/AI/CopilotSurface.tsx web/src/components/AI/historyProjection.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "fix(ai): stabilize projection handoff after streaming"
```

### Task 7: Final Verification and Release Notes

**Files:**
- Modify: `docs/superpowers/specs/2026-03-20-ai-session-projection-contract-design.md` only if implementation forced a design adjustment
- No new feature docs otherwise

- [ ] **Step 1: Run the backend verification suite**

Run: `go test ./internal/service/ai/handler ./internal/service/ai/logic ./internal/ai/runtime`
Expected: PASS.

- [ ] **Step 2: Run the focused frontend verification suite**

Run: `npm run test:run -- src/components/AI/historyProjection.test.ts src/components/AI/replyRuntime.test.ts src/components/AI/__tests__/AssistantReply.test.tsx`
Workdir: `/root/project/k8s-manage/web`
Expected: PASS.

- [ ] **Step 3: Run one broader frontend AI surface check**

Run: `npm run test:run -- src/components/AI`
Workdir: `/root/project/k8s-manage/web`
Expected: PASS, or at minimum no new failures outside the intentionally updated reply/runtime tests.

- [ ] **Step 4: Manually verify the release-order assumptions in code review**

Checklist:
- Backend session handlers no longer expose assistant `content`
- Frontend no longer depends on assistant `content` for history hydration
- No new local-storage/session-storage history cache was introduced
- Copy action only uses final markdown body

- [ ] **Step 5: Commit the final verification pass**

```bash
git add -A
git commit -m "test(ai): verify session projection contract rollout"
```
