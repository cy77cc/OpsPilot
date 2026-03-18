# AI Assistant Reply Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an integrated assistant reply experience in the existing AI copilot that turns backend SSE events into one continuous assistant answer with embedded status, activity, summary, and markdown content.

**Architecture:** Keep the existing drawer, scene routing, and markdown streaming stack, but introduce a frontend-owned assistant reply runtime contract. Use a small reducer layer to translate SSE events into runtime state, pass that runtime through the `XChatMessage` shape, and render assistant replies with a dedicated React renderer that composes runtime UI around the streamed markdown body without mutating the markdown AST.

**Tech Stack:** React 19, TypeScript, `@ant-design/x`, `@ant-design/x-sdk`, `@ant-design/x-markdown`, `antd-style`, Vitest, Testing Library.

---

## File Structure

### Core message and reducer files

- Modify: `web/src/components/AI/types.ts`
  - extend `XChatMessage`
  - define assistant reply phase, activity, summary, and status types
- Create: `web/src/components/AI/replyRuntime.ts`
  - runtime creation helpers
  - reducer-like event appliers for `meta`, `agent_handoff`, `plan`, `replan`, `tool_call`, `tool_approval`, `tool_result`, `delta`, `done`, and `error`
  - history hydration helpers for persisted `runtime`
- Create: `web/src/components/AI/replyRuntime.test.ts`
  - unit tests for reducer semantics and history precedence

### Provider integration files

- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
  - preserve runtime metadata while streaming
  - coalesce tool activity rows by `call_id`
  - convert event stream into `{ content, runtime }`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
  - cover runtime updates, placeholder replacement, soft timeout, and `done`

### Assistant rendering files

- Create: `web/src/components/AI/AssistantReply.tsx`
  - dedicated assistant reply renderer
  - render phase, activity feed, inline summary, markdown body, and footer status as one composition
- Create: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
  - focused rendering tests for integrated reply layout
- Modify: `web/src/components/AI/CopilotSurface.tsx`
  - pass full assistant message objects to the assistant renderer
  - keep user bubbles unchanged
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
  - assert assistant rendering uses runtime-aware composition while preserving markdown streaming flags

### History compatibility files

- Modify: `web/src/components/AI/CopilotSurface.tsx`
  - consume a dedicated history hydration helper instead of embedding hydration logic inline
- Create: `web/src/components/AI/historyRuntime.ts`
  - preserve persisted `message.runtime` during `defaultMessages()`
  - deterministic fallback for plain `content`
  - deterministic synthesis from replay `turns` and `blocks` when persisted `runtime` is absent
- Create: `web/src/components/AI/historyRuntime.test.ts`
  - unit tests for replay precedence and historical status mapping
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
  - cover persisted runtime hydration and plain-content fallback

## Chunk 1: Runtime Contract And Streaming Reducer

### Task 1: Extend the assistant message contract

**Files:**
- Modify: `web/src/components/AI/types.ts`
- Test: `web/src/components/AI/replyRuntime.test.ts`

- [ ] **Step 1: Write the failing type-driven runtime tests**

```ts
import { describe, expect, it } from 'vitest';
import { createEmptyAssistantRuntime } from '../replyRuntime';

describe('assistant reply runtime shape', () => {
  it('creates an empty runtime with append-ready activity state', () => {
    expect(createEmptyAssistantRuntime()).toEqual({
      activities: [],
      phase: undefined,
      phaseLabel: undefined,
      summary: undefined,
      status: undefined,
    });
  });

  it('defines spec-required phase, activity, summary, and status fields', () => {
    const runtime = createEmptyAssistantRuntime();

    expect(runtime.activities).toEqual([]);
    expectTypeOf(runtime.phase).toEqualTypeOf<
      'preparing' | 'identifying' | 'planning' | 'executing' | 'summarizing' | 'completed' | 'interrupted' | undefined
    >();
    expectTypeOf(runtime.summary).toEqualTypeOf<
      | {
          title?: string;
          items?: Array<{ label: string; value: string; tone?: 'default' | 'success' | 'warning' | 'danger' }>;
        }
      | undefined
    >();
    expectTypeOf(runtime.status).toEqualTypeOf<
      | { kind: 'streaming' | 'completed' | 'soft-timeout' | 'error' | 'interrupted'; label: string }
      | undefined
    >();
    expectTypeOf(runtime.activities).toEqualTypeOf<
      Array<{
        id: string;
        kind: 'agent_handoff' | 'plan' | 'replan' | 'tool_call' | 'tool_approval' | 'tool_result' | 'hint' | 'error';
        label: string;
        detail?: string;
        status?: 'pending' | 'active' | 'done' | 'error';
        createdAt?: string;
      }>
    >();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/replyRuntime.test.ts`
