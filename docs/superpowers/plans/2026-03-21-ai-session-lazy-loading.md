# AI 会话懒加载与滚动优化 实现计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 AI 会话的懒加载和滚动优化，包括 Reply 懒加载、Steps 二级折叠、简化滚动跟踪。

**Architecture:**
1. 数据层：改造 `historyProjection.ts`，新增轻量级 runtime 转换和 step 内容懒加载函数
2. 组件层：改造 `AssistantReply.tsx`，实现 step 独立折叠和懒加载触发
3. 滚动层：改造 `CopilotSurface.tsx`，移除锚点定位，简化滚动跟踪

**Tech Stack:** React, TypeScript, Ant Design, @ant-design/x-sdk

**Design Spec:** `docs/superpowers/specs/2026-03-21-ai-session-lazy-loading-design.md`

---

## Chunk 1: 数据层改造

### Task 1: 类型定义更新

**Files:**
- Modify: `web/src/components/AI/types.ts`

- [ ] **Step 1: 添加 SlimExecutorBlock 类型**

在 `web/src/components/AI/types.ts` 中，添加瘦身后的 executor block 类型：

```typescript
// 瘦身后的 executor block（减少内存占用，只保留懒加载必须字段）
export interface SlimExecutorBlock {
  id: string;
  items: Array<{
    type: string;
    content_id?: string;
    tool_call_id?: string;
    tool_name?: string;
    arguments?: Record<string, unknown>;
    result?: {
      status?: string;
      preview?: string;
      result_content_id?: string;
    };
  }>;
}
```

- [ ] **Step 2: 更新 AssistantReplyPlanStep 类型**

为 `AssistantReplyPlanStep` 添加 `loaded` 字段：

```typescript
export interface AssistantReplyPlanStep {
  id: string;
  title: string;
  status: 'pending' | 'active' | 'done';
  content?: string;
  segments?: AssistantReplySegment[];
  loaded?: boolean;  // 新增：标记内容是否已加载
}
```

- [ ] **Step 3: 更新 AssistantReplyRuntime 类型**

为 `AssistantReplyRuntime` 添加 `_executorBlocks` 字段：

```typescript
export interface AssistantReplyRuntime {
  phase?: AssistantReplyPhase;
  phaseLabel?: string;
  activities: AssistantReplyActivity[];
  plan?: AssistantReplyPlan;
  summary?: AssistantReplySummary;
  status?: AssistantReplyRuntimeStatus;
  _executorBlocks?: SlimExecutorBlock[];  // 新增：存储瘦身后的 executor blocks 引用
}
```

- [ ] **Step 4: 运行类型检查**

Run: `cd web && npm run type-check 2>&1 | head -50`
Expected: 可能有类型错误（因为 `_executorBlocks` 未使用），确保无语法错误

- [ ] **Step 5: 提交**

```bash
git add web/src/components/AI/types.ts
git commit -m "feat(ai): add lazy loading types with SlimExecutorBlock"
```

---

### Task 2: 实现轻量级 runtime 转换

**Files:**
- Modify: `web/src/components/AI/historyProjection.ts`

- [ ] **Step 1: 添加 projectionToLazyRuntime 函数**

在 `web/src/components/AI/historyProjection.ts` 中，添加新函数。注意：对 executor blocks 进行瘦身处理，减少内存占用：

```typescript
import type { SlimExecutorBlock } from './types';

/**
 * projectionToLazyRuntime 将 projection 转换为轻量级 runtime。
 * 只提取 steps 标题和 summary，不加载 executor 内容。
 * executor blocks 进行瘦身存储，只保留懒加载必须的字段。
 */
function projectionToLazyRuntime(projection: AIRunProjection): AssistantReplyRuntime {
  const steps: AssistantReplyPlanStep[] = [];

  // 从 plan/replan blocks 提取步骤标题
  for (const block of projection.blocks) {
    if (block.type === 'plan' || block.type === 'replan') {
      block.steps?.forEach((title) => {
        steps.push({
          id: `step-${steps.length}`,
          title,
          status: 'done',
          loaded: false,
        });
      });
    }
  }

  // executor blocks 瘦身存储，只保留懒加载必须的字段
  const executorBlocks: SlimExecutorBlock[] = projection.blocks
    .filter(b => b.type === 'executor')
    .map(block => ({
      id: block.id,
      items: (block.items || []).map(item => ({
        type: item.type,
        content_id: item.content_id,
        tool_call_id: item.tool_call_id,
        tool_name: item.tool_name,
        arguments: item.arguments,
        result: item.result ? {
          status: item.result.status,
          preview: item.result.preview,
          result_content_id: item.result.result_content_id,
        } : undefined,
      })),
    }));

  return {
    activities: [],
    plan: steps.length > 0 ? { steps } : undefined,
    summary: projection.summary?.title ? { title: projection.summary.title } : undefined,
    status: {
      kind: projection.status === 'failed_runtime' ? 'error' : 'completed',
      label: projection.status,
    },
    _executorBlocks: executorBlocks,
  };
}
```

