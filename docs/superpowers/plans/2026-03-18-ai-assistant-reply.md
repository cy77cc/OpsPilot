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
// === Phase Types ===
export type AssistantReplyPhase =
  | 'preparing'
  | 'identifying'
  | 'planning'
  | 'executing'
  | 'summarizing'
  | 'completed'
  | 'interrupted';

// === Activity Types ===
export type AssistantReplyActivityKind =
  | 'agent_handoff'
  | 'plan'
  | 'replan'
  | 'tool_call'
  | 'tool_approval'
  | 'tool_result'
  | 'hint'
  | 'error';

export type AssistantReplyActivityStatus = 'pending' | 'active' | 'done' | 'error';

export interface AssistantReplyActivity {
  id: string;
  kind: AssistantReplyActivityKind;
  label: string;
  detail?: string;
  status?: AssistantReplyActivityStatus;
  createdAt?: string;
}

// === Summary Types ===
export type AssistantSummaryTone = 'default' | 'success' | 'warning' | 'danger';

export interface AssistantReplySummary {
  title?: string;
  items?: Array<{ label: string; value: string; tone?: AssistantSummaryTone }>;
}

// === Status Types ===
export type AssistantReplyStatusKind = 'streaming' | 'completed' | 'soft-timeout' | 'error' | 'interrupted';

export interface AssistantReplyRuntimeStatus {
  kind: AssistantReplyStatusKind;
  label: string;
}

// === Runtime Aggregate ===
export interface AssistantReplyRuntime {
  phase?: AssistantReplyPhase;
  phaseLabel?: string;
  activities: AssistantReplyActivity[];
  summary?: AssistantReplySummary;
  status?: AssistantReplyRuntimeStatus;
}

// === Message Types ===
export interface XChatMessage {
  role: 'user' | 'assistant';
  content: string;
  runtime?: AssistantReplyRuntime;
}
```

**重要说明：**
- `AssistantReplyRuntime` 是前端运行时状态，由 reducer 从 SSE 事件合成
- `activities` 数组有上限保护（见 Task 2 Step 3），防止异常情况下的内存溢出
- `status.kind` 包含 `soft-timeout` 用于临时超时提示，成功事件后自动清除

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
// === Constants ===
const MAX_ACTIVITIES = 50; // 活动列表上限，防止内存溢出
const PLACEHOLDER_CONTENT = '[准备中]';

// === Helper: 活动列表上限保护 ===
function trimActivities(activities: AssistantReplyActivity[]): AssistantReplyActivity[] {
  if (activities.length <= MAX_ACTIVITIES) {
    return activities;
  }
  return activities.slice(-MAX_ACTIVITIES);
}

// === Helper: 清除软超时状态 ===
// 设计方案 §2.1: soft-timeout 在下一个成功事件后自动清除
function clearSoftTimeout(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  if (runtime.status?.kind === 'soft-timeout') {
    return { ...runtime, status: undefined };
  }
  return runtime;
}

// === Reducer: 创建空运行时 ===
export function createEmptyAssistantRuntime(): AssistantReplyRuntime {
  return {
    activities: [],
    phase: undefined,
    phaseLabel: undefined,
    summary: undefined,
    status: undefined,
  };
}

// === Reducer: 应用 delta 事件 ===
export function applyDelta(
  message: Pick<XChatMessage, 'content' | 'runtime'>,
  payload: { content: string },
): Pick<XChatMessage, 'content' | 'runtime'> {
  const next = payload.content || '';
  const isPlaceholder = !message.content || message.content === PLACEHOLDER_CONTENT;
  const runtime = message.runtime ? clearSoftTimeout(message.runtime) : createEmptyAssistantRuntime();
  return {
    content: isPlaceholder ? next : `${message.content}${next}`,
    runtime,
  };
}

// === Reducer: 应用 tool_call 事件 ===
export function applyToolCall(
  runtime: AssistantReplyRuntime,
  payload: { call_id: string; tool_name: string; arguments: Record<string, unknown> },
): AssistantReplyRuntime {
  const existingIndex = runtime.activities.findIndex((a) => a.id === payload.call_id);
  if (existingIndex >= 0) {
    // 已存在则更新状态
    const updated = [...runtime.activities];
    updated[existingIndex] = { ...updated[existingIndex], status: 'active' };
    return { ...runtime, activities: trimActivities(updated) };
  }
  // 新增活动
  const activity: AssistantReplyActivity = {
    id: payload.call_id,
    kind: 'tool_call',
    label: `正在执行 ${payload.tool_name}`,
    status: 'active',
  };
  return { ...runtime, activities: trimActivities([...runtime.activities, activity]) };
}

// === Reducer: 应用 tool_result 事件 ===
export function applyToolResult(
  runtime: AssistantReplyRuntime,
  payload: { call_id: string; tool_name: string; content: string },
): AssistantReplyRuntime {
  const cleared = clearSoftTimeout(runtime);
  const activities = cleared.activities.map((a) =>
    a.id === payload.call_id ? { ...a, status: 'done' as const } : a,
  );
  return { ...cleared, activities };
}

// === Reducer: 应用 soft_timeout 事件 ===
export function applySoftTimeout(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  return {
    ...runtime,
    status: { kind: 'soft-timeout', label: '工具执行较慢，正在继续等待结果…' },
  };
}

// === Reducer: 应用 recoverable_error 事件 ===
export function applyRecoverableError(
  message: Pick<XChatMessage, 'content' | 'runtime'>,
  payload: { message: string; code?: string },
): Pick<XChatMessage, 'content' | 'runtime'> {
  const runtime = message.runtime || createEmptyAssistantRuntime();
  return {
    content: message.content,
    runtime: {
      ...runtime,
      status: { kind: 'soft-timeout', label: payload.message },
    },
  };
}

// === Reducer: 应用 terminal_error 事件 ===
export function applyTerminalError(
  message: Pick<XChatMessage, 'content' | 'runtime'>,
  payload: { message: string; code?: string },
): Pick<XChatMessage, 'content' | 'runtime'> {
  const runtime = message.runtime || createEmptyAssistantRuntime();
  return {
    content: message.content,
    runtime: {
      ...runtime,
      status: { kind: 'error', label: payload.message },
    },
  };
}

// === Reducer: 应用 done 事件 ===
export function applyDone(runtime: AssistantReplyRuntime): AssistantReplyRuntime {
  return {
    ...runtime,
    phase: 'completed',
    status: { kind: 'completed', label: '已生成' },
  };
}
```

