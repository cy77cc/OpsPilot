# Agent 原生输出的前端呈现设计方案

> 讨论日期：2026-03-15

## 一、背景

通过 `TestAgentRawOutput` 测试，我们观察到 ADK Agent 的原生事件输出：

```
Event #1 (planner):   {"steps": ["步骤1", "步骤2"]}
Event #2 (executor):  自然语言回答 / tool_calls
Event #3 (replanner): {"response": "..."} 或 BreakLoop
Event #4 (replanner): BreakLoop 表示完成
```

## 二、设计原则

### 2.1 Planner 输出保持结构化

- Planner 的 JSON 输出（`{"steps": [...]}`）是给 Executor 用的，保持不变
- 前端需要解析并美化展示，而非直接显示原始 JSON

### 2.2 流程链路

```
planner → executor → replan → planner → executor → ...
```

**关键点**：
- 流程是沿着链路**往下走**，不是"回到"第一个节点
- 每次循环产生**新的独立节点**，避免思维链从最后面跳回最前面
- 前端展示是**追加式**的，不是覆盖式

### 2.3 事件展示策略

| 事件类型 | 展示方式 | 说明 |
|---------|---------|------|
| planner JSON | 解析为步骤列表 | 美化展示，不显示原始 JSON |
| executor 回答 | 作为最终回答候选 | 可能是最终答案 |
| tool_calls | 显示工具调用卡片 | 包含工具名、参数 |
| tool response | 显示执行结果 | 可折叠查看详情 |
| replanner BreakLoop | 标记完成 | 不显示给用户 |
| replanner 调整 | 新增调整节点 | 显示调整原因 |

## 三、审批流程改进

### 3.1 目标

将"风险审批"和"工具调用审批"合并，在工具调用前统一审批，并支持参数修改。

### 3.2 参考：官方 ReviewEditInfo

```go
// 来自 eino-examples/adk/common/tool/review_edit_wrapper.go
type ReviewEditInfo struct {
    ToolName        string
    ArgumentsInJSON string
    ReviewResult    *ReviewEditResult
}

type ReviewEditResult struct {
    EditedArgumentsInJSON *string  // 修改后的参数
    NoNeedToEdit          bool     // 无需修改，直接执行
    Disapproved           bool     // 拒绝执行
    DisapproveReason      *string  // 拒绝原因
}
```

### 3.3 后端改造

#### 扩展 ApprovalInterruptInfo

```go
type ApprovalInterruptInfo struct {
    ToolName        string         `json:"tool_name"`
    ToolDisplayName string         `json:"tool_display_name"`
    Mode            string         `json:"mode"`
    RiskLevel       string         `json:"risk_level"`
    Summary         string         `json:"summary"`
    Params          map[string]any `json:"params"`
    ArgumentsInJSON string         `json:"arguments_json"`       // 新增：原始参数 JSON
    ToolSchema      *ToolSchema    `json:"tool_schema,omitempty"` // 新增：工具参数 schema
}

type ToolSchema struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Parameters  map[string]ParamSchema `json:"parameters"`
}

type ParamSchema struct {
    Type        string `json:"type"`
    Description string `json:"description"`
    Required    bool   `json:"required"`
}
```

#### 扩展 ResumeRequest

```go
type ResumeRequest struct {
    SessionID    string  `json:"session_id"`
    PlanID       string  `json:"plan_id"`
    StepID       string  `json:"step_id"`
    CheckpointID string  `json:"checkpoint_id"`
    Approved     bool    `json:"approved"`
    Reason       string  `json:"reason,omitempty"`

    // 新增：修改后的参数
    EditedArguments string `json:"edited_arguments,omitempty"`
}
```

### 3.4 前端改造

#### 扩展 ConfirmationRequest

```tsx
interface ConfirmationRequest {
  id: string;
  title: string;
  description: string;
  risk: RiskLevel;
  status?: 'waiting_user' | 'submitting' | 'failed';

  // 工具信息
  toolName: string;
  argumentsJson: string;
  toolSchema?: ToolSchema;

  // 支持编辑
  editable?: boolean;
  onConfirm: (editedArgs?: string) => void;
  onCancel: (reason?: string) => void;
}
```

#### 审批面板增强

```
┌─────────────────────────────────────────────┐
│ ⚠️ 执行前确认                                │
│   删除主机                                   │
│   该步骤会在确认后继续执行                    │
│                                             │
│ 操作详情：                                   │
│   主机ID: 7                                  │
│   主机名: test-server                        │
│                                             │
│ [查看/编辑参数] ← 点击展开 JSON 编辑器        │
│   {                                         │
│     "id": 7,                                │
│     "force": false                          │
│   }                                         │
│                                             │
│ 风险等级: 高风险                             │
│                                             │
│ [确认执行]  [取消]                          │
└─────────────────────────────────────────────┘
```

## 四、前端展示优化

### 4.1 Planner 输出解析

```tsx
function parsePlannerOutput(content: string): PlanStep[] | null {
  try {
    const parsed = JSON.parse(content);
    if (parsed.steps && Array.isArray(parsed.steps)) {
      return parsed.steps.map((step, i) => ({
        id: `step-${i}`,
        title: typeof step === 'string' ? step : step.title || step.content,
        status: 'pending'
      }));
    }
  } catch {}
  return null;
}
```

### 4.2 思维链节点结构

```tsx
interface RuntimeThoughtChainNode {
  nodeId: string;          // 唯一ID，每次循环生成新的
  kind: 'plan' | 'execute' | 'tool' | 'replan' | 'approval';
  title: string;
  status: 'pending' | 'active' | 'done' | 'error' | 'waiting';
  headline?: string;
  body?: string;
  structured?: Record<string, unknown>;
  // ...
}
```

### 4.3 追加式更新示例

```
第一次循环：
  node-1: plan (整理执行步骤)
  node-2: execute (执行步骤)
  node-3: tool:xxx (工具调用)

第二次循环（replan 触发）：
  node-4: replan (发现新信息，调整计划)  ← 新节点，不更新 node-1
  node-5: plan (新的计划)               ← 新节点
  node-6: execute (执行新步骤)           ← 新节点
```

## 五、实施计划

### Phase 1: 后端审批增强
- [ ] 扩展 `ApprovalInterruptInfo` 增加 `ArgumentsInJSON`
- [ ] 扩展 `ResumeRequest` 支持 `EditedArguments`
- [ ] 修改 `Gate` 支持参数修改后重新执行

### Phase 2: 前端参数编辑
- [ ] 扩展 `ConfirmationRequest` 类型
- [ ] 在 `ConfirmationPanel` 中添加 JSON 编辑器
- [ ] 支持参数修改后提交

### Phase 3: Planner 输出美化
- [ ] 前端解析 planner JSON
- [ ] 展示为结构化步骤列表
- [ ] 过滤原始 JSON 显示

### Phase 4: 思维链优化
- [ ] 确保追加式更新，不覆盖之前节点
- [ ] 优化 replan 节点展示
- [ ] 处理 BreakLoop 事件（标记完成，不显示）

---

## 六、相关文件

- 后端审批：`internal/ai/tools/approval/gate.go`
- 后端事件处理：`internal/ai/orchestrator.go`
- 前端类型：`web/src/components/AI/types.ts`
- 前端审批面板：`web/src/components/AI/components/ConfirmationPanel.tsx`
- 前端思维链：`web/src/components/AI/components/RuntimeThoughtChain.tsx`
- 官方参考：`/root/learn/eino-examples/adk/human-in-the-loop/6_plan-execute-replan/`
- 官方参考：`/root/learn/eino-examples/adk/common/tool/review_edit_wrapper.go`