- [ ] **Step 2: 运行类型检查**

Run: `cd web && npm run type-check 2>&1 | head -50`
Expected: 类型检查通过

- [ ] **Step 3: 提交**

```bash
git add web/src/components/AI/historyProjection.ts
git commit -m "feat(ai): add projectionToLazyRuntime with slimmed executor blocks"
```

---

### Task 3: 实现 step 内容懒加载函数

**Files:**
- Modify: `web/src/components/AI/historyProjection.ts`

- [ ] **Step 1: 添加 loadStepContent 函数**

在 `projectionToLazyRuntime` 函数之后添加：

```typescript
/**
 * loadStepContent 加载单个 step 的内容。
 * 根据 executor block 的 items 加载文本内容和工具调用信息。
 * 同时构建该 step 对应的 activities。
 */
export async function loadStepContent(
  block: AIRunProjectionBlock,
  stepIndex: number,
): Promise<{
  content: string;
  segments: AssistantReplySegment[];
  activities: AssistantReplyActivity[];
}> {
  const segments: AssistantReplySegment[] = [];
  const activities: AssistantReplyActivity[] = [];
  let content = '';

  for (const item of block.items || []) {
    if (item.type === 'content' && item.content_id) {
      const runContent = await loadRunContent(item.content_id);
      const text = normalizeMarkdownContent(runContent?.body_text || '');
      if (text) {
        segments.push({ type: 'text', text });
        content += text;
      }
    }
    if (item.type === 'tool_call' && item.tool_call_id && item.tool_name) {
      const resultContent = item.result?.result_content_id
        ? await loadRunContent(item.result.result_content_id)
        : null;
      const rawContent = resultContent?.body_text || item.result?.preview;

      activities.push({
        id: item.tool_call_id,
        kind: 'tool',
        label: item.tool_name,
        detail: item.result ? item.result.preview : INTERRUPTED_TOOL_MESSAGE,
        rawContent,
        status: item.result
          ? item.result.status === 'done' ? 'done' : 'error'
          : 'error',
        stepIndex,
        arguments: item.arguments,
      });

      segments.push({ type: 'tool_ref', callId: item.tool_call_id });
    }
  }

  return { content, segments, activities };
}
```

- [ ] **Step 2: 运行类型检查**

Run: `cd web && npm run type-check 2>&1 | head -50`
Expected: 类型检查通过

- [ ] **Step 3: 提交**

```bash
git add web/src/components/AI/historyProjection.ts
git commit -m "feat(ai): add loadStepContent for lazy step loading"
```

---

### Task 4: 修改 hydrateAssistantHistoryFromProjection

**Files:**
- Modify: `web/src/components/AI/historyProjection.ts`

- [ ] **Step 1: 修改 hydrateAssistantHistoryFromProjection 使用懒加载**

将 `hydrateAssistantHistoryFromProjection` 函数中调用 `projectionToRuntime` 的地方改为 `projectionToLazyRuntime`：

找到这段代码：
```typescript
  const runtime = await projectionToRuntime(projection);
```

替换为：
```typescript
  const runtime = projectionToLazyRuntime(projection);
```

完整修改后的函数：

