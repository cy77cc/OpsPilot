# AI 助手 Tool Result 引用展示优化

## 概述

优化 AI 助手回复中 `tool_result` 的显示样式，解决当前 tool_result 内容过长导致回复被占满的问题。

## 当前问题

- `tool_result` 统一显示在 executor 输出下方的 activities 区域
- 返回结果特别长时，整个回复基本被 tool_result 占满
- 用户难以快速获取关键信息

## 目标效果

1. executor 输出文本后，紧跟工具名称的引用样式
2. 点击引用弹出卡片显示完整结果（美化后的 JSON）
3. 根据内容长度自动选择 Popover 或 Drawer 展示方式

## 设计细节

### 1. 数据结构变更

#### 后端 (`internal/ai/runtime/types.go`)

```go
type PersistedActivity struct {
    ID         string         `json:"id"`
    Kind       string         `json:"kind"`
    Label      string         `json:"label"`
    Detail     string         `json:"detail,omitempty"`
    Status     string         `json:"status,omitempty"`
    StepIndex  int            `json:"stepIndex,omitempty"`
    Arguments  map[string]any `json:"arguments,omitempty"`  // 新增：工具调用参数
    RawContent string         `json:"rawContent,omitempty"` // 新增：完整结果内容
}
```

#### 前端 (`web/src/components/AI/types.ts`)

```typescript
export interface AssistantReplyActivity {
  id: string;
  kind: AssistantReplyActivityKind;
  label: string;
  detail?: string;
  status?: AssistantReplyActivityStatus;
  stepIndex?: number;
  arguments?: Record<string, unknown>;  // 新增：工具调用参数
  rawContent?: string;                  // 新增：完整结果内容
}
```

### 2. 前端组件设计

#### ToolReference.tsx - 工具引用组件

显示样式：
- 执行中：`[◐ tool_name]` （带动画）
- 完成：`[→ tool_name]` （可点击，蓝色）
- 错误：`[✗ tool_name]` （可点击，红色）

交互：
- 点击后打开 ToolResultCard 显示详情
- 多个工具调用时显示为 `[→ tool1] [→ tool2]`

#### ToolResultCard.tsx - 结果卡片组件

内容：
- 工具名称（标题）
- 调用参数（JSON 格式化，可折叠）
- 执行结果（JSON 格式化，语法高亮）

自适应展示：

| 条件 | 展示方式 | 样式 |
|------|---------|------|
| JSON 行数 ≤ 20 | Popover | 最大高度 300px，最大宽度 360px |
| JSON 行数 > 20 | Modal | 居中弹窗，宽度 600px，避免嵌套 Drawer 问题 |

> **注意**：不使用 Drawer 是因为 AI 助手本身已是 Drawer，嵌套会导致 z-index 冲突和多层遮罩问题。

```typescript
function getDisplayMode(content: string): 'popover' | 'modal' {
  // 预计算并缓存结果，避免每次渲染重复计算
  try {
    const parsed = JSON.parse(content);
    const formatted = JSON.stringify(parsed, null, 2);
    const lineCount = formatted.split('\n').length;
    return lineCount > 20 ? 'modal' : 'popover';
  } catch {
    // 非 JSON 内容：纯文本错误消息、Markdown 等
    const lineCount = content.split('\n').length;
    return lineCount > 20 ? 'modal' : 'popover';
  }
}
```

**非 JSON 内容处理**：
- 纯文本错误消息：直接显示原文
- Markdown 内容：使用 XMarkdown 渲染
- 大型二进制数据（base64）：截断显示，提示"内容过大"

#### AssistantReply.tsx 修改

1. 渲染 `activeStep.content` 后，检测关联的 `tool_call`/`tool_result` activities
2. 在文本末尾追加 `ToolReference` 组件
3. 过滤 activities 列表：不显示 `kind === 'tool_call' || kind === 'tool_result'` 的条目
4. 保留其他 activities：`agent_handoff`、`plan`、`replan`、`tool_approval`、`hint`、`error`

**工具引用与文本内容的关联**：

