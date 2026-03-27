# DeepAgents 前端交互迁移 Implementation Plan

> Superseded by `docs/superpowers/plans/2026-03-25-deepagents-unified-implementation.md`.

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不重写现有聊天框架的前提下，为 DeepAgents 迁移补齐 `ops_plan_updated`、Task Board、审批断线兜底和长内容性能保护。

**Architecture:** 保持 `ai.ts -> PlatformChatProvider -> replyRuntime -> AssistantReply` 单向数据流。新增 `ops_plan_updated` 作为 runtime 的结构化 todo 快照输入；审批状态保持“事件驱动 + 本地过渡态”原则；重内容块采用“阈值外置 + 视口挂载”降低 DOM 压力。

**Tech Stack:** TypeScript, React, Ant Design, Ant Design X, Vitest

---

## Scope Check

该 spec 仅覆盖 AI 前端交互层（协议解析、runtime 状态、UI 呈现、性能保护），是单一子系统，不再拆分。

## File Structure (Responsibilities)

### Stream protocol + API module
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.test.ts`
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
- Purpose: 增加 `ops_plan_updated` 事件类型与分发逻辑，保持未知事件容错。

### Runtime state model
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/replyRuntime.test.ts`
- Purpose: 扩展 `AssistantReplyRuntime.todos`，新增 full-snapshot 覆盖更新器。

### Stream provider integration
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Purpose: 消费 `onOpsPlanUpdated` 并写入 runtime；补强 agent 活动标签防抖。

### Reply rendering + approval fallback + heavy content perf
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/ToolReference.tsx`
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Purpose: 渲染可折叠 Task Board；`submitting` 超时转 `refresh-needed`；超长内容采用外置查看和视口挂载。

## Implementation Rules

- 使用 @test-driven-development：先写失败测试，再最小实现，再回归。
- 使用 @verification-before-completion：每个任务结束都执行命令验证。
- 保持 DRY/YAGNI：不引入第二套消息模型，统一使用 `runtime`。
- 每个任务结束单独 commit，保证可回滚。

## Chunk 1: 协议与 Runtime 建模

### Task 1: 新增 `ops_plan_updated` 协议类型与事件分发

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.test.ts`
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
- Test: `web/src/api/modules/ai.streamChunk.test.ts`

- [ ] **Step 1: 写失败测试，断言 `event: ops_plan_updated` 会命中专用 handler**

```ts
it('dispatches ops_plan_updated as full snapshot todo payload', () => {
  const handlers = { onOpsPlanUpdated: vi.fn() };
  dispatchAIStreamEvent('event: ops_plan_updated\ndata: {"todos":[{"id":"t1","content":"a","status":"pending"}]}\n\n', handlers as any);
  expect(handlers.onOpsPlanUpdated).toHaveBeenCalledWith({
    todos: [{ id: 't1', content: 'a', status: 'pending' }],
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts -t "ops_plan_updated"`
Expected: FAIL（`onOpsPlanUpdated` 未定义或未调用）

- [ ] **Step 3: 最小实现类型与分发**

```ts
export interface A2UIOpsPlanUpdatedEvent {
  todos: Array<{ id: string; content: string; status: 'pending' | 'in_progress' | 'completed' }>;
}

export interface A2UIStreamHandlers {
  onOpsPlanUpdated?: (payload: A2UIOpsPlanUpdatedEvent) => void;
}

// in dispatchAIStreamEvent
} else if (eventType === 'ops_plan_updated') {
  handlers.onOpsPlanUpdated?.(payload as A2UIOpsPlanUpdatedEvent);
}
```

- [ ] **Step 4: 运行 API 模块回归**

Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts src/api/modules/ai.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts
git commit -m "feat(ai-web): add ops_plan_updated stream event contract"
```

### Task 2: 扩展 runtime todos 模型与 full-snapshot 覆盖策略

**Files:**
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/replyRuntime.test.ts`
- Test: `web/src/components/AI/replyRuntime.test.ts`

- [ ] **Step 1: 写失败测试，断言 todos 采用覆盖而非 merge**

```ts
it('replaces runtime todos by snapshot instead of merging', () => {
  const runtime = createEmptyAssistantRuntime();
  const afterFirst = applyOpsPlanUpdated(runtime, { todos: [{ id: 'a', content: 'A', status: 'pending' }] });
  const afterSecond = applyOpsPlanUpdated(afterFirst, { todos: [{ id: 'b', content: 'B', status: 'completed' }] });
  expect(afterSecond.todos).toEqual([{ id: 'b', content: 'B', status: 'completed' }]);
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `npm run test:run -- src/components/AI/replyRuntime.test.ts -t "snapshot"`
Expected: FAIL（`applyOpsPlanUpdated` 不存在）

- [ ] **Step 3: 实现最小 runtime 更新器与类型**

```ts
export interface RuntimeTodoItem {
  id: string;
  content: string;
  status: 'pending' | 'in_progress' | 'completed';
  cluster?: string;
  namespace?: string;
  resourceType?: string;
  riskLevel?: 'low' | 'medium' | 'high' | 'critical';
}