```typescript
export async function hydrateAssistantHistoryFromProjection(
  message: AIMessage,
): Promise<XChatMessage> {
  const fallbackContent = message.content || '';
  if (message.role !== 'assistant') {
    return {
      id: message.id,
      role: 'user',
      content: fallbackContent,
    };
  }

  const runId = (message as AIMessage & { run_id?: string }).run_id;
  if (!runId) {
    return {
      id: message.id,
      role: 'assistant',
      content: fallbackContent,
    };
  }

  const projection = await loadRunProjection(runId);
  if (!projection) {
    if (fallbackContent.trim()) {
      return {
        id: message.id,
        role: 'assistant',
        content: fallbackContent,
        runtime: {
          activities: [],
          status: {
            kind: 'error',
            label: message.error_message || '生成中断，请稍后重试',
          },
        },
      };
    }
    return {
      id: message.id,
      role: 'assistant',
      content: PROJECTION_UNRECOVERABLE_PLACEHOLDER,
      runtime: {
        activities: [],
        status: {
          kind: 'error',
          label: PROJECTION_MISSING_SUMMARY_LABEL,
        },
      },
    };
  }

  const summaryContent = normalizeMarkdownContent(projection.summary?.content || '').trim();
  if (!summaryContent) {
    return {
      id: message.id,
      role: 'assistant',
      content: PROJECTION_UNRECOVERABLE_PLACEHOLDER,
      runtime: {
        activities: [],
        status: {
          kind: 'error',
          label: PROJECTION_MISSING_SUMMARY_LABEL,
        },
      },
    };
  }

  // 使用轻量级 runtime 转换，不加载 executor 内容
  const runtime = projectionToLazyRuntime(projection);
  return {
    id: message.id,
    role: 'assistant',
    content: summaryContent,
    runtime,
  };
}
```

- [ ] **Step 2: 移除旧的 projectionToRuntime 函数**

如果 `projectionToRuntime` 不再被其他地方使用，可以移除或注释掉。先检查是否有其他引用：

Run: `grep -n "projectionToRuntime" web/src/components/AI/historyProjection.ts`
Expected: 只在定义处出现

如果确认无其他引用，删除 `projectionToRuntime` 函数（约 80 行代码）。

- [ ] **Step 3: 运行类型检查**

Run: `cd web && npm run type-check 2>&1 | head -50`
Expected: 类型检查通过

- [ ] **Step 4: 提交**

```bash
git add web/src/components/AI/historyProjection.ts
git commit -m "refactor(ai): use lazy runtime in hydrateAssistantHistoryFromProjection"
```

---

## Chunk 2: 组件层改造

### Task 5: 添加 step 展开状态管理

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`

- [ ] **Step 1: 添加 onLoadStepContent prop**

修改 `AssistantReplyProps` 接口：

```typescript
interface AssistantReplyProps {
  content: string;
  runtime?: AssistantReplyRuntime;
  status?: string;
  messageId?: string;
  hasRuntime?: boolean;
  onLoadRuntime?: (messageId: string) => Promise<AssistantReplyRuntime | null>;
  onLoadStepContent?: (
    stepId: string,
    stepIndex: number,
  ) => Promise<{
    content: string;
    segments: AssistantReplySegment[];
    activities: AssistantReplyActivity[];
  } | null>;
}
```

- [ ] **Step 2: 添加完整的状态机管理**

在 `AssistantReplyContent` 组件中添加状态管理。采用局部状态合并方案确保 React 响应性：

```typescript
// Step 加载状态：完整的状态机
type StepLoadState = 'idle' | 'loading' | 'success' | 'error';

function AssistantReplyContent({
  content,
  runtime,
  status,
  styles,
  onLoadStepContent,
}: {
  content: string;
  runtime: AssistantReplyRuntime;
  status?: string;
  styles: Record<string, string>;
  onLoadStepContent?: AssistantReplyProps['onLoadStepContent'];
}) {
  // 展开状态
  const [stepExpandStates, setStepExpandStates] = useState<Record<string, boolean>>({});
  // 加载状态：状态机
  const [stepLoadStates, setStepLoadStates] = useState<Record<string, StepLoadState>>({});
  // 内容缓存：异步加载的数据存储于此
  const [stepContentCache, setStepContentCache] = useState<Record<string, {
    content: string;
    segments: AssistantReplySegment[];
    activities: AssistantReplyActivity[];
  } | null>>({});

  // ... 其余代码
}
```

- [ ] **Step 3: 添加 handleStepExpand 处理函数（含竞态条件处理）**

```typescript
const handleStepExpand = async (stepId: string, stepIndex: number) => {
  // 1. 状态检查：防止重复请求（竞态条件处理）
  const currentState = stepLoadStates[stepId] || 'idle';
  if (currentState === 'loading' || currentState === 'success') {
    // 已在加载或已成功，直接展开
    if (currentState === 'success') {
      setStepExpandStates(prev => ({ ...prev, [stepId]: true }));
    }
    return;
  }

  // 2. 检查是否已有缓存
  if (stepContentCache[stepId]) {
    setStepExpandStates(prev => ({ ...prev, [stepId]: true }));
    return;
  }

  if (!onLoadStepContent) {
    return;
  }

  // 3. 设置 loading 状态
  setStepLoadStates(prev => ({ ...prev, [stepId]: 'loading' }));

  try {
    const result = await onLoadStepContent(stepId, stepIndex);
    if (result) {
      // 4. 成功：存储内容并标记成功
      setStepContentCache(prev => ({ ...prev, [stepId]: result }));
      setStepLoadStates(prev => ({ ...prev, [stepId]: 'success' }));
      setStepExpandStates(prev => ({ ...prev, [stepId]: true }));
    } else {
      // 5. 返回 null：标记错误
      setStepLoadStates(prev => ({ ...prev, [stepId]: 'error' }));
    }
  } catch (error) {
    // 6. 异常：标记错误
    setStepLoadStates(prev => ({ ...prev, [stepId]: 'error' }));
  }
};