由于 SSE 事件顺序是 `delta -> tool_call -> tool_result`，LLM 通常在调用工具前会说一句话。渲染逻辑：

```typescript
// 获取当前步骤关联的工具 activities（tool_call 被更新为 tool_result 后）
const toolActivities = runtime?.activities?.filter(
  (a) => a.stepIndex === activeStepIndex && (a.kind === 'tool_result')
) || [];

// 渲染：步骤内容 + 工具引用
<div>
  <XMarkdown content={activeStep.content} />
  {toolActivities.map((tool) => (
    <ToolReference key={tool.id} activity={tool} />
  ))}
</div>
```

**Activity 状态转换说明**：

后端处理逻辑：当 `tool_result` 到达时，会**更新已有的 activity**（而非创建新条目）：

```go
// project.go 第 142-148 行
// 找到对应的 tool_call activity 并更新
for i := range state.Persisted.Activities {
    if state.Persisted.Activities[i].ID == event.Tool.CallID {
        state.Persisted.Activities[i].Status = status      // "done" 或 "error"
        state.Persisted.Activities[i].Kind = "tool_result" // kind 从 tool_call 变为 tool_result
        state.Persisted.Activities[i].RawContent = ...
    }
}
```

因此前端只需关注 `kind === 'tool_result'` 的 activities，它们已包含完整的状态信息。

### 3. 后端改动

#### project.go

处理 `tool_call` 事件时：
```go
state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
    ID:        event.Tool.CallID,
    Kind:      "tool_call",
    Label:     event.Tool.ToolName,
    Status:    "active",
    StepIndex: activeStepIndex,
    Arguments: event.Tool.Arguments,  // 存储参数
})
```

处理 `tool_result` 事件时：
```go
// 更新 activity，存储完整结果
state.Persisted.Activities[i].Status = status
state.Persisted.Activities[i].Kind = "tool_result"
state.Persisted.Activities[i].RawContent = event.Tool.Content  // 完整结果
state.Persisted.Activities[i].Detail = truncateString(event.Tool.Content, 200)  // 保留预览
```

### 4. 事件顺序与渲染逻辑

SSE 事件顺序：`delta -> tool_call -> tool_result`

渲染时机：
1. 收到 `tool_call` 时：显示 `[◐ tool_name]`（执行中状态）
2. 收到 `tool_result` 时：
   - 成功：更新为 `[→ tool_name]`
   - 失败：更新为 `[✗ tool_name]`（红色）

### 5. 错误处理

- 错误状态的引用显示为红色
- 点击后在卡片中显示错误详情
- 卡片中明确标注错误状态

### 6. 持久化考虑

`RawContent` 字段可能包含大型数据（如完整的 K8s 资源列表），会增加 `runtime_json` 的数据库存储大小。

**处理策略**：
- 正常存储完整内容（当前方案）
- 如果后续出现性能问题，可考虑：
  - 添加大小限制（如超过 10KB 截断）
  - 大型内容仅存储预览，完整内容按需从日志系统获取

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/ai/runtime/types.go` | 修改 | 添加 `Arguments`、`RawContent` 字段 |
| `internal/ai/runtime/project.go` | 修改 | 存储 arguments 和完整 content |
| `web/src/components/AI/types.ts` | 修改 | 添加 `arguments`、`rawContent` 字段 |
| `web/src/components/AI/replyRuntime.ts` | 修改 | 更新 `applyToolCall`、`applyToolResult` 存储 arguments 和 rawContent |
| `web/src/components/AI/ToolReference.tsx` | 新增 | 工具引用组件（执行中/完成/错误状态） |
| `web/src/components/AI/ToolResultCard.tsx` | 新增 | 结果卡片组件（Popover/Modal 自适应） |
| `web/src/components/AI/AssistantReply.tsx` | 修改 | 集成引用渲染，过滤 tool_call/tool_result activities |

## 实现优先级

1. 后端数据结构变更（types.go、project.go）
2. 前端类型变更（types.ts、replyRuntime.ts）
3. ToolReference 组件实现
4. ToolResultCard 组件实现
5. AssistantReply 集成
6. 测试与调优