**关键设计决策：**
1. `MAX_ACTIVITIES = 50` 限制活动列表长度，保留最近的 50 条
2. `clearSoftTimeout()` 在 `applyDelta`、`applyToolResult` 等成功事件中自动调用
3. `applyDelta` 会清除软超时状态，符合设计方案 §2.1 的要求

- [ ] **Step 4: Add reducer coverage for `meta`, `agent_handoff`, `plan`, `replan`, `tool_approval`, `tool_result`, `done`, `tool_timeout_soft`, recoverable `error`, and terminal `error`**

```ts
it('marks soft timeout as transient footer status', () => {
  const runtime = applySoftTimeout(createEmptyAssistantRuntime());
  expect(runtime.status).toEqual({
    kind: 'soft-timeout',
    label: '工具执行较慢，正在继续等待结果…',
  });
});

it('clears soft timeout when the next delta arrives', () => {
  const state = applyDelta(
    {
      content: '已有内容',
      runtime: {
        ...createEmptyAssistantRuntime(),
        status: { kind: 'soft-timeout', label: '工具执行较慢，正在继续等待结果…' },
      },
    },
    { content: '新内容' },
  );

  expect(state.runtime?.status).toBeUndefined();
  expect(state.content).toBe('已有内容新内容');
});

it('clears soft timeout when tool result arrives', () => {
  const runtime = applyToolResult(
    {
      ...createEmptyAssistantRuntime(),
      status: { kind: 'soft-timeout', label: '工具执行较慢，正在继续等待结果…' },
      activities: [{ id: 'call-1', kind: 'tool_call', label: 'test', status: 'active' }],
    },
    { call_id: 'call-1', tool_name: 'test', content: 'result' },
  );

  expect(runtime.status).toBeUndefined();
  expect(runtime.activities[0].status).toBe('done');
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

it('preserves partial markdown on terminal error', () => {
  const state = applyTerminalError(
    {
      content: '部分回答内容',
      runtime: createEmptyAssistantRuntime(),
    },
    { message: '流式传输失败', code: 'stream_failed' },
  );

  expect(state.content).toBe('部分回答内容');
  expect(state.runtime?.status).toEqual({
    kind: 'error',
    label: '流式传输失败',
  });
});

it('limits activities to MAX_ACTIVITIES to prevent memory overflow', () => {
  let runtime = createEmptyAssistantRuntime();
  for (let i = 0; i < 60; i++) {
    runtime = applyToolCall(runtime, {
      call_id: `call-${i}`,
      tool_name: `tool-${i}`,
      arguments: {},
    });
  }

  expect(runtime.activities.length).toBeLessThanOrEqual(50);
  // 应该保留最近的 50 条
  expect(runtime.activities[0].id).toBe('call-10');
  expect(runtime.activities[49].id).toBe('call-59');
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

**3.1 修改 `PlatformStreamChunk` 类型**

```ts
// types.ts - 扩展 chunk 类型
export interface PlatformStreamChunk {
  content: string;
  mode?: 'replace' | 'append';
  runtime?: AssistantReplyRuntime;
}
```

**3.2 修改 `PlatformChatRequest` 内部状态管理**

```ts
// PlatformChatProvider.ts - PlatformChatRequest 类内部
import {
  createEmptyAssistantRuntime,
  applyDelta,
  applyToolCall,
  applyToolResult,
  applySoftTimeout,
  applyRecoverableError,
  applyTerminalError,
  applyDone,
} from '../replyRuntime';
import type { AssistantReplyRuntime } from '../types';

