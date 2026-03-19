# Tool Result 引用展示优化 - 实现计划

## 概述

将 AI 助手回复中的 tool_result 从 activities 区域移到文本末尾的引用样式，点击后弹出卡片显示详情。

**设计文档**: `docs/superpowers/specs/2026-03-19-tool-result-reference-display-design.md`

## 任务列表

### Phase 1: 后端数据结构变更

#### Task 1.1: 更新 PersistedActivity 类型

**文件**: `internal/ai/runtime/types.go`

**改动**:
```go
type PersistedActivity struct {
    ID         string         `json:"id"`
    Kind       string         `json:"kind"`
    Label      string         `json:"label"`
    Detail     string         `json:"detail,omitempty"`
    Status     string         `json:"status,omitempty"`
    StepIndex  int            `json:"stepIndex,omitempty"`
    Arguments  map[string]any `json:"arguments,omitempty"`  // 新增
    RawContent string         `json:"rawContent,omitempty"` // 新增
}
```

**验证**: `go build ./...`

---

#### Task 1.2: 更新 project.go 存储逻辑

**文件**: `internal/ai/runtime/project.go`

**改动点**:

1. `NormalizedKindToolCall` 分支（约第 107-131 行）：
   - 存储 `Arguments: event.Tool.Arguments`

2. `NormalizedKindToolResult` 分支（约第 132-173 行）：
   - 存储 `RawContent: event.Tool.Content`
   - 保持 `Detail` 截断到 200 字符

**验证**: `go test ./internal/ai/runtime/... -v`

---

### Phase 2: 前端类型变更

#### Task 2.1: 更新前端类型定义

**文件**: `web/src/components/AI/types.ts`

**改动**:
```typescript
export interface AssistantReplyActivity {
  id: string;
  kind: AssistantReplyActivityKind;
  label: string;
  detail?: string;
  status?: AssistantReplyActivityStatus;
  stepIndex?: number;
  arguments?: Record<string, unknown>;  // 新增
  rawContent?: string;                  // 新增
}
```

**验证**: `make web-test`

---

#### Task 2.2: 更新 replyRuntime.ts

**文件**: `web/src/components/AI/replyRuntime.ts`

**改动点**:

1. `applyToolCall` 函数（约第 145-161 行）：
   - 添加 `arguments: payload.arguments`

2. `applyToolResult` 函数（约第 271-288 行）：
   - 添加 `rawContent: payload.content`
   - 修复 `status` 传递：`payload.status === 'error' ? 'error' : 'done'`
   - `detail` 改为截断：`payload.content.slice(0, 200)`

**验证**: `make web-test`

---

### Phase 3: ToolReference 组件实现

#### Task 3.1: 创建 ToolReference 组件

**文件**: `web/src/components/AI/ToolReference.tsx`

**功能**:
- 接收 `activity: AssistantReplyActivity` prop
- 根据状态显示不同样式：
  - 执行中：`[◐ tool_name]` 灰色 + 动画
  - 完成：`[→ tool_name]` 蓝色 + 可点击
  - 错误：`[✗ tool_name]` 红色 + 可点击
- 点击后打开 ToolResultCard

**样式**:
- 使用 `antd-style` 的 `createStyles`
- 引用样式：等宽字体、小字号、圆角背景
- 加载动画：CSS keyframes

**验证**: 手动测试 + 单元测试

---

### Phase 4: ToolResultCard 组件实现

#### Task 4.1: 创建 ToolResultCard 组件

**文件**: `web/src/components/AI/ToolResultCard.tsx`

**功能**:
- 接收 `activity: AssistantReplyActivity` prop
- 根据 `rawContent` 行数选择 Popover 或 Modal
- 显示内容：工具名称、调用参数（可折叠）、执行结果

**子组件**:
- `ToolResultPopover`: 紧凑浮动卡片
- `ToolResultModal`: 居中弹窗

**辅助函数**:
```typescript
function getDisplayMode(content: string): 'popover' | 'modal'
function formatContent(content: string, maxSize?: number): string
```

**样式**:
- Popover: 最大宽度 360px，最大高度 300px
- Modal: 宽度 600px
- JSON 语法高亮：使用 `<pre>` + 等宽字体

**验证**: 手动测试各种内容长度

---

### Phase 5: AssistantReply 集成

#### Task 5.1: 修改 AssistantReplyContent 组件

**文件**: `web/src/components/AI/AssistantReply.tsx`

**改动点**:

1. 新增 `toolActivities` 过滤逻辑（约第 230 行附近）：
```typescript
const toolActivities = runtime?.activities?.filter(
  (a) => a.stepIndex === activeStepIndex &&
        (a.kind === 'tool_call' || a.kind === 'tool_result')
) || [];
```

2. 在 `activeStep.content` 渲染后追加 ToolReference 组件

3. 过滤 `activeStepActivities`，排除 `tool_call` 和 `tool_result`

**验证**: `make web-test` + 手动测试

---

### Phase 6: 测试与验证

#### Task 6.1: 更新单元测试

**文件**:
- `internal/ai/runtime/projector_test.go`
- `internal/ai/runtime/normalize_test.go`
- `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- `web/src/components/AI/replyRuntime.test.ts`

**测试点**:
- 后端存储 arguments 和 rawContent
- 前端 applyToolCall 存储 arguments
- 前端 applyToolResult 存储 rawContent 和正确 status
- ToolReference 组件各状态渲染
- AssistantReply 过滤 activities

---

#### Task 6.2: 集成测试

**测试场景**:
1. 新对话：工具调用完整流程
2. 历史对话：缺少 arguments/rawContent 字段时正常显示
3. 错误状态：工具执行失败时红色引用
4. 多工具调用：显示多个引用
5. 长内容：Modal 展示
6. 短内容：Popover 展示

---

## 实现顺序

```
Phase 1 (后端) ──→ Phase 2 (前端类型) ──→ Phase 3 (ToolReference) ──→ Phase 4 (ToolResultCard) ──→ Phase 5 (集成) ──→ Phase 6 (测试)
```

## 风险与缓解

| 风险 | 缓解措施 |
|------|---------|
| 历史对话显示异常 | 使用可选链访问新字段，缺失时降级处理 |
| 大型 rawContent 影响性能 | 前端截断显示，后端可后续添加大小限制 |
| Modal 嵌套问题 | 使用 `getContainer` 挂载到正确 DOM 节点 |

## 完成标准

- [ ] 后端单元测试通过
- [ ] 前端单元测试通过
- [ ] 新对话工具引用正常显示
- [ ] 历史对话不报错
- [ ] 错误状态红色显示
- [ ] Popover/Modal 自适应切换