// 重试函数
const handleRetry = (stepId: string, stepIndex: number) => {
  // 重置状态后重新触发加载
  setStepLoadStates(prev => ({ ...prev, [stepId]: 'idle' }));
  handleStepExpand(stepId, stepIndex);
};
```

- [ ] **Step 4: 运行类型检查**

Run: `cd web && npm run type-check 2>&1 | head -50`
Expected: 类型检查通过

- [ ] **Step 5: 提交**

```bash
git add web/src/components/AI/AssistantReply.tsx
git commit -m "feat(ai): add step state machine with error handling"
```

---

### Task 6: 改造已完成步骤的渲染（含错误处理）

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`

- [ ] **Step 1: 修改 completedSteps 的渲染方式**

将现有的 `Collapse` 组件改为每个 step 独立的折叠，包含错误状态处理：

找到这段代码：
```typescript
      {completedSteps.length > 0 ? (
        <Collapse
          className={styles.completedStepsCollapse}
          ghost
          items={[{
            key: 'completed',
            label: `已完成 ${completedSteps.length} 个步骤`,
            children: completedSteps.map((step) => (
              <div key={step.id} className={styles.completedStepItem}>
                <div className={styles.completedStepTitle}>
                  <span>✓</span>
                  <span>{step.title}</span>
                </div>
                <StepContentRenderer
                  step={step}
                  activities={runtime?.activities || []}
                  isStreaming={false}
                  styles={styles}
                />
              </div>
            )),
          }]}
        />
      ) : null}
```

替换为：
```typescript
      {completedSteps.length > 0 ? (
        <Collapse
          className={styles.completedStepsCollapse}
          ghost
          items={completedSteps.map((step, index) => {
            const loadState = stepLoadStates[step.id] || 'idle';
            const isExpanded = stepExpandStates[step.id];
            const cachedContent = stepContentCache[step.id];
            const displayStep = cachedContent
              ? { ...step, content: cachedContent.content, segments: cachedContent.segments, loaded: true }
              : step;

            return {
              key: step.id,
              label: (
                <div className={styles.completedStepTitle}>
                  <span>✓</span>
                  <span>{step.title}</span>
                </div>
              ),
              children: loadState === 'loading' ? (
                <div className={styles.loadingContainer}>
                  <Skeleton active paragraph={{ rows: 2 }} />
                </div>
              ) : loadState === 'error' ? (
                <div className={styles.errorContainer}>
                  <span style={{ color: token.colorError }}>加载失败</span>
                  <Button
                    type="link"
                    size="small"
                    onClick={() => handleRetry(step.id, index)}
                  >
                    重试
                  </Button>
                </div>
              ) : (
                <StepContentRenderer
                  step={displayStep}
                  activities={cachedContent?.activities || []}
                  isStreaming={false}
                  styles={styles}
                />
              ),
            };
          })}
          activeKey={Object.keys(stepExpandStates).filter((k) => stepExpandStates[k])}
          onChange={(keys) => {
            const newExpanded = Array.isArray(keys) ? keys : [keys];
            const prevExpanded = Object.keys(stepExpandStates).filter((k) => stepExpandStates[k]);

            // 找到新展开的 step
            const newlyExpanded = newExpanded.filter((k) => !prevExpanded.includes(k));
            newlyExpanded.forEach((stepId) => {
              const stepIndex = completedSteps.findIndex((s) => s.id === stepId);
              if (stepIndex >= 0) {
                handleStepExpand(stepId, stepIndex);
              }
            });
          }}
        />
      ) : null}
```