// 在 run() 方法内部维护 runtime 状态
run(params?: ChatRequest) {
  // ... 现有代码 ...

  let runtime = createEmptyAssistantRuntime();
  let currentContent = '';

  const emitVisibleChunk = (content: string, runtimeState: AssistantReplyRuntime, mode: 'replace' | 'append') => {
    const chunk: PlatformStreamChunk = { content, mode, runtime: runtimeState };
    hasVisibleContent = true;
    visibleChunks.push(chunk);
    this.options.callbacks?.onUpdate?.(chunk, headers);
  };

  const handlers: A2UIStreamHandlers = {
    onMeta: (payload) => {
      this.onMeta?.(payload);
      runtime = { ...runtime, phase: 'preparing' };
      emitStatus('preparing', '[准备中]');
    },

    onAgentHandoff: (payload) => {
      const status = resolveAgentStatus(payload.to);
      runtime = {
        ...runtime,
        phase: status.stage === 'agent' ? 'executing' : 'identifying',
        activities: [
          ...runtime.activities,
          {
            id: `handoff-${Date.now()}`,
            kind: 'agent_handoff',
            label: `切换到 ${resolveAgentLabel(payload.to)}`,
            status: 'done',
          },
        ],
      };
      emitStatus(status.stage, status.content);
    },

    onPlan: (payload) => {
      runtime = {
        ...runtime,
        phase: 'planning',
        activities: [
          ...runtime.activities.filter(a => a.kind !== 'plan'),
          {
            id: `plan-${payload.iteration}`,
            kind: 'plan',
            label: '正在规划任务',
            detail: payload.steps.join(' → '),
            status: 'active',
          },
        ],
      };
      emitStatus('planning', '[正在规划处理方式]');
    },

    onReplan: (payload) => {
      runtime = {
        ...runtime,
        activities: [
          ...runtime.activities.filter(a => a.kind !== 'plan' && a.kind !== 'replan'),
          {
            id: `replan-${payload.iteration}`,
            kind: 'replan',
            label: `调整计划 (${payload.completed}/${payload.steps.length})`,
            detail: payload.steps.join(' → '),
            status: 'active',
          },
        ],
      };
    },

    onToolCall: (payload) => {
      runtime = applyToolCall(runtime, payload);
      // 不 emit visible chunk，只更新 runtime
    },

    onToolApproval: (payload) => {
      const existingIndex = runtime.activities.findIndex(a => a.id === payload.call_id);
      if (existingIndex >= 0) {
        const updated = [...runtime.activities];
        updated[existingIndex] = {
          ...updated[existingIndex],
          kind: 'tool_approval',
          detail: '等待审批',
          status: 'pending',
        };
        runtime = { ...runtime, activities: updated };
      }
    },

    onToolResult: (payload) => {
      runtime = applyToolResult(runtime, payload);
    },

    onDelta: (payload) => {
      currentContent = currentContent && currentContent !== '[准备中]'
        ? `${currentContent}${payload.content}`
        : payload.content;
      const result = applyDelta({ content: currentContent, runtime }, payload);
      currentContent = result.content;
      runtime = result.runtime!;
      emitVisibleChunk(currentContent, runtime, hasVisibleContent ? 'append' : 'replace');
    },

    onDone: (payload) => {
      runtime = applyDone(runtime);
    },

    onError: (payload) => {
      if (payload.code === 'tool_timeout_soft') {
        runtime = applySoftTimeout(runtime);
        return;
      }
      if (payload.recoverable) {
        const result = applyRecoverableError({ content: currentContent, runtime }, payload);
        runtime = result.runtime!;
      } else {
        const result = applyTerminalError({ content: currentContent, runtime }, payload);
        runtime = result.runtime!;
        terminalError = { error: new Error(payload.message), info: payload };
      }
    },
  };

  // ... 其余代码 ...
}
```

**3.3 修改 `transformMessage` 方法保留 runtime**

```ts
// PlatformChatProvider.ts - PlatformChatProvider 类
transformMessage(info: TransformMessage<XChatMessage, PlatformStreamChunk>): XChatMessage {
  const current = info.originMessage?.content || '';
  const chunkContent = info.chunk?.content || '';

  if (info.status === 'success') {
    const finalContent = buildFinalContent(info.chunks);
    // 保留最后一个 chunk 的 runtime
    const lastChunk = info.chunks[info.chunks.length - 1];
    return {
      role: 'assistant',
      content: finalContent || current,
      runtime: lastChunk?.runtime,
    };
  }

  if (info.chunk) {
    return {
      role: 'assistant',
      content: applyChunkContent(current, info.chunk),
      runtime: info.chunk.runtime,
    };
  }

  return {
    role: 'assistant',
    content: `${current}${chunkContent}`,
    runtime: info.originMessage?.runtime,
  };
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

**4.1 使用 `antd-style` 定义样式**

```tsx
import { createStyles } from 'antd-style';

const useAssistantReplyStyles = createStyles(({ token, css }) => ({
  container: css`
    display: flex;
    flex-direction: column;
    gap: 12px;
    width: 100%;
  `,

  // Phase 样式
  phase: css`
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: ${token.colorTextSecondary};
    padding-bottom: 8px;
    border-bottom: 1px solid ${token.colorBorderSecondary};
  `,

  phaseIcon: css`
    font-size: 14px;
    color: ${token.colorPrimary};
  `,

  // Activity 列表样式
  activities: css`
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin: 0;
    padding: 0;
    list-style: none;
  `,

  activityItem: css`
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: ${token.colorTextSecondary};
    padding: 4px 0;
  `,

  activityItemDone: css`
    color: ${token.colorTextTertiary};
  `,

  activityItemError: css`
    color: ${token.colorError};
  `,

  activityIcon: css`
    font-size: 12px;
    flex-shrink: 0;
  `,

  // Summary 样式
  summary: css`
    background: ${token.colorFillAlter};
    border-radius: 8px;
    padding: 12px 16px;
    margin: 4px 0;
  `,

  summaryTitle: css`
    font-size: 13px;
    font-weight: 500;
    color: ${token.colorText};
    margin-bottom: 8px;
  `,

  summaryItems: css`
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
  `,

  summaryItem: css`
    display: flex;
    flex-direction: column;
    gap: 2px;
  `,

  summaryItemLabel: css`
    font-size: 11px;
    color: ${token.colorTextTertiary};
  `,

  summaryItemValue: css`
    font-size: 14px;
    font-weight: 500;
  `,

  summaryItemValueDanger: css`
    color: ${token.colorError};
  `,

  summaryItemValueWarning: css`
    color: ${token.colorWarning};
  `,

  summaryItemValueSuccess: css`
    color: ${token.colorSuccess};
  `,

  // Markdown 容器样式
  markdown: css`
    width: 100%;
    max-width: 100%;
    line-height: 1.65;
    word-break: break-word;

    pre {
      overflow-x: auto;
      padding: 12px;
      border-radius: 10px;
      background: #111827;
      color: #f9fafb;
    }

    table {
      width: 100%;
      border-collapse: collapse;
    }

    th, td {
      border: 1px solid ${token.colorBorderSecondary};
      padding: 8px 10px;
      text-align: left;
    }
  `,

  // Footer 状态样式
  footer: css`
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: ${token.colorTextTertiary};
    padding-top: 8px;
    border-top: 1px dashed ${token.colorBorderSecondary};
    margin-top: 8px;
  `,

  footerStreaming: css`
    color: ${token.colorPrimary};
  `,

  footerError: css`
    color: ${token.colorError};
  `,

  footerCompleted: css`
    color: ${token.colorSuccess};
  `,
}));
```

**4.2 完整的 `AssistantReply` 组件实现**

```tsx
import React from 'react';
import { LoadingOutlined, CheckCircleOutlined, CloseCircleOutlined, SyncOutlined } from '@ant-design/icons';
import XMarkdown from '@ant-design/x-markdown';
import type { AssistantReplyRuntime, AssistantSummaryTone } from './types';

interface AssistantReplyProps {
  content: string;
  runtime?: AssistantReplyRuntime;
  status: 'loading' | 'updating' | 'success' | 'error';
}

export function AssistantReply({ content, runtime, status }: AssistantReplyProps) {
  const { styles } = useAssistantReplyStyles();
  const isStreaming = status === 'loading' || status === 'updating';

  // 获取状态图标
  const getStatusIcon = () => {
    const kind = runtime?.status?.kind;
    if (kind === 'streaming' || isStreaming) {
      return <SyncOutlined spin className={styles.activityIcon} />;
    }
    if (kind === 'completed') {
      return <CheckCircleOutlined className={styles.activityIcon} style={{ color: '#52c41a' }} />;
    }
    if (kind === 'error' || kind === 'interrupted') {
      return <CloseCircleOutlined className={styles.activityIcon} style={{ color: '#ff4d4f' }} />;
    }
    if (kind === 'soft-timeout') {
      return <LoadingOutlined className={styles.activityIcon} />;
    }
    return null;
  };

  // 获取 summary item 的样式
  const getSummaryValueStyle = (tone?: AssistantSummaryTone) => {
    switch (tone) {
      case 'danger':
        return styles.summaryItemValueDanger;
      case 'warning':
        return styles.summaryItemValueWarning;
      case 'success':
        return styles.summaryItemValueSuccess;
      default:
        return styles.summaryItemValue;
    }
  };

  // 获取 activity item 的样式
  const getActivityStyle = (activityStatus?: string) => {
    switch (activityStatus) {
      case 'done':
        return styles.activityItemDone;
      case 'error':
        return styles.activityItemError;
      default:
        return styles.activityItem;
    }
  };

  // 获取 footer 样式
  const getFooterStyle = () => {
    const kind = runtime?.status?.kind;
    if (kind === 'streaming' || isStreaming) return styles.footerStreaming;
    if (kind === 'error' || kind === 'interrupted') return styles.footerError;
    if (kind === 'completed') return styles.footerCompleted;
    return styles.footer;
  };

  return (
    <div className={styles.container}>
      {/* Phase 区域 */}
      {runtime?.phaseLabel && (
        <div className={styles.phase}>
          {getStatusIcon()}
          <span>{runtime.phaseLabel}</span>
        </div>
      )}

      {/* Activity 列表 */}
      {runtime?.activities && runtime.activities.length > 0 && (
        <ul className={styles.activities}>
          {runtime.activities.map((activity) => (
            <li key={activity.id} className={getActivityStyle(activity.status)}>
              {activity.status === 'done' ? (
                <CheckCircleOutlined className={styles.activityIcon} style={{ color: '#52c41a' }} />
              ) : activity.status === 'error' ? (
                <CloseCircleOutlined className={styles.activityIcon} style={{ color: '#ff4d4f' }} />
              ) : (
                <SyncOutlined spin className={styles.activityIcon} />
              )}
              <span>{activity.label}</span>
              {activity.detail && <span style={{ color: '#8c8c8c' }}> — {activity.detail}</span>}
            </li>
          ))}
        </ul>
      )}

      {/* Summary 区域 */}
      {runtime?.summary && (
        <div className={styles.summary}>
          {runtime.summary.title && (
            <div className={styles.summaryTitle}>{runtime.summary.title}</div>
          )}
          {runtime.summary.items && (
            <div className={styles.summaryItems}>
              {runtime.summary.items.map((item, index) => (
                <div key={index} className={styles.summaryItem}>
                  <span className={styles.summaryItemLabel}>{item.label}</span>
                  <span className={getSummaryValueStyle(item.tone)}>{item.value}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Markdown 主体 */}
      <div className={styles.markdown}>
        <XMarkdown
          content={content}
          streaming={{
            hasNextChunk: isStreaming,
            enableAnimation: true,
            animationConfig: {
              fadeDuration: 180,
              easing: 'ease-out',
            },
          }}
        />
      </div>

      {/* Footer 状态 */}
      {runtime?.status && (
        <div className={getFooterStyle()}>
          {getStatusIcon()}
          <span>{runtime.status.label}</span>
        </div>
      )}
    </div>
  );
}
```

**关键设计决策：**
1. 使用 `antd-style` 保持与现有代码风格一致
2. 无硬边框卡片设计，通过背景色、间距、虚线分隔符表达结构
3. Activity 列表保持轻量，仅显示标签和可选详情
4. Summary 区域使用浅色背景区分，但无边框
5. Footer 使用虚线分隔，避免与内容争抢视觉权重

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

- [ ] **Step 2.5: Modify `Bubble.List` items to include `runtime`**

当前 `CopilotSurface.tsx:740-750` 的实现只传递 `content`：

```tsx
// 当前代码
<Bubble.List
  autoScroll
  items={messages.map((item) => ({
    key: item.id,
    role: item.message.role,
    content: item.message.content,  // 只有 content
    loading: item.status === 'loading' && !item.message.content,
    status: item.status,
  }))}
  role={bubbleRole}
/>
```

需要修改为包含完整的 message 对象：

```tsx
// 修改后的代码
<Bubble.List
  autoScroll
  items={messages.map((item) => ({
    key: item.id,
    role: item.message.role,
    content: item.message.content,
    message: item.message,  // 传递完整 message，包含 runtime
    loading: item.status === 'loading' && !item.message.content,
    status: item.status,
  }))}
  role={bubbleRole}
/>
```

**重要说明：**
- `@ant-design/x` 的 `Bubble.List` 支持传递额外属性，`contentRender` 的 `info` 参数可以访问
- 传递 `message` 字段后，`contentRender` 中可以通过 `info.message` 访问完整消息对象

- [ ] **Step 3: Render assistant bubbles with `AssistantReply` and keep user bubbles unchanged**

**3.1 修改 `bubbleRole.assistant.contentRender`**

```tsx
import { AssistantReply } from './AssistantReply';

const bubbleRole = React.useMemo<BubbleListProps['role']>(
  () => ({
    assistant: {
      placement: 'start',
      variant: 'borderless',
      footer: (
        <div style={{ display: 'flex' }}>
          <Button type="text" size="small" icon={<CopyOutlined />} />
          <Button type="text" size="small" icon={<LikeOutlined />} />
          <Button type="text" size="small" icon={<DislikeOutlined />} />
          <Button type="text" size="small" icon={<ReloadOutlined />} />
        </div>
      ),
      styles: {
        root: {
          paddingInline: 0,
          maxWidth: '100%',
        },
        content: {
          padding: 0,
          border: 'none',
          borderRadius: 0,
          background: 'transparent',
          boxShadow: 'none',
        },
        body: {
          padding: 0,
        },
      },
      // 关键修改：使用 AssistantReply 替代 XMarkdown
      contentRender: (content: string, info) => (
        <AssistantReply
          content={content}
          runtime={info.message?.runtime}
          status={info.status}
        />
      ),
    },
    user: {
      // 保持用户消息不变
      placement: 'end',
      styles: {
        content: {
          borderRadius: 14,
          border: 'none',
          boxShadow: 'none',
        },
      },
    },
  }),
  [],
);
```

**3.2 移除旧的 XMarkdown 渲染逻辑**

当前 `CopilotSurface.tsx:493-519` 中的 `contentRender` 使用 `XMarkdown`：

```tsx
// 删除这部分
contentRender: (content: string, info) => (
  <div className={styles.markdown}>
    <XMarkdown
      content={content}
      streaming={{
        hasNextChunk: info.status === 'loading' || info.status === 'updating',
        enableAnimation: true,
        animationConfig: {
          fadeDuration: 180,
          easing: 'ease-out',
        },
      }}
      components={{
        think: ({ children }: any) => (
          <Think title="Thinking" loading={false}>
            {children}
          </Think>
        ),
        table: ({ children, ...props }: any) => (
          <div style={{ overflowX: 'auto' }}>
            <table {...props}>{children}</table>
          </div>
        ),
        code: Code
      }}
    />
  </div>
),
```

**注意：** `AssistantReply` 组件内部已经包含 `XMarkdown` 渲染和样式，无需在 `bubbleRole` 中重复定义

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
import type { AIMessage, AIReplayTurn, AIReplayBlock } from '../../api/modules/ai';
import type { XChatMessage, AssistantReplyRuntime, AssistantReplyStatusKind } from './types';

// === 历史状态映射 ===
const HISTORICAL_STATUS_MAP: Record<string, AssistantReplyStatusKind> = {
  done: 'completed',
  complete: 'completed',
  incomplete: 'streaming',
  loading: 'streaming',
  error: 'error',
  failed: 'error',
  interrupted: 'interrupted',
};

// === 从 replay blocks 合成 runtime ===
function synthesizeRuntimeFromReplay(
  turns: AIReplayTurn[],
  blocks: AIReplayBlock[],
): AssistantReplyRuntime | null {
  // 优先使用 blocks（新版 API）
  if (blocks && blocks.length > 0) {
    const runtime: AssistantReplyRuntime = { activities: [] };

    for (const block of blocks) {
      if (block.blockType === 'phase' && block.contentJson) {
        runtime.phase = block.contentJson.phase;
        runtime.phaseLabel = block.contentJson.phaseLabel;
      }
      if (block.blockType === 'summary' && block.contentJson) {
        runtime.summary = block.contentJson;
      }
      if (block.blockType === 'activity' && block.contentJson) {
        runtime.activities.push(block.contentJson);
      }
    }

    return runtime;
  }

  // 回退到 turns（旧版 API）
  if (turns && turns.length > 0) {
    const assistantTurn = turns.find(t => t.role === 'assistant');
    if (assistantTurn && assistantTurn.blocks && assistantTurn.blocks.length > 0) {
      return synthesizeRuntimeFromReplay([], assistantTurn.blocks);
    }
  }

  return null;
}

// === 映射历史状态到运行时 status ===
function mapHistoricalStatus(status?: string): AssistantReplyStatusKind | undefined {
  if (!status) return undefined;
  return HISTORICAL_STATUS_MAP[status.toLowerCase()];
}

// === 主要导出函数：水合历史消息 ===
export function hydrateAssistantHistoryMessage(
  message: AIMessage,
  turns: AIReplayTurn[] = [],
  blocks: AIReplayBlock[] = [],
): XChatMessage {
  // 1. 如果消息已有 runtime，优先使用
  if (message.runtime) {
    const runtime = message.runtime as AssistantReplyRuntime;
    // 补充 status（如果缺失）
    if (!runtime.status && message.status) {
      const statusKind = mapHistoricalStatus(message.status);
      if (statusKind) {
        runtime.status = { kind: statusKind, label: statusKind === 'completed' ? '已生成' : statusKind };
      }
    }
    return {
      role: 'assistant',
      content: message.content || '',
      runtime,
    };
  }

  // 2. 尝试从 turns/blocks 合成 runtime
  const synthesized = synthesizeRuntimeFromReplay(turns, blocks);
  if (synthesized) {
    // 补充 status
    if (!synthesized.status && message.status) {
      const statusKind = mapHistoricalStatus(message.status);
      if (statusKind) {
        synthesized.status = { kind: statusKind, label: statusKind === 'completed' ? '已生成' : statusKind };
      }
    }
    return {
      role: 'assistant',
      content: message.content || '',
      runtime: synthesized,
    };
  }

  // 3. 纯文本降级
  return {
    role: 'assistant',
    content: message.content || '',
  };
}

// === 批量水合历史消息 ===
export function hydrateHistoryMessages(
  messages: AIMessage[],
  turns: AIReplayTurn[] = [],
  blocks: AIReplayBlock[] = [],
): XChatMessage[] {
  return messages.map(msg => {
    if (msg.role === 'assistant') {
      return hydrateAssistantHistoryMessage(msg, turns, blocks);
    }
    return {
      role: 'user',
      content: msg.content || '',
    };
  });
}
```

**关键设计决策：**
1. `HISTORICAL_STATUS_MAP` 定义后端状态到前端运行时状态的映射
2. 优先使用 `message.runtime`，其次尝试从 `blocks`/`turns` 合成
3. 自动补充缺失的 `status` 字段

- [ ] **Step 4: Add atomic history-runtime tests for replay precedence and footer status mapping**

```ts
import { describe, expect, it } from 'vitest';
import { hydrateAssistantHistoryMessage } from '../historyRuntime';

describe('hydrateAssistantHistoryMessage', () => {
  it('preserves persisted assistant runtime from session history', () => {
    const hydrated = hydrateAssistantHistoryMessage(
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
      } as any,
      [],
      [],
    );

    expect(hydrated.runtime?.phaseLabel).toBe('已完成诊断');
    expect(hydrated.runtime?.status?.kind).toBe('completed');
  });

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

  it('maps historical incomplete status to streaming footer state', () => {
    const hydrated = hydrateAssistantHistoryMessage(
      { role: 'assistant', content: '中断的回答', status: 'incomplete', runtime: { activities: [] } } as any,
      [],
      [],
    );

    expect(hydrated.runtime?.status?.kind).toBe('streaming');
  });

  it('maps historical error status to error footer state', () => {
    const hydrated = hydrateAssistantHistoryMessage(
      { role: 'assistant', content: '失败的回答', status: 'error', runtime: { activities: [] } } as any,
      [],
      [],
    );

    expect(hydrated.runtime?.status?.kind).toBe('error');
  });

  it('synthesizes runtime from replay turns and blocks when persisted runtime is absent', () => {
    const hydrated = hydrateAssistantHistoryMessage(
      { role: 'assistant', content: '历史回答', status: 'done' } as any,
      [],
      [
        { blockType: 'phase', contentJson: { phase: 'completed', phaseLabel: '已完成诊断' } } as any,
        { blockType: 'summary', contentJson: { title: '巡检摘要', items: [{ label: '高风险', value: '1' }] } } as any,
      ],
    );

    expect(hydrated.runtime?.phaseLabel).toBe('已完成诊断');
    expect(hydrated.runtime?.summary?.title).toBe('巡检摘要');
    expect(hydrated.runtime?.status?.kind).toBe('completed');
  });

  it('falls back to markdown-only history when runtime and replay blocks are absent', () => {
    const hydrated = hydrateAssistantHistoryMessage(
      { role: 'assistant', content: '纯文本历史', status: 'done' } as any,
      [],
      [],
    );

    expect(hydrated.runtime).toBeUndefined();
    expect(hydrated.content).toBe('纯文本历史');
  });

  it('prefers persisted runtime over synthesized runtime', () => {
    const hydrated = hydrateAssistantHistoryMessage(
      {
        role: 'assistant',
        content: '历史回答',
        status: 'done',
        runtime: { phase: 'completed', activities: [], status: { kind: 'completed', label: '持久化的状态' } },
      } as any,
      [],
      [
        { blockType: 'phase', contentJson: { phase: 'executing', phaseLabel: '合成的状态' } } as any,
      ],
    );

    // 应该使用持久化的 runtime，而不是合成的
    expect(hydrated.runtime?.status?.label).toBe('持久化的状态');
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

---

## Appendix: Future Enhancements (Out of Scope)

以下功能不在当前实现范围内，但建议在未来迭代中考虑：

### A. Error Boundary for AssistantReply

当 `runtime` 解析失败时的降级策略：

```tsx
import { ErrorBoundary } from 'react-error-boundary';

function AssistantReplyFallback({ error, resetErrorBoundary }: FallbackProps) {
  return (
    <div>
      <XMarkdown content="抱歉，渲染出现异常" />
      <Button size="small" onClick={resetErrorBoundary}>重试</Button>
    </div>
  );
}

// 使用
<ErrorBoundary FallbackComponent={AssistantReplyFallback}>
  <AssistantReply content={content} runtime={runtime} status={status} />
</ErrorBoundary>
```

### B. Accessibility (a11y) Enhancements

为 phase/summary/status 区域添加 ARIA 标签：

```tsx
<div
  className={styles.phase}
  role="status"
  aria-live="polite"
  aria-label={`当前阶段: ${runtime?.phaseLabel || '准备中'}`}
>
  {/* ... */}
</div>

<ul
  className={styles.activities}
  role="log"
  aria-label="执行活动列表"
  aria-live="polite"
>
  {/* ... */}
</ul>
```

### C. Performance Optimizations

**C.1 Activity 列表 memo 化**

```tsx
const ActivityList = React.memo(function ActivityList({
  activities,
}: {
  activities: AssistantReplyActivity[];
}) {
  return (
    <ul className={styles.activities}>
      {activities.map((activity) => (
        <li key={activity.id}>{/* ... */}</li>
      ))}
    </ul>
  );
});
```

**C.2 Summary 区域 memo 化**

```tsx
const SummarySection = React.memo(function SummarySection({
  summary,
}: {
  summary?: AssistantReplySummary;
}) {
  if (!summary) return null;
  return (
    <div className={styles.summary}>
      {/* ... */}
    </div>
  );
});
```

**C.3 使用 `useDeferredValue` 优化流式渲染**

```tsx
import { useDeferredValue } from 'react';

export function AssistantReply({ content, runtime, status }: AssistantReplyProps) {
  const deferredContent = useDeferredValue(content);
  const isStale = content !== deferredContent;

  return (
    <div className={styles.container} style={{ opacity: isStale ? 0.7 : 1 }}>
      {/* 使用 deferredContent 进行 markdown 渲染 */}
      <XMarkdown content={deferredContent} streaming={{ ... }} />
    </div>
  );
}
```

### D. Data Persistence Considerations

**问题**：当前 `runtime` 是纯前端运行时状态，刷新页面后会丢失。

**方案选择**：
1. **纯前端状态**（当前实现）：不持久化，历史消息依赖后端 `turns`/`blocks` 合成
2. **部分持久化**：在会话结束时，将最终的 `runtime` 状态保存到 `AIMessage.runtime`
3. **全量持久化**：每次更新都保存（不推荐，增加存储压力）

**建议**：采用方案 2，在 `done` 事件时将 `runtime` 通过 API 保存到后端。

```ts
// 在 PlatformChatProvider 中
onDone: (payload) => {
  runtime = applyDone(runtime);
  // 可选：持久化最终的 runtime 状态
  aiApi.updateMessageRuntime(sessionId, messageId, runtime);
}
```
