# AI 会话懒加载与滚动优化设计

**日期**: 2026-03-21
**状态**: 草稿
**作者**: Claude

## 背景

当前 AI 会话存在以下问题：

1. **Reply 加载阻塞** - 加载历史会话时，`hydrateAssistantHistoryFromProjection` 会遍历所有 executor blocks 并同步加载每个 `content_id`，导致渲染阻塞
2. **Steps 一次性渲染** - 已完成的步骤在 Collapse 中一次性渲染所有内容，不适合内容较多的场景
3. **锚点定位体验差** - 现有的 `getMessageAnchor` 定位逻辑复杂，滚动行为难以预测

## 目标

1. **Reply 懒加载** - 先显示 summary.content 和 steps 标题，内容按需加载
2. **Steps 二级折叠** - 每个 step 独立折叠，展开时才加载对应的 executor 内容
3. **简化滚动跟踪** - 移除锚点定位，采用"固定底部显示"模式，用户上滑时取消跟踪，回到底部时恢复

## 约束

- **不允许改变现有 UI 样式** - 只改逻辑，不动样式

## 技术方案

### 数据结构变更

**types.ts**

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

export interface AssistantReplyPlanStep {
  id: string;
  title: string;
  status: 'pending' | 'active' | 'done';
  content?: string;
  segments?: AssistantReplySegment[];
  loaded?: boolean;  // 新增：标记内容是否已加载
}

export interface AssistantReplyRuntime {
  // ... 现有字段
  _executorBlocks?: SlimExecutorBlock[];  // 新增：存储瘦身后的 executor blocks 引用
}
```

### 1. Reply 懒加载

**改造 historyProjection.ts**

当前流程：
```
hydrateAssistantHistoryFromProjection
  → loadRunProjection (阻塞)
  → projectionToRuntime (遍历所有 executor)
  → loadRunContent (每个 content_id，阻塞)
  → 返回完整 XChatMessage
```

改造后流程：
```
hydrateAssistantHistoryFromProjection (轻量)
  → loadRunProjection (阻塞)
  → projectionToLazyRuntime (只提取标题)
  → 返回 XChatMessage (steps 有标题，无内容)

用户展开 Step 时:
  onLoadStepContent(stepId, stepIndex)
  → 从 _executorBlocks 获取对应 block
  → loadRunContent(content_id)
  → 更新 runtime.steps[i].content
```

**新增函数：**

```typescript
/**
 * projectionToLazyRuntime 将 projection 转换为轻量级 runtime。
 * 只提取 steps 标题和 summary，不加载 executor 内容。
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

  // executor blocks 存储，供后续懒加载
  const executorBlocks = projection.blocks.filter(b => b.type === 'executor');

  return {
    activities: [],
    plan: steps.length > 0 ? { steps } : undefined,
    summary: projection.summary?.title ? { title: projection.summary.title } : undefined,
    status: { kind: 'completed', label: projection.status },
    _executorBlocks: executorBlocks,
  };
}
```

**修改 hydrateAssistantHistoryFromProjection：**

```typescript
export async function hydrateAssistantHistoryFromProjection(
  message: AIMessage,
): Promise<XChatMessage> {
  // ... 前置检查逻辑不变 ...

  const projection = await loadRunProjection(runId);
  if (!projection) {
    // ... 错误处理不变 ...
  }

  const summaryContent = normalizeMarkdownContent(projection.summary?.content || '').trim();
  if (!summaryContent) {
    // ... 错误处理不变 ...
  }

  // 改为调用轻量级转换
  const runtime = projectionToLazyRuntime(projection);

  return {
    id: message.id,
    role: 'assistant',
    content: summaryContent,
    runtime,
  };
}
```

**新增 step 内容加载函数：**

```typescript
const INTERRUPTED_TOOL_MESSAGE = '执行未完成';

// 瘦身后的 executor block 引用（减少内存占用）
interface SlimExecutorBlock {
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

/**
 * projectionToLazyRuntime 将 projection 转换为轻量级 runtime。
 * 只提取 steps 标题和 summary，不加载 executor 内容。
 * 用于历史消息的懒加载场景。
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
  const executorBlocks = projection.blocks
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
    status: { kind: 'completed', label: projection.status },
    _executorBlocks: executorBlocks,
  };
}

/**
 * loadStepContent 加载单个 step 的内容。
 * 根据 executor block 的 items 加载文本内容和工具调用信息。
 * 同时构建该 step 对应的 activities。
 */