注意：需要从 `antd-style` 的 `useStyles` 中获取 `token` 用于错误提示的颜色。

- [ ] **Step 2: 运行类型检查**

Run: `cd web && npm run type-check 2>&1 | head -50`
Expected: 类型检查通过

- [ ] **Step 3: 提交**

```bash
git add web/src/components/AI/AssistantReply.tsx
git commit -m "feat(ai): implement per-step collapse with lazy loading"
```

---

### Task 7: 传递 onLoadStepContent 回调

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/CopilotSurface.tsx`

- [ ] **Step 1: 修改 AssistantReply 主组件**

在 `AssistantReply` 组件中传递 `onLoadStepContent`：

```typescript
export function AssistantReply({
  content,
  runtime,
  status,
  messageId,
  hasRuntime,
  onLoadRuntime,
  onLoadStepContent,
}: AssistantReplyProps) {
  // ... 现有代码

  // 有 runtime 时直接显示完整内容
  if (displayRuntime) {
    return (
      <div className={styles.root}>
        <AssistantReplyContent
          content={content}
          runtime={displayRuntime}
          status={status}
          styles={styles}
          onLoadStepContent={onLoadStepContent}
        />
      </div>
    );
  }

  // ... 其余代码
}
```

- [ ] **Step 2: 在 CopilotSurface 中创建 loadStepContent 回调**

在 `CopilotSurface.tsx` 中，找到 `bubbleRole` 的定义位置，添加：

```typescript
const handleLoadStepContent = React.useCallback(
  async (stepId: string, stepIndex: number) => {
    // 从当前 assistant 消息获取 runtime
    const currentMsg = currentAssistantMessage?.message;
    if (!currentMsg?.runtime?._executorBlocks) {
      return null;
    }

    const executorBlocks = currentMsg.runtime._executorBlocks;
    if (stepIndex < 0 || stepIndex >= executorBlocks.length) {
      return null;
    }

    const block = executorBlocks[stepIndex];
    if (!block) {
      return null;
    }

    return loadStepContent(block, stepIndex);
  },
  [currentAssistantMessage],
);
```

需要添加 import：
```typescript
import { loadStepContent } from './historyProjection';
```

- [ ] **Step 3: 传递回调到 AssistantReply**

找到 `bubbleRole` 中的 `contentRender`，修改为：

```typescript
contentRender: (content: string, info) => (
  <div data-message-anchor={(item as any).extraInfo?.messageId}>
    <AssistantReply
      content={content}
      runtime={(info as any).extraInfo?.runtime}
      status={info.status}
      onLoadStepContent={handleLoadStepContent}
    />
  </div>
),
```

- [ ] **Step 4: 运行类型检查**

Run: `cd web && npm run type-check 2>&1 | head -50`
Expected: 类型检查通过

- [ ] **Step 5: 提交**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/CopilotSurface.tsx
git commit -m "feat(ai): wire up step content lazy loading"
```

---

## Chunk 3: 滚动优化

### Task 8: 移除锚点定位逻辑

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`

- [ ] **Step 1: 移除 getMessageAnchor 函数**

找到并删除以下代码：

```typescript
const getMessageAnchor = React.useCallback((messageId?: string) => {
  const el = contentRef.current;
  if (!el || !messageId) {
    return null;
  }
  return el.querySelector<HTMLElement>(`[data-message-anchor="${messageId}"]`);
}, []);
```

- [ ] **Step 2: 移除锚点定位的 useEffect**

找到并删除以下 useEffect（约 25 行）：

```typescript
React.useEffect(() => {
  if (!open || !currentAssistantMessage?.renderKey || followStateRef.current !== 'following') {
    return;
  }

  const frame = requestAnimationFrame(() => {
    const el = contentRef.current;
    const anchor = getMessageAnchor(currentAssistantMessage.renderKey);
    if (!el || !anchor) {
      return;
    }
    const nextTop = Math.max(
      0,
      anchor.offsetTop + anchor.offsetHeight - el.clientHeight + FOLLOW_BOTTOM_SAFE_GAP,
    );
    withProgrammaticScroll(() => {
      el.scrollTo({ top: nextTop, behavior: 'auto' });
    });
  });

  return () => cancelAnimationFrame(frame);
}, [
  currentAssistantMessage?.renderKey,
  currentAssistantMessage?.message.content,
  currentAssistantMessage?.message.runtime,
  currentAssistantMessage?.status,
  getMessageAnchor,
  open,
  withProgrammaticScroll,
]);
```

- [ ] **Step 3: 移除不再使用的常量**

找到并删除：

```typescript
const FOLLOW_BOTTOM_SAFE_GAP = 72;
```

- [ ] **Step 4: 运行类型检查**

Run: `cd web && npm run type-check 2>&1 | head -50`
Expected: 类型检查通过

- [ ] **Step 5: 提交**

```bash
git add web/src/components/AI/CopilotSurface.tsx
git commit -m "refactor(ai): remove anchor positioning logic"
```

---

### Task 9: 使用 ResizeObserver 简化滚动跟踪

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`