export function applyOpsPlanUpdated(runtime: AssistantReplyRuntime, payload: { todos?: RuntimeTodoItem[] }): AssistantReplyRuntime {
  return { ...runtime, todos: [...(payload.todos || [])] };
}
```

- [ ] **Step 4: 运行 runtime 测试**

Run: `npm run test:run -- src/components/AI/replyRuntime.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/replyRuntime.test.ts
git commit -m "feat(ai-web): add runtime todo snapshot model"
```

## Chunk 2: Provider 接入与 Task Board 呈现

### Task 3: PlatformChatProvider 接入 `onOpsPlanUpdated`

**Files:**
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: 写失败测试，断言 SSE todos 会进入最新 runtime**

```ts
it('writes ops_plan_updated snapshot into runtime.todos', async () => {
  handlers.onOpsPlanUpdated?.({ todos: [{ id: 't1', content: 'Inspect pods', status: 'in_progress' }] });
  expect(lastChunk.runtime?.todos?.[0].id).toBe('t1');
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts -t "ops_plan_updated"`
Expected: FAIL

- [ ] **Step 3: 最小实现 provider handler**

```ts
onOpsPlanUpdated: (payload) => {
  runtime = applyOpsPlanUpdated(runtime, payload);
  emitRuntimeOnlyUpdate();
},
```

- [ ] **Step 4: 运行 provider 回归**

Run: `npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "feat(ai-web): wire ops_plan_updated into platform provider"
```

### Task 4: AssistantReply 顶部渲染可折叠 Task Board

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: 写失败测试，断言 todos 存在时显示 Task Board 且可折叠**

```tsx
it('renders collapsible task board from runtime.todos', () => {
  render(<AssistantReply content="" runtime={{ ...baseRuntime, todos: [{ id: '1', content: 'check pod', status: 'pending' }] }} />);
  expect(screen.getByText('Task Board')).toBeInTheDocument();
  expect(screen.getByText('check pod')).toBeInTheDocument();
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx -t "task board"`
Expected: FAIL

- [ ] **Step 3: 最小实现 Task Board 卡片（顶部、可折叠）**

```tsx
{runtime?.todos?.length ? (
  <Collapse items={[{ key: 'todos', label: 'Task Board', children: <TodoList todos={runtime.todos} /> }]} />
) : null}
```

- [ ] **Step 4: 运行组件回归**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai-web): render task board from runtime todos"
```

### Task 5: 审批 `submitting` 超时转 `refresh-needed`，并提供刷新入口

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/ToolReference.tsx`
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: 写失败测试，模拟提交后 15 秒未确认进入 `refresh-needed`**

```ts
it('moves submitting approval to refresh-needed after 15s timeout', async () => {
  vi.useFakeTimers();
  // trigger approval submit
  vi.advanceTimersByTime(15000);
  expect(screen.getByText(/状态可能已更新，请刷新/i)).toBeInTheDocument();
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx -t "15s"`
Expected: FAIL

- [ ] **Step 3: 最小实现超时器与“刷新审批状态”按钮**

```tsx
const APPROVAL_SUBMITTING_TIMEOUT_MS = 15000;

setTimeout(() => setApprovalViewState(activity, {
  state: 'refresh-needed',
  message: '状态可能已更新，请刷新审批状态',
}), APPROVAL_SUBMITTING_TIMEOUT_MS);
```

- [ ] **Step 4: 运行审批相关回归测试**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx src/api/modules/ai.approval.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/ToolReference.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "fix(ai-web): fallback approval submitting state to refresh-needed"
```

## Chunk 3: Agent 指示与长内容性能护栏

### Task 6: Agent 活动标签防抖（300-500ms）

**Files:**
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: 写失败测试，断言高频 handoff 不会每次触发状态刷新**

```ts
it('debounces rapid agent handoff status updates', () => {
  // emit three handoffs within 100ms
  // expect one visible status update after debounce window
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts -t "debounce"`
Expected: FAIL

- [ ] **Step 3: 实现防抖状态更新（默认 400ms）**

```ts
const AGENT_STATUS_DEBOUNCE_MS = 400;
```

- [ ] **Step 4: 运行 provider 回归**

Run: `npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "perf(ai-web): debounce active agent status updates"
```

### Task 7: 长内容阈值外置查看 + 历史块视口挂载

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: 写失败测试，断言超阈值结果默认折叠并出现“查看原文”**

```ts
it('externalizes oversized content with view-full-content entry', () => {
  // content > 64KB
  expect(screen.getByText('查看原文')).toBeInTheDocument();
});
```

- [ ] **Step 2: 写失败测试，断言离开视口时回收重内容节点**

```ts
it('unmounts heavy block when leaving viewport', () => {
  // mock IntersectionObserver false -> expect placeholder rendered
});
```

- [ ] **Step 3: 运行测试确认失败**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx -t "查看原文|viewport"`
Expected: FAIL

- [ ] **Step 4: 最小实现阈值策略与 IntersectionObserver 挂载控制**

```tsx
const LARGE_CONTENT_LINE_THRESHOLD = 200;
const LARGE_CONTENT_BYTES_THRESHOLD = 64 * 1024;
```

- [ ] **Step 5: 运行组件回归**

Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "perf(ai-web): add heavy content externalization and viewport mounting"
```

## Final Verification

- [ ] **Step 1: 运行前端 AI 相关回归套件**

Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts src/api/modules/ai.test.ts src/components/AI/replyRuntime.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS

- [ ] **Step 2: 运行静态检查**

Run: `npm run lint`
Expected: PASS（无新增 lint 错误）

- [ ] **Step 3: 最终 Commit（若前序任务未拆 commit，则在此补齐）**

```bash
git add web/src/api/modules/ai.ts web/src/components/AI/types.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/AssistantReply.tsx web/src/components/AI/ToolReference.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/replyRuntime.test.ts web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts
git commit -m "feat(ai-web): complete deepagents frontend interaction migration"
```