export async function loadStepContent(
  block: SlimExecutorBlock,
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

### 2. Steps 二级折叠并懒加载

**改造 AssistantReply.tsx**

#### 2.1 状态管理（响应性问题）

由于 `runtime` 作为 props 从上层传入，直接修改其属性不会触发 React 重渲染。采用**局部状态合并**方案：

```typescript
// Step 加载状态：完整的状态机
type StepLoadState = 'idle' | 'loading' | 'success' | 'error';

interface StepLoadStates {
  [stepId: string]: StepLoadState;
}

// Step 内容缓存：异步加载的数据存储于此
interface StepContentCache {
  [stepId: string]: {
    content: string;
    segments: AssistantReplySegment[];
    activities: AssistantReplyActivity[];
  } | null;
}

// Step 展开状态
interface StepExpandStates {
  [stepId: string]: boolean;
}
```

#### 2.2 错误处理与竞态条件

```typescript
const handleStepExpand = async (stepId: string, stepIndex: number) => {
  // 1. 状态检查：防止重复请求
  const currentState = stepLoadStates[stepId];
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

#### 2.3 渲染逻辑

```typescript
const renderCompletedStep = (step: AssistantReplyPlanStep, index: number) => {
  const loadState = stepLoadStates[step.id] || 'idle';
  const isExpanded = stepExpandStates[step.id];
  const cachedContent = stepContentCache[step.id];

  // 合并 props.runtime 和局部缓存
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
        <span>加载失败</span>
        <Button type="link" onClick={() => handleRetry(step.id, index)}>重试</Button>
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
};
```

修改 AssistantReplyProps：

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
    stepIndex: number
  ) => Promise<{
    content: string;
    segments: AssistantReplySegment[];
    activities: AssistantReplyActivity[];
  } | null>;  // 新增：加载 step 内容的回调
}
```

### 3. 固定显示最新响应

**改造 CopilotSurface.tsx**

#### 3.1 移除锚点定位逻辑

```typescript
// 删除
const getMessageAnchor = React.useCallback((messageId?: string) => {
  // ...
}, []);

// 删除锚点定位 useEffect
React.useEffect(() => {
  if (!open || !currentAssistantMessage?.renderKey || followStateRef.current !== 'following') {
    return;
  }
  // ... 锚点定位逻辑
}, [currentAssistantMessage?.renderKey, ...]);

// 删除
const FOLLOW_BOTTOM_SAFE_GAP = 72;
```

#### 3.2 使用 ResizeObserver 处理动态内容

流式输出过程中，Markdown 中的图片、代码块等异步加载会导致 DOM 高度变化。使用 ResizeObserver 确保滚动跟随：

```typescript
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

#### 3.3 保留的滚动相关逻辑

- `followStateRef` - 'following' | 'detached' 状态
- `showScrollBottomBtn` - 滚动按钮显示状态
- `handleScrollToBottom` - 点击按钮滚动到底部
- 用户手动上滑时切换为 'detached'
- 滚回底部时切换为 'following'

## 改动文件清单

| 文件 | 改动内容 |
|------|----------|
| `web/src/components/AI/types.ts` | `AssistantReplyPlanStep` 新增 `loaded` 字段，`AssistantReplyRuntime` 新增 `_executorBlocks` |
| `web/src/components/AI/historyProjection.ts` | 新增 `projectionToLazyRuntime`、`loadStepContent`，修改 `hydrateAssistantHistoryFromProjection` |
| `web/src/components/AI/AssistantReply.tsx` | 新增 step 独立折叠状态管理，展开时触发懒加载 |
| `web/src/components/AI/CopilotSurface.tsx` | 移除锚点定位逻辑，简化滚动跟踪 |

## 测试要点

1. **懒加载正确性**
   - 历史会话加载时只显示 summary 和 steps 标题
   - 点击展开 step 时正确加载内容
   - 缓存生效，重复展开不重复请求

2. **滚动行为**
   - 流式响应时自动滚动到底部
   - 用户手动上滑后停止自动滚动
   - 点击按钮或滚回底部后恢复自动滚动

3. **UI 样式不变**
   - 所有现有样式保持不变
   - 只改变加载和交互逻辑