- [ ] **Step 1: 使用 ResizeObserver 处理动态内容滚动**

流式输出过程中，Markdown 中的图片、代码块等异步加载会导致 DOM 高度变化。使用 ResizeObserver 确保滚动跟随：

找到初始滚动 useEffect，替换为 ResizeObserver 版本：

```typescript
// 初始化滚动 + 流式响应滚动（使用 ResizeObserver）
React.useEffect(() => {
  if (!open) return;

  const el = contentRef.current;
  if (!el) return;

  // 滚动到底部的辅助函数
  const scrollToBottom = () => {
    if (followStateRef.current !== 'following') return;
    withProgrammaticScroll(() => {
      el.scrollTo({ top: el.scrollHeight, behavior: 'auto' });
    });
  };

  // 创建 ResizeObserver 监听内容区域高度变化
  const resizeObserver = new ResizeObserver(() => {
    if (followStateRef.current === 'following') {
      scrollToBottom();
    }
  });

  // 观察内容容器
  resizeObserver.observe(el);

  // 初始滚动
  scrollToBottom();

  return () => {
    resizeObserver.disconnect();
  };
}, [open, withProgrammaticScroll]);
```

**优点：**
- 自动处理图片加载、代码块渲染等导致的布局变化
- 流式输出时平滑跟随
- 用户上滑后停止跟随（通过 `followStateRef` 控制）

- [ ] **Step 2: 移除 withProgrammaticScroll（如果不再需要）**

检查 `withProgrammaticScroll` 是否还有其他使用。如果只有滚动跟踪使用，可以移除：

Run: `grep -n "withProgrammaticScroll" web/src/components/AI/CopilotSurface.tsx`

如果只有定义和上面那个 useEffect 使用，可以移除整个函数定义。

- [ ] **Step 3: 运行类型检查**

Run: `cd web && npm run type-check 2>&1 | head -50`
Expected: 类型检查通过

- [ ] **Step 4: 提交**

```bash
git add web/src/components/AI/CopilotSurface.tsx
git commit -m "refactor(ai): use ResizeObserver for scroll tracking"
```

---

## Chunk 4: 测试与验证

### Task 10: 端到端测试

**Files:**
- None (manual testing)

- [ ] **Step 1: 启动开发服务器**

Run: `make dev-backend & make dev-frontend &`

- [ ] **Step 2: 测试懒加载功能**

1. 打开 AI 助手面板
2. 切换到有历史消息的会话
3. 验证：只显示 summary 和步骤标题，无内容
4. 点击展开某个步骤
5. 验证：内容正确加载并显示

- [ ] **Step 3: 测试滚动行为**

1. 开始新的对话
2. 发送消息，观察流式响应
3. 验证：自动滚动到底部
4. 手动向上滚动
5. 继续对话，验证：不自动滚动
6. 滚回底部
7. 继续对话，验证：恢复自动滚动

- [ ] **Step 4: 提交最终版本**

```bash
git add -A
git commit -m "feat(ai): complete lazy loading and scroll optimization"
```

---

## 改动文件清单

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `web/src/components/AI/types.ts` | 修改 | 添加 `loaded` 和 `_executorBlocks` 字段 |
| `web/src/components/AI/historyProjection.ts` | 修改 | 新增 `projectionToLazyRuntime`、`loadStepContent`，修改 `hydrateAssistantHistoryFromProjection` |
| `web/src/components/AI/AssistantReply.tsx` | 修改 | 添加 step 独立折叠状态管理，改造 completedSteps 渲染 |
| `web/src/components/AI/CopilotSurface.tsx` | 修改 | 移除锚点定位，简化滚动跟踪，传递懒加载回调 |