Expected: FAIL with missing `replyRuntime.ts` module or missing exported runtime helpers

- [ ] **Step 3: Extend `XChatMessage` and runtime types**

```ts
export interface AssistantReplyRuntimeStatus {
  kind: 'streaming' | 'completed' | 'soft-timeout' | 'error' | 'interrupted';
  label: string;
}

export interface XChatMessage {
  role: 'user' | 'assistant';
  content: string;
  runtime?: AssistantReplyRuntime;
}
```

- [ ] **Step 4: Run test to verify the contract compiles**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/replyRuntime.test.ts`
Expected: PASS for the empty runtime test, or fail later on reducer helpers not yet implemented

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/replyRuntime.test.ts
git commit -m "feat: add assistant reply runtime contract"
```

### Task 2: Build the reply runtime reducer helpers

**Files:**
- Create: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/replyRuntime.test.ts`

- [ ] **Step 1: Write failing reducer tests for core event semantics**

```ts
it('replaces placeholder status when the first delta arrives', () => {
  const state = applyDelta(
    {
      content: '[准备中]',
      runtime: createEmptyAssistantRuntime(),
    },
    { content: '第一段' },
  );

  expect(state.content).toBe('第一段');
});

it('coalesces duplicate tool activity rows by call id', () => {
  let runtime = applyToolCall(createEmptyAssistantRuntime(), {
    call_id: 'call-1',
    tool_name: 'kubectl_describe',
    arguments: {},
  });
  runtime = applyToolCall(runtime, {
    call_id: 'call-1',
    tool_name: 'kubectl_describe',
    arguments: {},
  });

  expect(runtime.activities).toHaveLength(1);
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/replyRuntime.test.ts`
Expected: FAIL with missing reducer exports such as `applyDelta` and `applyToolCall`

- [ ] **Step 3: Implement the minimal reducer helpers**

```ts
export function applyDelta(
  message: Pick<XChatMessage, 'content' | 'runtime'>,
  payload: { content: string },
): Pick<XChatMessage, 'content' | 'runtime'> {
  const next = payload.content || '';
  return {
    content: message.content && message.content !== '[准备中]' ? `${message.content}${next}` : next,
    runtime: message.runtime,
  };
}
```

- [ ] **Step 4: Add reducer coverage for `meta`, `agent_handoff`, `plan`, `replan`, `tool_approval`, `tool_result`, `done`, `tool_timeout_soft`, recoverable `error`, and terminal `error`**

```ts
it('marks soft timeout as transient footer status', () => {
  const runtime = applySoftTimeout(createEmptyAssistantRuntime());
  expect(runtime.status).toEqual({
    kind: 'soft-timeout',
    label: '工具执行较慢，正在继续等待结果…',
  });
});

it('keeps streamed markdown while surfacing recoverable errors as hints', () => {
  const state = applyRecoverableError(
    {
      content: '已经拿到部分结果',
      runtime: createEmptyAssistantRuntime(),
    },
    { message: '工具执行较慢，正在继续等待结果…' },
  );

  expect(state.content).toBe('已经拿到部分结果');
  expect(state.runtime?.status).toEqual({
    kind: 'soft-timeout',
    label: '工具执行较慢，正在继续等待结果…',
  });
});
```

- [ ] **Step 5: Run reducer test file**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/replyRuntime.test.ts`
Expected: PASS with explicit coverage for plan supersession, duplicate `call_id` coalescing, first-delta replacement, and terminal status preservation

- [ ] **Step 6: Commit**

```bash
git add web/src/components/AI/replyRuntime.ts web/src/components/AI/replyRuntime.test.ts
git commit -m "feat: add assistant reply runtime reducer"
```

### Task 3: Integrate runtime state into `PlatformChatProvider`

**Files:**
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: Write failing provider tests for runtime-aware streaming**

```ts
it('preserves runtime activities while streaming visible markdown', async () => {
  const request = new PlatformChatRequest();
  const onUpdate = vi.fn();
  request.options.callbacks = { onUpdate, onSuccess: vi.fn(), onError: vi.fn() };

  vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
    handlers.onToolCall?.({ call_id: 'call-1', tool_name: 'kubectl_get', arguments: {} });
    handlers.onDelta?.({ content: '巡检完成' });
    handlers.onDone?.({ run_id: 'run-1', status: 'completed', iterations: 1 });
  });

  request.run({ message: 'hi', scene: 'cluster' });
  await request.asyncHandler;

  expect(onUpdate).toHaveBeenCalledWith(
    expect.objectContaining({
      runtime: expect.objectContaining({
        activities: [expect.objectContaining({ id: 'call-1' })],
      }),
    }),
    expect.any(Headers),
  );
});

it('supersedes planning activities on replan and exposes tool approval state', async () => {
  const request = new PlatformChatRequest();
  const onUpdate = vi.fn();
  request.options.callbacks = { onUpdate, onSuccess: vi.fn(), onError: vi.fn() };

  vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
    handlers.onMeta?.({ session_id: 'sess-1', run_id: 'run-1', turn: 1 });
    handlers.onPlan?.({ steps: ['检查节点'], iteration: 0 });
    handlers.onReplan?.({ steps: ['检查节点', '汇总异常'], completed: 1, iteration: 1, is_final: false });
    handlers.onToolApproval?.({
      approval_id: 'approval-1',
      call_id: 'call-1',
      tool_name: 'restart_workload',
      preview: {},
      timeout_seconds: 300,
    });
    handlers.onError?.({ message: '工具执行较慢，正在继续等待结果…', code: 'tool_timeout_soft', recoverable: true });
  });

  request.run({ message: 'hi', scene: 'cluster' });
  await request.asyncHandler;

  expect(onUpdate).toHaveBeenCalledWith(
    expect.objectContaining({
      runtime: expect.objectContaining({
        activities: expect.arrayContaining([
          expect.objectContaining({ kind: 'replan' }),
          expect.objectContaining({ kind: 'tool_approval', id: 'call-1' }),
        ]),
        status: expect.objectContaining({ kind: 'soft-timeout' }),
      }),
    }),
    expect.any(Headers),
  );
});

it('projects agent handoff, tool result, and terminal error without losing partial markdown', async () => {
  const request = new PlatformChatRequest();
  const onUpdate = vi.fn();
  const onError = vi.fn();
  request.options.callbacks = { onUpdate, onSuccess: vi.fn(), onError };

  vi.mocked(aiApi.chatStream).mockImplementation(async (_params, handlers) => {
    handlers.onAgentHandoff?.({ from: 'OpsPilotAgent', to: 'DiagnosisAgent', intent: 'diagnosis' });
    handlers.onDelta?.({ content: '已经拿到部分结果' });
    handlers.onToolResult?.({ call_id: 'call-1', tool_name: 'kubectl_get', content: 'node-1 ok' });
    handlers.onError?.({ message: 'stream failed', code: 'stream_failed', recoverable: false });
  });

  request.run({ message: 'hi', scene: 'cluster' });
  await request.asyncHandler;

  expect(onUpdate).toHaveBeenCalledWith(
    expect.objectContaining({
      content: '已经拿到部分结果',
      runtime: expect.objectContaining({
        activities: expect.arrayContaining([
          expect.objectContaining({ kind: 'agent_handoff' }),
          expect.objectContaining({ kind: 'tool_result', id: 'call-1' }),
        ]),
      }),
    }),
    expect.any(Headers),
  );
  expect(onError).toHaveBeenCalledWith(
    expect.any(Error),
    expect.anything(),
    expect.any(Headers),
  );
});
```

- [ ] **Step 2: Run provider tests to verify they fail**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: FAIL because provider callbacks only emit `{ content, mode }` today

- [ ] **Step 3: Refactor `PlatformChatRequest` to emit runtime-aware chunks**

```ts
type PlatformStreamChunk = {
  content: string;
  mode?: 'replace' | 'append';
  runtime?: AssistantReplyRuntime;
};
```

- [ ] **Step 4: Wire all supported SSE events into the reducer helpers**

```ts
onToolResult: (payload) => {
  runtime = applyToolResult(runtime, payload);
  emitVisibleChunk(currentContent, runtime, 'append');
}
```

Required coverage at this step:

- `meta`
- `agent_handoff`
- `plan`
- `replan`
- `tool_call`
- `tool_approval`
- `tool_result`
- recoverable `error`
- terminal `error`
- `done`

- [ ] **Step 5: Run provider tests**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS with runtime propagation, placeholder replacement, replan supersession, approval-state transitions, recoverable-error propagation, soft-timeout clearing, and completed footer state
Expected: PASS with runtime propagation, placeholder replacement, agent-handoff projection, tool-result completion, replan supersession, approval-state transitions, recoverable-error propagation, terminal-error preservation, soft-timeout clearing, and completed footer state

- [ ] **Step 6: Commit**

```bash
git add web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/types.ts web/src/components/AI/replyRuntime.ts
git commit -m "feat: project assistant runtime from stream events"
```

## Chunk 2: Integrated Assistant Rendering And History Fallback

### Task 4: Create the dedicated assistant reply renderer

**Files:**
- Create: `web/src/components/AI/AssistantReply.tsx`
- Create: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Write failing renderer tests for integrated reply composition**

```tsx
it('renders phase, activities, markdown body, and footer as one assistant reply', () => {
  render(
    <AssistantReply
      content="发现 2 个异常节点"
      status="updating"
      runtime={{
        phase: 'executing',
        phaseLabel: '诊断助手正在巡检节点',
        activities: [{ id: 'call-1', kind: 'tool_call', label: '正在获取节点状态', status: 'done' }],
        status: { kind: 'streaming', label: '持续生成中' },
      }}
    />,
  );

  expect(screen.getByText('诊断助手正在巡检节点')).toBeInTheDocument();
  expect(screen.getByText('正在获取节点状态')).toBeInTheDocument();
  expect(screen.getByTestId('x-markdown')).toHaveTextContent('发现 2 个异常节点');
});

it('renders runtime summary inline without duplicating the markdown body', () => {
  render(
    <AssistantReply
      content="建议先处理 node-2 的磁盘压力。"
      status="success"
      runtime={{
        activities: [],
        summary: {
          title: '巡检摘要',
          items: [
            { label: '节点总数', value: '3' },
            { label: '高风险', value: '1', tone: 'danger' },
          ],
        },
        status: { kind: 'completed', label: '已生成' },
      }}
    />,
  );

  expect(screen.getByText('巡检摘要')).toBeInTheDocument();
  expect(screen.getByText('节点总数')).toBeInTheDocument();
  expect(screen.getAllByText('建议先处理 node-2 的磁盘压力。')).toHaveLength(1);
});
```

- [ ] **Step 2: Run renderer tests to verify they fail**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: FAIL with missing `AssistantReply.tsx`

- [ ] **Step 3: Implement the minimal integrated assistant renderer**

```tsx
export function AssistantReply({ content, runtime, status }: Props) {
  return (
    <div>
      {runtime?.phaseLabel ? <div>{runtime.phaseLabel}</div> : null}
      {runtime?.activities?.length ? <ul>{runtime.activities.map((item) => <li key={item.id}>{item.label}</li>)}</ul> : null}
      <XMarkdown content={content} streaming={{ hasNextChunk: status === 'loading' || status === 'updating', enableAnimation: true }} />
      {runtime?.status ? <div>{runtime.status.label}</div> : null}
    </div>
  );
}
```

- [ ] **Step 4: Implement inline summary rendering and spec-aligned styling**

```tsx
// Add summary rows between activities and markdown.
// Use token-based tonal backgrounds, subtle separators, and no card borders.
// Keep tables/code blocks readable and keep activity/footer mounted while content streams.
```

Acceptance criteria:

- Summary rows render when `runtime.summary` exists
- No section uses heavy bordered card chrome
- Code blocks and markdown tables remain readable
- Activity feed and footer status stay mounted across `loading` -> `success` transitions

- [ ] **Step 5: Run renderer tests**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS with one composed reply containing phase, activities, summary, markdown, and footer with no duplicate summary/body content

- [ ] **Step 6: Commit**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat: add integrated assistant reply renderer"
```

### Task 5: Replace assistant bubble rendering in `CopilotSurface`

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Test: `web/src/components/AI/__tests__/CopilotSurface.errorFallback.test.ts`

- [ ] **Step 1: Write failing `CopilotSurface` tests for runtime-aware assistant items**

```tsx
it('passes full assistant message objects into the assistant renderer', () => {
  mockUseXChat.mockReturnValue({
    messages: [
      {
        id: 'assistant-1',
        status: 'updating',
        message: {
          role: 'assistant',
          content: 'hello',
          runtime: { phase: 'planning', phaseLabel: '正在规划', activities: [] },
        },
      },
    ],
    onRequest: vi.fn(),
    isRequesting: true,
    queueRequest: vi.fn(),
  });

  render(
    <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
      <CopilotSurface open onClose={() => undefined} />
    </MemoryRouter>,
  );

  expect(screen.getByText('正在规划')).toBeInTheDocument();
  expect(screen.getByTestId('x-markdown')).toHaveTextContent('hello');
});
```

- [ ] **Step 2: Run surface tests to verify they fail**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/__tests__/CopilotSurface.test.tsx src/components/AI/__tests__/CopilotSurface.errorFallback.test.ts`
Expected: FAIL because `Bubble.List` currently flattens assistant items to `content` only

- [ ] **Step 3: Render assistant bubbles with `AssistantReply` and keep user bubbles unchanged**

```tsx
contentRender: (_content, info) => (
  <AssistantReply
    content={String(info.content ?? '')}
    runtime={info.message?.runtime}
    status={info.status}
  />
)
```

- [ ] **Step 4: Preserve markdown fallback and existing error behavior**

```ts
const content = buildAssistantErrorContent(messageInfo?.message?.content, error.message || 'Request failed');
```

- [ ] **Step 5: Run surface tests**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/__tests__/CopilotSurface.test.tsx src/components/AI/__tests__/CopilotSurface.errorFallback.test.ts`
Expected: PASS with runtime-aware assistant rendering and unchanged partial-content error fallback

- [ ] **Step 6: Commit**

```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/__tests__/CopilotSurface.test.tsx web/src/components/AI/__tests__/CopilotSurface.errorFallback.test.ts web/src/components/AI/AssistantReply.tsx
git commit -m "feat: render assistant replies as integrated compositions"
```

### Task 6: Hydrate persisted history and replay data without breaking plain markdown fallback

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Create: `web/src/components/AI/historyRuntime.ts`
- Create: `web/src/components/AI/historyRuntime.test.ts`
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`

- [ ] **Step 1: Write failing history hydration tests for precedence and fallback**

```tsx
it('preserves persisted assistant runtime from session history', async () => {
  vi.mocked(aiApi.getSession).mockResolvedValue({
    data: {
      messages: [
        {
          role: 'assistant',
          content: '历史回答',
          status: 'done',
          runtime: {
            phase: 'completed',
            phaseLabel: '已完成诊断',
            activities: [],
            status: { kind: 'completed', label: '已生成' },
          },
        },
      ],
    },
  } as any);

  render(
    <MemoryRouter initialEntries={['/deployment/infrastructure/clusters/42']}>
      <CopilotSurface open onClose={() => undefined} />
    </MemoryRouter>,
  );

  expect(await screen.findByText('已完成诊断')).toBeInTheDocument();
  expect(await screen.findByText('已生成')).toBeInTheDocument();
});

it('synthesizes runtime from replay turns and blocks when persisted runtime is absent', () => {
  const hydrated = hydrateAssistantHistoryMessage(
    {
      role: 'assistant',
      content: '历史回答',
      status: 'done',
    } as any,
    [
      {
        role: 'assistant',
        blocks: [
          { blockType: 'phase', contentJson: { phase: 'completed', phaseLabel: '已完成诊断' } },
          { blockType: 'summary', contentJson: { title: '巡检摘要', items: [{ label: '高风险', value: '1' }] } },
        ],
      },
    ] as any,
  );

  expect(hydrated.runtime?.phaseLabel).toBe('已完成诊断');
  expect(hydrated.runtime?.summary?.title).toBe('巡检摘要');
});

it('falls back to markdown-only history when runtime and replay blocks are absent', () => {
  const hydrated = hydrateAssistantHistoryMessage(
    { role: 'assistant', content: '纯文本历史', status: 'done' } as any,
    [],
  );

  expect(hydrated.runtime).toBeUndefined();
  expect(hydrated.content).toBe('纯文本历史');
});
```

- [ ] **Step 2: Run history tests to verify they fail**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/historyRuntime.test.ts src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: FAIL because persisted runtime is discarded, replay data is not synthesized, and markdown-only fallback is not centralized

- [ ] **Step 3: Create `historyRuntime.ts` and preserve persisted runtime with deterministic replay synthesis**

```ts
export function hydrateAssistantHistoryMessage(
  message: AIMessage,
  turns: AIReplayTurn[] = [],
  blocks: AIReplayBlock[] = [],
): XChatMessage {
  if (message.runtime) {
    return { role: 'assistant', content: message.content || '', runtime: message.runtime as any };
  }
  const synthesized = synthesizeRuntimeFromReplay(turns, blocks);
  return synthesized
    ? { role: 'assistant', content: message.content || '', runtime: synthesized }
    : { role: 'assistant', content: message.content || '' };
}
```

- [ ] **Step 4: Add atomic history-runtime tests for replay precedence and footer status mapping**

```ts
it('maps historical done status to completed footer state', () => {
  const hydrated = hydrateAssistantHistoryMessage(
    { role: 'assistant', content: '历史回答', status: 'done', runtime: { activities: [] } } as any,
    [],
    [],
  );

  expect(hydrated.runtime?.status).toEqual({
    kind: 'completed',
    label: '已生成',
  });
});
```

- [ ] **Step 5: Run history and renderer tests**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/historyRuntime.test.ts src/components/AI/__tests__/CopilotSurface.test.tsx src/components/AI/__tests__/AssistantReply.test.tsx src/components/AI/replyRuntime.test.ts`
Expected: PASS with persisted runtime precedence, deterministic replay synthesis from `turns` and `blocks`, historical footer status mapping, and plain markdown-only history still rendering

- [ ] **Step 6: Commit**

```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/historyRuntime.ts web/src/components/AI/historyRuntime.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "feat: hydrate assistant reply runtime from history"
```

### Task 7: Run focused verification and repository checks

**Files:**
- Modify: none
- Test: `web/src/components/AI/replyRuntime.test.ts`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Test: `web/src/components/AI/__tests__/CopilotSurface.errorFallback.test.ts`

- [ ] **Step 1: Run the focused assistant reply test suite**

Run: `cd /root/project/k8s-manage/web && npm run test:run -- src/components/AI/replyRuntime.test.ts src/components/AI/historyRuntime.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts src/components/AI/__tests__/AssistantReply.test.tsx src/components/AI/__tests__/CopilotSurface.test.tsx src/components/AI/__tests__/CopilotSurface.errorFallback.test.ts`
Expected: PASS with no skipped runtime coverage and no duplicate-content failures

- [ ] **Step 2: Run the broader frontend test suite**

Run: `cd /root/project/k8s-manage/web && npm run test:run`
Expected: PASS with no regressions in unrelated AI surface tests

- [ ] **Step 3: Run the production build**

Run: `cd /root/project/k8s-manage/web && npm run build`
Expected: PASS with no TypeScript errors and no invalid assistant renderer props

- [ ] **Step 4: Review the diff for scope control**

Run: `cd /root/project/k8s-manage && git diff --stat`
Expected: Only assistant reply runtime, provider, surface, renderer, and related tests changed

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/replyRuntime.test.ts web/src/components/AI/historyRuntime.ts web/src/components/AI/historyRuntime.test.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx web/src/components/AI/CopilotSurface.tsx web/src/components/AI/__tests__/CopilotSurface.test.tsx web/src/components/AI/__tests__/CopilotSurface.errorFallback.test.ts
git commit -m "feat: land integrated assistant reply experience"
```
