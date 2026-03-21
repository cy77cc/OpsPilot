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
  _executorBlocks?: AIRunProjectionBlock[];  // 新增：存储 executor blocks 引用，供懒加载使用
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
/**
 * loadStepContent 加载单个 step 的内容。
 * 根据 executor block 的 items 加载文本内容和工具调用信息。
 */
export async function loadStepContent(
  block: AIRunProjectionBlock,
  activities: AssistantReplyActivity[],
  stepIndex: number,
): Promise<{ content: string; segments: AssistantReplySegment[] }> {
  const segments: AssistantReplySegment[] = [];
  let content = '';

  const activityMap = new Map<string, AssistantReplyActivity>();
  activities.forEach(a => activityMap.set(a.id, a));

  for (const item of block.items || []) {
    if (item.type === 'content' && item.content_id) {
      const runContent = await loadRunContent(item.content_id);
      const text = normalizeMarkdownContent(runContent?.body_text || '');
      if (text) {
        segments.push({ type: 'text', text });
        content += text;
      }
    }
    if (item.type === 'tool_call' && item.tool_call_id) {
      segments.push({ type: 'tool_ref', callId: item.tool_call_id });
    }
  }

  return { content, segments };
}
```

### 2. Steps 二级折叠并懒加载

**改造 AssistantReply.tsx**

新增状态管理：

```typescript
interface StepExpandState {
  [stepId: string]: boolean;
}

interface StepLoadingState {
  [stepId: string]: boolean;
}
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
  onLoadStepContent?: (stepId: string, stepIndex: number) => Promise<{ content: string; segments: AssistantReplySegment[] } | null>;  // 新增
}
```

组件改造要点：

1. 每个 step 使用独立的 Collapse 控制展开状态
2. 展开时检查 `step.loaded`，如果未加载则调用 `onLoadStepContent`
3. 加载中显示 Skeleton
4. 加载完成后更新 step 的 content 和 segments

### 3. 固定显示最新响应

**改造 CopilotSurface.tsx**

移除以下代码：

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
```

简化滚动跟踪：

```typescript
// 简化：流式响应时滚动到底部
React.useEffect(() => {
  if (!open || followStateRef.current !== 'following') return;

  const el = contentRef.current;
  if (!el || !isRequesting) return;

  requestAnimationFrame(() => {
    el.scrollTo({ top: el.scrollHeight, behavior: 'auto' });
  });
}, [messages, isRequesting, open]);
```

保留的滚动相关逻辑：

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
